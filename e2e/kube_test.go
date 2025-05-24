package e2e

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/go-logr/logr"
	etcdv3 "go.etcd.io/etcd/client/v3"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubescheme "k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client"
	clientconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/agoda-com/etcd-operator/pkg/etcd"
)

var kubecontext = flag.String("context", "", "Kubeconfig context")
var namespace = flag.String("namespace", os.Getenv("POD_NAMESPACE"), "Namespace scope for requests")
var backupSecretName = flag.String("backup-secret-name", "etcd-backup", "Backup secret name")

var resources = corev1.ResourceList{
	corev1.ResourceCPU:     resource.MustParse("250m"),
	corev1.ResourceMemory:  resource.MustParse("256M"),
	corev1.ResourceStorage: resource.MustParse("256M"),
}

func kubeClient(t testing.TB, options client.Options) client.Client {
	t.Helper()

	if *namespace == "" {
		t.SkipNow()
	}

	log.SetLogger(logr.Discard())

	builder := runtime.NewSchemeBuilder(
		kubescheme.AddToScheme,
		apiv1.AddToScheme,
		cmv1.AddToScheme,
	)

	if options.Scheme == nil {
		options.Scheme = runtime.NewScheme()
		err := builder.AddToScheme(options.Scheme)
		if err != nil {
			t.Fatal("scheme:", err)
		}
	}

	config, err := clientconfig.GetConfigWithContext(*kubecontext)
	if err != nil {
		t.Fatal("load kubeconfig:", err)
	}

	cl, err := client.New(config, options)
	if err != nil {
		t.Fatal("k8s client:", err)
	}

	return cl
}

func etcdClient(t testing.TB, kcl client.Client, cluster *apiv1.EtcdCluster) *etcdv3.Client {
	t.Helper()

	key := client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      cluster.Status.SecretName,
	}
	tlsConfig, err := etcd.TLSConfig(etcd.LoadSecret(t.Context(), kcl, key))
	if err != nil {
		t.Fatal("load credentials:", err)
	}

	cl, err := etcd.Connect(t.Context(), tlsConfig, cluster.Status.Endpoint)
	if err != nil {
		t.Fatal("etcd client:", err)
	}
	t.Cleanup(func() {
		_ = cl.Close()
	})

	return cl
}

func createCluster(t testing.TB, kcl client.Client, timeout time.Duration, spec apiv1.EtcdClusterSpec) *apiv1.EtcdCluster {
	t.Helper()

	cluster := &apiv1.EtcdCluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "etcd-test-",
			Namespace:    *namespace,
		},
		Spec: spec,
	}

	// set pod as owner so etcdcluster get garbage collected after pod termination
	podName := os.Getenv("POD_NAME")
	if podName != "" {
		key := client.ObjectKey{
			Namespace: *namespace,
			Name:      podName,
		}
		pod := &corev1.Pod{}
		err := kcl.Get(t.Context(), key, pod)
		if err != nil {
			t.Fatal("get pod:", err)
		}

		err = ctrlutil.SetOwnerReference(pod, cluster, kcl.Scheme())
		if err != nil {
			t.Fatal("set owner reference:", err)
		}
	}

	err := kcl.Create(t.Context(), cluster)
	if err != nil {
		t.Fatal("create cluster:", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := kcl.Delete(ctx, cluster)
		if err != nil {
			t.Error("delete cluster:", err)
		}
	})

	key := client.ObjectKeyFromObject(cluster)
	t.Logf("cluster %q created", key)

	Poll(t, kcl, cluster, timeout, Available)

	t.Logf("cluster %q available", key)

	return cluster
}

func Poll[T client.Object](t testing.TB, kcl client.Client, obj T, timeout time.Duration, f func(obj T) bool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	t.Cleanup(cancel)

	err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, timeout, true, func(ctx context.Context) (bool, error) {
		err := kcl.Get(ctx, client.ObjectKeyFromObject(obj), obj)
		if err != nil {
			return false, err
		}

		return f(obj), nil
	})
	if err != nil {
		t.Fatal("poll:", err)
	}
}

func Available(cluster *apiv1.EtcdCluster) bool {
	return cluster.Status.AvailableReplicas == cluster.Spec.Replicas
}

func Scale(t testing.TB, kcl client.Client, obj client.Object, replicas int32) {
	scale := &autoscalingv1.Scale{}
	err := kcl.SubResource("scale").Get(t.Context(), obj, scale)
	if err != nil {
		t.Fatal("get scale:", err)
	}

	scale.Spec.Replicas = replicas
	err = kcl.SubResource("scale").Update(t.Context(), obj, client.WithSubResourceBody(scale))
	if err != nil {
		t.Fatal("update scale:", err)
	}
}

func triggerCronJob(t testing.TB, kcl client.Client, key client.ObjectKey, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	t.Cleanup(cancel)

	cronjob := &batchv1.CronJob{}
	err := kcl.Get(ctx, key, cronjob)
	if err != nil {
		t.Fatal("get backup cronjob:", err)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    cronjob.Namespace,
			GenerateName: cronjob.Name + "-",
			Annotations: map[string]string{
				"cronjob.kubernetes.io/instantiate": "manual",
			},
		},
		Spec: cronjob.Spec.JobTemplate.Spec,
	}
	err = ctrlutil.SetControllerReference(cronjob, job, kcl.Scheme())
	if err != nil {
		t.Fatal(err)
	}

	err = kcl.Create(ctx, job)
	if err != nil {
		t.Fatal("create job:", err)
	}

	t.Logf("created job %q", client.ObjectKeyFromObject(job))

	err = wait.PollUntilContextCancel(ctx, 500*time.Millisecond, true, func(ctx context.Context) (done bool, err error) {
		err = kcl.Get(ctx, client.ObjectKeyFromObject(job), job)
		switch {
		case apierrors.IsTooManyRequests(err):
			return false, err
		case err != nil:
			return true, fmt.Errorf("get backup job: %w", err)
		}

		for _, cond := range job.Status.Conditions {
			if cond.Status != corev1.ConditionTrue {
				continue
			}

			switch cond.Type {
			case batchv1.JobFailed:
				return true, errors.New(cond.Message)
			case batchv1.JobComplete:
				return true, nil
			}
		}

		return false, nil
	})
	if err != nil {
		t.Fatalf("job %q failed:", err)
	}

	t.Logf("backup job %q completed", client.ObjectKeyFromObject(job))
}
