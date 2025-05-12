package sidecar

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"slices"
	"time"

	"golang.org/x/sync/errgroup"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/agoda-com/etcd-operator/pkg/etcd"
)

type Config struct {
	client.ObjectKey

	BaseConfigFile  string
	ConfigFile      string
	Endpoint        string
	HealthAddress   string
	Interval        time.Duration
	Timeout         time.Duration
	ShutdownTimeout time.Duration

	Prune bool
}

type Sidecar struct {
	kcl        client.Client
	kubeconfig *rest.Config
	config     Config

	pod        corev1.Pod
	tlsConfig  tls.Config
	etcdConfig etcd.Config
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;patch
//+kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests,verbs=get;create;delete

func New(kcl client.Client, kubeconfig *rest.Config, config Config) *Sidecar {
	return &Sidecar{
		kcl:        kcl,
		kubeconfig: kubeconfig,
		config:     config,
	}
}

// Start implements manager.Runnable
func (s *Sidecar) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)

	pod := &s.pod
	err := s.kcl.Get(ctx, s.config.ObjectKey, pod)
	if err != nil {
		return err
	}

	// wait for pod ip
	err = Poll(ctx, s.kcl, pod, s.config.Interval, func(pod *corev1.Pod) (bool, error) {
		if s.pod.Status.PodIP != "" {
			return true, nil
		}

		logger.Info("waiting for pod IP")

		return false, nil
	})
	if err != nil {
		return err
	}

	errg, ctx := errgroup.WithContext(ctx)
	errg.Go(func() error {
		return s.Health(ctx)
	})

	// configure the member
	err = s.Configure(ctx)
	if err != nil {
		return fmt.Errorf("configure: %v", err)
	}

	go func() {
		// remove member when terminating using separate context
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
		defer cancel()

		err = s.Remove(ctx)
		if err != nil {
			logger.Error(err, "remove member")
		}
	}()

	// wait until etcd container is ready
	err = Poll(ctx, s.kcl, pod, s.config.Interval, func(pod *corev1.Pod) (bool, error) {
		ready := slices.ContainsFunc(s.pod.Status.ContainerStatuses, func(cs corev1.ContainerStatus) bool {
			return cs.Name == "etcd" && cs.Ready
		})
		return ready, nil
	})
	if err != nil {
		return fmt.Errorf("etcd not ready: %v", err)
	}

	errg.Go(func() error {
		return s.WatchCluster(ctx)
	})

	errg.Go(func() error {
		wait.UntilWithContext(ctx, func(ctx context.Context) {
			err := s.GenerateCredentials(ctx)
			if err != nil {
				logger.Error(err, "generate credentials")
				return
			}
		}, s.config.Interval)

		return nil
	})

	return errg.Wait()
}

func (s *Sidecar) Health(ctx context.Context) error {
	if s.config.HealthAddress == "" {
		return nil
	}

	logger := log.FromContext(ctx).WithName("health").WithValues("addr", s.config.HealthAddress)

	mux := http.NewServeMux()
	mux.Handle("GET /healthz", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, err := os.Stat(s.config.ConfigFile)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			logger.V(3).Info("config file does not exist yet")
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		case err != nil:
			logger.Error(err, "config")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		_, _ = fmt.Fprintln(w, "ok")
	}))

	server := &http.Server{
		Addr:    s.config.HealthAddress,
		Handler: mux,
	}
	defer func() {
		<-ctx.Done()

		logger.Info("shutdown")

		ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
		defer cancel()

		err := server.Shutdown(ctx)
		if err != nil {
			logger.Error(err, "shutdown")
			return
		}
	}()

	logger.Info("listen")

	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error(err, "serve")
		return err
	}

	return nil
}

func (s *Sidecar) WatchCluster(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("watch-cluster")

	ecl, err := etcd.Connect(ctx, &s.tlsConfig, s.config.Endpoint)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, ecl.Close())
	}()

	wait.UntilWithContext(ctx, func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		err := s.Sync(ctx, ecl)
		if err != nil {
			logger.Error(err, "sync")
		}
	}, s.config.Interval)

	return nil
}

func (s *Sidecar) Remove(ctx context.Context) error {
	pod := &s.pod
	err := s.kcl.Get(ctx, s.config.ObjectKey, pod)
	if err != nil {
		return err
	}

	// bail if pod is not deleted
	if s.pod.DeletionTimestamp.IsZero() {
		return nil
	}

	id := apiv1.ParseMemberID(s.pod.Labels)
	if id == 0 {
		return nil
	}

	logger := log.FromContext(ctx, "id", s.pod.Labels[apiv1.MemberIDLabel]).WithName("remove")

	ecl, err := etcd.Connect(ctx, &s.tlsConfig, s.config.Endpoint)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, ecl.Close())
	}()

	err = wait.PollUntilContextCancel(ctx, s.config.Interval, true, func(ctx context.Context) (bool, error) {
		ctx, cancel := context.WithTimeout(ctx, s.config.Interval)
		defer cancel()

		_, err = ecl.MemberRemove(ctx, id)
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			logger.Info("remove member: timeout")
			return false, err
		case errors.Is(err, rpctypes.ErrUnhealthy):
			logger.Info("remove member: waiting for cluster to be healthy")
			return false, err
		case errors.Is(err, rpctypes.ErrMemberNotFound):
			return true, nil
		default:
			return true, err
		}
	})
	if err != nil {
		return err
	}

	logger.Info("removed")

	return nil
}
