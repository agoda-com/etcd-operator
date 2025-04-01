package metrics

import (
	"context"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"go.opentelemetry.io/otel/metric/noop"
)

// NoopTelemetryProvider is a no-operation implementation of TelemetryProvider
// that uses the official OpenTelemetry noop implementation internally
type NoopTelemetryProvider struct {
	mp             noop.MeterProvider
	clusterMetrics *ClusterMetricsManager
}

// NewNoopTelemetryProvider creates a new no-operation telemetry provider
// using the official OpenTelemetry noop implementation
func NewNoopTelemetryProvider() TelemetryProvider {
	mp := noop.NewMeterProvider()
	meter := mp.Meter("noop")

	return &NoopTelemetryProvider{
		mp:             mp,
		clusterMetrics: NewClusterMetricsManager(meter, "noop"),
	}
}

// GetClusterMetrics returns a no-operation cluster metrics implementation
func (p *NoopTelemetryProvider) GetClusterMetrics() *ClusterMetricsManager {
	return p.clusterMetrics
}

// Shutdown is a no-op
func (p *NoopTelemetryProvider) Shutdown(ctx context.Context) error {
	return nil
}

// NoopClusterMetrics is a no-operation implementation of ClusterMetrics
type NoopClusterMetrics struct{}

// Record is a no-op
func (m *NoopClusterMetrics) Record(ctx context.Context, cluster *apiv1.EtcdCluster) {
	// No-op
}
