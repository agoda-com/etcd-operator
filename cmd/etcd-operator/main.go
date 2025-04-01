/*
Copyright 2021 wferguson.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/go-logr/logr"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"

	"github.com/agoda-com/etcd-operator/pkg/cluster"
	"github.com/agoda-com/etcd-operator/pkg/etcd"
	"github.com/agoda-com/etcd-operator/pkg/metrics"
)

func main() {
	cmd := Command()
	err := cmd.ExecuteContext(signals.SetupSignalHandler())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, cmd.UsageString())

		os.Exit(1)
	}
}

func run(ctx context.Context, logger logr.Logger, kubeconfig *rest.Config, config Config) error {
	SetupCoverage(ctx, logger, syscall.SIGUSR1)

	// Initialize telemetry
	telemetry := setupTelemetry(ctx, logger)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetry.Shutdown(shutdownCtx); err != nil {
			logger.Error(err, "Failed to shutdown telemetry provider")
		}
	}()

	builder := runtime.NewSchemeBuilder(
		kscheme.AddToScheme,
		apiv1.AddToScheme,
		cmv1.AddToScheme,
	)

	scheme := runtime.NewScheme()
	err := builder.AddToScheme(scheme)
	if err != nil {
		return fmt.Errorf("scheme: %w", err)
	}

	namespaces := make(map[string]cache.Config, len(config.WatchNamespaces))
	for _, ns := range config.WatchNamespaces {
		namespaces[ns] = cache.Config{
			LabelSelector: config.WatchSelector.Selector(),
		}
	}

	mgr, err := manager.New(kubeconfig, manager.Options{
		Logger: logger,
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: config.MetricsAddr,
		},
		HealthProbeBindAddress: config.HealthProbeAddr,
		LeaderElection:         config.LeaderElection,
		LeaderElectionID:       "3c04a122.fleet.agoda.com",
		Cache: cache.Options{
			DefaultNamespaces: namespaces,
		},
	})
	if err != nil {
		return fmt.Errorf("manager: %w", err)
	}

	tlsCache, err := etcd.NewTLSCache(mgr.GetClient(), 1024)
	if err != nil {
		return fmt.Errorf("tls cache: %w", err)
	}

	err = cluster.CreateControllerWithManager(mgr, tlsCache, cluster.Config{
		Image:           config.Image,
		ControllerImage: config.ControllerImage,
	}, telemetry)
	if err != nil {
		return fmt.Errorf("cluster controller: %w", err)
	}

	err = mgr.AddHealthzCheck("ping", healthz.Ping)
	if err != nil {
		return err
	}

	err = mgr.AddReadyzCheck("ping", healthz.Ping)
	if err != nil {
		return err
	}

	return mgr.Start(ctx)
}

func setupTelemetry(ctx context.Context, logger logr.Logger) metrics.TelemetryProvider {
	telemetryConfig := metrics.ConfigFromEnv()

	// Initialize telemetry provider
	telemetry, err := metrics.NewTelemetryProvider(ctx, telemetryConfig)
	if err != nil {
		logger.Error(err, "Failed to initialize telemetry provider, using no-op implementation")
		return metrics.NewNoopTelemetryProvider()
	}

	logger.Info("Telemetry provider initialized successfully",
		"endpoint", telemetryConfig.OTLPEndpoint,
		"serviceName", telemetryConfig.ServiceName,
		"insecure", telemetryConfig.Insecure)

	return telemetry
}
