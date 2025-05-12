package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/runtime"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	"sigs.k8s.io/controller-runtime/pkg/client"
	clientconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/agoda-com/etcd-operator/pkg/sidecar"
)

const (
	DefaultTimeout  = 30 * time.Second
	DefaultInterval = 5 * time.Second
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Short:         "ETCD member lifecycle sidecar",
		Use:           "etcd-sidecar",
		SilenceErrors: true,
	}

	config := sidecar.Config{
		ObjectKey: client.ObjectKey{
			Namespace: os.Getenv("POD_NAMESPACE"),
			Name:      os.Getenv("POD_NAME"),
		},
	}

	flags := cmd.Flags()
	kubecontext := flags.String(clientcmd.FlagContext, "", "name of the kubeconfig context to use.")
	flags.StringVar(&config.BaseConfigFile, "base-config", "", "base etcd cluster config file.")
	flags.StringVar(&config.ConfigFile, "config", "", "output path for generated config file.")
	flags.StringVar(&config.Endpoint, "endpoint", "https://127.0.0.1:2379", "etcd cluster endpoint.")
	flags.StringVar(&config.HealthAddress, "health-address", "", "The address the health endpoint binds to.")
	flags.DurationVar(&config.Interval, "interval", DefaultInterval, "operation retry interval.")
	flags.DurationVar(&config.Timeout, "timeout", DefaultTimeout, "operation timeout.")
	flags.DurationVar(&config.ShutdownTimeout, "shutdown-timeout", DefaultTimeout, "shutdown timeout.")
	flags.BoolVar(&config.Prune, "prune", true, "prune members without pods.")

	_ = cmd.MarkFlagRequired("base-config")
	_ = cmd.MarkFlagRequired("config")

	zflags := &zap.Options{}
	gflags := &flag.FlagSet{}
	zflags.BindFlags(gflags)
	flags.AddGoFlagSet(gflags)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if config.Name == "" || config.Namespace == "" {
			return errors.New("POD_NAMESPACE and POD_NAME are required")
		}

		logger := zap.New(zap.UseFlagOptions(zflags))
		log.SetLogger(logger)
		ctx := log.IntoContext(cmd.Context(), logger)

		kubeconfig, err := clientconfig.GetConfigWithContext(*kubecontext)
		if err != nil {
			return err
		}

		builder := runtime.NewSchemeBuilder(
			kscheme.AddToScheme,
			cmv1.AddToScheme,
		)
		scheme := runtime.NewScheme()
		if err := builder.AddToScheme(scheme); err != nil {
			return err
		}

		kcl, err := client.New(kubeconfig, client.Options{
			Scheme: scheme,
		})
		if err != nil {
			return fmt.Errorf("k8s client: %w", err)
		}

		sc := sidecar.New(kcl, kubeconfig, config)
		err = sc.Start(ctx)
		if err != nil {
			logger.Error(err, "sidecar")
		}

		return nil
	}

	return cmd
}
