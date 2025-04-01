package metrics

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/agoda-com/etcd-operator/pkg/conditions"
)

const (
	ClusterMetricsNamespace = "cluster"

	ClusterNamespaceLabel = "namespace"
	ClusterNameLabel      = "name"

	ClusterMetricAvailable                = "available"
	ClusterMetricAvailableReplicas        = "available_replicas"
	ClusterMetricDesiredReplicas          = "desired_replicas"
	ClusterMetricUpdatedReplicas          = "updated_replicas"
	ClusterMetricLearnerReplicas          = "learner_replicas"
	ClusterMetricBackupEnabled            = "backup_enabled"
	ClusterMetricBackupLastScheduleTime   = "backup_last_schedule_time"
	ClusterMetricBackupLastSuccessfulTime = "backup_last_successful_time"
)

// ClusterMetrics provides methods for recording individual ETCD cluster metrics
type ClusterMetrics interface {
	// Record records all metrics for a given EtcdCluster
	Record(ctx context.Context, cluster *apiv1.EtcdCluster)
}

type clusterMetrics struct {
	meter        metric.Meter
	registration metric.Registration
	values       sync.Map // maps string (metric name) to int64
	key          client.ObjectKey
	namespace    string
}

// ClusterMetricsManager manages per-cluster metrics
type ClusterMetricsManager struct {
	sync.Map  // maps client.ObjectKey to *clusterMetrics
	meter     metric.Meter
	namespace string
}

// NewClusterMetricsManager creates a new ClusterMetricsManager
func NewClusterMetricsManager(meter metric.Meter, namespace string) *ClusterMetricsManager {
	return &ClusterMetricsManager{
		meter:     meter,
		namespace: namespace,
	}
}

// GetOrCreateMetrics returns existing metrics for a cluster or creates new ones
func (m *ClusterMetricsManager) GetOrCreateMetrics(key client.ObjectKey) (ClusterMetrics, error) {
	if metrics, ok := m.Load(key); ok {
		return metrics.(ClusterMetrics), nil
	}

	metrics, err := newClusterMetrics(m.meter, m.namespace, key)
	if err != nil {
		return nil, fmt.Errorf("create cluster metrics: %w", err)
	}

	actual, loaded := m.LoadOrStore(key, metrics)
	if loaded {
		// Another goroutine created metrics before us, use those instead
		return actual.(ClusterMetrics), nil
	}

	return metrics, nil
}

// DeleteMetrics removes metrics for a cluster
func (m *ClusterMetricsManager) DeleteMetrics(ctx context.Context, key client.ObjectKey) error {
	metrics, ok := m.LoadAndDelete(key)
	if !ok {
		// No metrics found for this key, which is fine
		return nil
	}

	clusterMetrics, ok := metrics.(*clusterMetrics)
	if !ok {
		return fmt.Errorf("invalid metrics type for key %v", key)
	}

	// Unregister the callback to stop exporting metrics
	if err := clusterMetrics.registration.Unregister(); err != nil {
		return fmt.Errorf("failed to unregister metrics for cluster %v: %w", key, err)
	}

	return nil
}

// newClusterMetrics creates and initializes cluster metrics
func newClusterMetrics(meter metric.Meter, namespace string, key client.ObjectKey) (*clusterMetrics, error) {
	m := &clusterMetrics{
		meter:     meter,
		key:       key,
		namespace: namespace,
	}

	// Define all our observable gauges
	gauges := []struct {
		name        string
		description string
	}{
		{
			name:        ClusterMetricAvailable,
			description: "Whether the cluster is available",
		},
		{
			name:        ClusterMetricAvailableReplicas,
			description: "Number of available replicas",
		},
		{
			name:        ClusterMetricDesiredReplicas,
			description: "Number of desired replicas",
		},
		{
			name:        ClusterMetricUpdatedReplicas,
			description: "Number of updated replicas",
		},
		{
			name:        ClusterMetricLearnerReplicas,
			description: "Number of learner replicas",
		},
		{
			name:        ClusterMetricBackupEnabled,
			description: "Whether backup is enabled",
		},
		{
			name:        ClusterMetricBackupLastScheduleTime,
			description: "Last backup schedule time",
		},
		{
			name:        ClusterMetricBackupLastSuccessfulTime,
			description: "Last backup successful time",
		},
	}

	// Create all the observable gauges and store name mapping
	observables := make([]metric.Observable, 0, len(gauges))
	nameToObservable := make(map[string]metric.Int64Observable)

	for _, g := range gauges {
		metricName := fmt.Sprintf("%s.%s.%s", m.namespace, ClusterMetricsNamespace, g.name)
		observable, err := meter.Int64ObservableGauge(
			metricName,
			metric.WithDescription(g.description),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create gauge %s: %w", g.name, err)
		}
		observables = append(observables, observable)
		nameToObservable[g.name] = observable
	}

	// Register a single callback for all gauges
	reg, err := meter.RegisterCallback(
		func(_ context.Context, o metric.Observer) error {
			m.values.Range(func(key, value any) bool {
				name := key.(string)
				val := value.(int64)
				if observable, ok := nameToObservable[name]; ok {
					o.ObserveInt64(observable, val,
						metric.WithAttributes(
							attribute.String(ClusterNamespaceLabel, m.key.Namespace),
							attribute.String(ClusterNameLabel, m.key.Name),
						))
				}
				return true
			})
			return nil
		},
		observables...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register callback: %w", err)
	}

	m.registration = reg
	return m, nil
}

// Record records all metrics for a given EtcdCluster
func (m *clusterMetrics) Record(ctx context.Context, cluster *apiv1.EtcdCluster) {
	if cluster == nil {
		return
	}

	spec, status := cluster.Spec, cluster.Status

	// Record availability
	if available, ok := conditions.Get(status.Conditions, apiv1.ClusterAvailable); ok {
		value := int64(0)
		if available.Status == corev1.ConditionTrue {
			value = int64(1)
		}
		m.values.Store(ClusterMetricAvailable, value)
	}

	// Record replica counts
	m.values.Store(ClusterMetricDesiredReplicas, int64(spec.Replicas))
	m.values.Store(ClusterMetricAvailableReplicas, int64(status.AvailableReplicas))
	m.values.Store(ClusterMetricUpdatedReplicas, int64(status.UpdatedReplicas))
	m.values.Store(ClusterMetricLearnerReplicas, int64(status.LearnerReplicas))

	// Record backup metrics
	value := int64(0)
	if status.Backup.Enabled {
		value = int64(1)
	}
	m.values.Store(ClusterMetricBackupEnabled, value)

	if !status.Backup.LastScheduleTime.IsZero() {
		m.values.Store(ClusterMetricBackupLastScheduleTime, status.Backup.LastScheduleTime.Unix())
	}
	if !status.Backup.LastSuccessfulTime.IsZero() {
		m.values.Store(ClusterMetricBackupLastSuccessfulTime, status.Backup.LastSuccessfulTime.Unix())
	}
}
