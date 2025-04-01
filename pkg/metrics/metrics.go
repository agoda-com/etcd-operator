package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// TelemetryProvider handles the lifecycle of telemetry and provides access to metric recorders
type TelemetryProvider interface {
	// GetClusterMetrics returns the cluster metrics manager
	GetClusterMetrics() *ClusterMetricsManager

	// Shutdown gracefully shuts down the telemetry provider
	Shutdown(ctx context.Context) error
}

// provider implements the TelemetryProvider interface
type provider struct {
	mp             *metric.MeterProvider
	clusterMetrics *ClusterMetricsManager
}

// NewTelemetryProvider initializes a new OTEL telemetry provider
func NewTelemetryProvider(ctx context.Context, config Config) (TelemetryProvider, error) {
	// Configure OTLP exporter options
	var opts []otlpmetricgrpc.Option

	if config.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	// Add endpoint option
	opts = append(opts, otlpmetricgrpc.WithEndpoint(config.OTLPEndpoint))

	// Create OTLP exporter
	exporter, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(config.ServiceName),
	)

	// Create meter provider
	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(
			metric.NewPeriodicReader(
				exporter,
				metric.WithInterval(config.ExportInterval),
			),
		),
	)

	// Set global meter provider
	otel.SetMeterProvider(mp)

	// Create meter
	meter := mp.Meter(config.ServiceName)

	// Initialize cluster metrics manager
	clusterMetrics := NewClusterMetricsManager(meter, config.Namespace)

	return &provider{
		mp:             mp,
		clusterMetrics: clusterMetrics,
	}, nil
}

// GetClusterMetrics returns the cluster metrics manager
func (p *provider) GetClusterMetrics() *ClusterMetricsManager {
	return p.clusterMetrics
}

// Shutdown gracefully shuts down the telemetry provider
func (p *provider) Shutdown(ctx context.Context) error {
	return p.mp.Shutdown(ctx)
}
