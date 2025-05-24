package main

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func SetupTelemetry(ctx context.Context) (metric.MeterProvider, error) {
	logger := log.FromContext(ctx).WithName("metrics")

	endpoint := os.Getenv("OTEL_EXPORTER_OLTP_ENDPOINT")
	if endpoint != "" {
		exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpoint(endpoint))
		if err != nil {
			return nil, err
		}

		provider := metricsdk.NewMeterProvider(
			metricsdk.WithReader(metricsdk.NewPeriodicReader(exporter)),
		)

		logger.Info("enabled otlp grpc metrics", "endpoint", endpoint)

		go func() {
			<-ctx.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := provider.Shutdown(ctx)
			if err != nil {
				logger.Error(err, "shutdown metrics")
			}
		}()

		return provider, nil
	}

	// fallback on noop provider if endpoint is not configured
	return noop.NewMeterProvider(), nil
}
