package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/agoda-com/etcd-operator/pkg/backup"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/controller-runtime/pkg/client"
	clientconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	DefaultImage           = "quay.io/coreos/etcd"
	DefaultControllerImage = "ghcr.io/agoda-com/etcd-operator"
)

type Config struct {
	Pod               client.ObjectKey
	MetricsAddr       string
	HealthProbeAddr   string
	LeaderElection    bool
	WatchNamespaces   []string
	WatchSelector     LabelSelector
	Image             string
	ControllerImage   string
	PriorityClassName string
	BackupEnv         map[string]string
}

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Short: "ETCD operator",
		Use:   "etcd-operator",
	}

	flags := cmd.Flags()
	kubecontext := flags.String(clientcmd.FlagContext, "", "Kubernetes context to use.")

	config := Config{
		Pod: client.ObjectKey{
			Namespace: os.Getenv("POD_NAMESPACE"),
			Name:      os.Getenv("POD_NAME"),
		},
		BackupEnv: backup.LoadEnv(),
	}
	flags.StringSliceVar(&config.WatchNamespaces, "watch-namespaces", nil, "Namespaces to watch for resources.")
	flags.Var(&config.WatchSelector, "watch-selector", "Selector to watch for resources.")
	flags.StringVar(&config.PriorityClassName, "priority-class-name", "", "ETCD cluster pods priorityClassName")

	flags.StringVar(&config.MetricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flags.StringVar(&config.HealthProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flags.BoolVar(&config.LeaderElection, "leader-elect", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	stdFlags := flag.NewFlagSet("etcd-operator", flag.ContinueOnError)
	zapOptions := &zap.Options{}
	zapOptions.BindFlags(stdFlags)
	flags.AddGoFlagSet(stdFlags)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger := zap.New(zap.UseFlagOptions(zapOptions))
		log.SetLogger(logger)
		ctx := log.IntoContext(cmd.Context(), logger)

		if config.Image == "" {
			config.Image = DefaultImage
		}

		kubeconfig, err := clientconfig.GetConfigWithContext(*kubecontext)
		if err != nil {
			return fmt.Errorf("kubeconfig: %w", err)
		}

		kcl, err := client.New(kubeconfig, client.Options{DryRun: ptr.To(true)})
		if err != nil {
			return err
		}

		if config.Pod.Name != "" && config.Pod.Namespace != "" {
			pod := &corev1.Pod{}
			err = kcl.Get(ctx, config.Pod, pod)
			if err != nil {
				return fmt.Errorf("get pod: %w", err)
			}

			for _, container := range pod.Spec.Containers {
				if container.Name == "operator" {
					config.ControllerImage = container.Image
				}
			}
		}

		if config.ControllerImage == "" {
			config.ControllerImage = DefaultControllerImage
		}

		if config.WatchNamespaces == nil {
			config.WatchNamespaces = []string{config.Pod.Namespace}
		}

		return run(ctx, logger, kubeconfig, config)
	}

	return cmd
}

type LabelSelector struct {
	delegate labels.Selector
}

func (s LabelSelector) Selector() labels.Selector {
	if s.delegate == nil {
		return labels.Everything()
	}

	return s.delegate
}

func (s *LabelSelector) Set(data string) error {
	selector, err := labels.Parse(data)
	if err != nil {
		return err
	}

	s.delegate = selector
	return nil
}

func (s *LabelSelector) String() string {
	return s.Selector().String()
}

func (s *LabelSelector) Type() string {
	return "string"
}
