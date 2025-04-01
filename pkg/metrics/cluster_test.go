package metrics_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/agoda-com/etcd-operator/pkg/metrics"
)

func TestClusterMetrics(t *testing.T) {
	// Create a test context and config
	ctx := context.Background()
	config := metrics.Config{
		OTLPEndpoint:   "localhost:4317",
		ServiceName:    "test-service",
		Namespace:      "test.namespace",
		ExportInterval: 1 * time.Second,
		Insecure:       true,
	}

	// Create a provider with a short timeout to avoid hanging tests
	ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	provider, err := metrics.NewTelemetryProvider(ctxTimeout, config)
	if err != nil {
		// If we can't create a real provider, use a no-op one
		t.Logf("Using no-op provider due to: %v", err)
		provider = metrics.NewNoopTelemetryProvider()
	} else {
		defer func() {
			err := provider.Shutdown(ctx)
			if err != nil {
				t.Logf("Error shutting down provider: %v", err)
			}
		}()
	}

	// Get the cluster metrics manager
	manager := provider.GetClusterMetrics()
	require.NotNil(t, manager)

	t.Run("Basic Operations", func(t *testing.T) {
		// Create test object key
		key := client.ObjectKey{Namespace: "test-ns", Name: "test-cluster"}

		now := metav1.Now()
		// Create a minimal test cluster
		cluster := &apiv1.EtcdCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: key.Namespace,
				Name:      key.Name,
			},
			Spec: apiv1.EtcdClusterSpec{
				Replicas: 3,
				Version:  "3.5.0",
				Backup: apiv1.BackupSpec{
					Enabled: true,
				},
			},
			Status: apiv1.EtcdClusterStatus{
				AvailableReplicas: 3,
				UpdatedReplicas:   3,
				LearnerReplicas:   1,
				Conditions: []apiv1.ClusterCondition{
					{
						Type:   apiv1.ClusterAvailable,
						Status: corev1.ConditionTrue,
					},
				},
				Backup: apiv1.BackupStatus{
					Enabled:            true,
					LastScheduleTime:   &now,
					LastSuccessfulTime: &now,
				},
			},
		}

		// Get metrics for the cluster
		metrics, err := manager.GetOrCreateMetrics(key)
		require.NoError(t, err)
		require.NotNil(t, metrics)

		// Record metrics
		metrics.Record(ctx, cluster)

		// Test Record with nil (should not panic)
		assert.NotPanics(t, func() {
			metrics.Record(ctx, nil)
		})

		// Delete metrics
		err = manager.DeleteMetrics(ctx, key)
		assert.NoError(t, err)

		// Test getting metrics after deletion (should create new ones)
		metrics, err = manager.GetOrCreateMetrics(key)
		require.NoError(t, err)
		require.NotNil(t, metrics)
	})

	t.Run("Metric Values and Attributes", func(t *testing.T) {
		// Create a manual reader for testing
		reader := metric.NewManualReader()
		mp := metric.NewMeterProvider(metric.WithReader(reader))
		meter := mp.Meter("test")

		// Create a manager with our test meter
		manager := metrics.NewClusterMetricsManager(meter, "test.namespace")
		require.NotNil(t, manager)

		key := client.ObjectKey{Namespace: "test-ns", Name: "test-cluster"}
		now := time.Now()

		cluster := &apiv1.EtcdCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: key.Namespace,
				Name:      key.Name,
			},
			Spec: apiv1.EtcdClusterSpec{
				Replicas: 3,
			},
			Status: apiv1.EtcdClusterStatus{
				AvailableReplicas: 2,
				UpdatedReplicas:   2,
				LearnerReplicas:   1,
				Conditions: []apiv1.ClusterCondition{
					{
						Type:   apiv1.ClusterAvailable,
						Status: corev1.ConditionTrue,
					},
				},
				Backup: apiv1.BackupStatus{
					Enabled:            true,
					LastScheduleTime:   &metav1.Time{Time: now},
					LastSuccessfulTime: &metav1.Time{Time: now},
				},
			},
		}

		// Get metrics and record values
		clusterMetrics, err := manager.GetOrCreateMetrics(key)
		require.NoError(t, err)
		clusterMetrics.Record(ctx, cluster)

		// Collect metrics
		var resourceMetrics metricdata.ResourceMetrics
		err = reader.Collect(ctx, &resourceMetrics)
		require.NoError(t, err)

		// Helper function to find metric by name
		findMetric := func(name string) *metricdata.Gauge[int64] {
			for _, scope := range resourceMetrics.ScopeMetrics {
				for _, metric := range scope.Metrics {
					if metric.Name == "test.namespace.cluster."+name {
						if gauge, ok := metric.Data.(metricdata.Gauge[int64]); ok {
							return &gauge
						}
					}
				}
			}
			return nil
		}

		// Verify metric values and attributes
		expectedAttributes := []attribute.KeyValue{
			attribute.String("namespace", key.Namespace),
			attribute.String("name", key.Name),
		}

		testCases := []struct {
			metricName string
			expected   int64
		}{
			{metrics.ClusterMetricAvailable, 1},
			{metrics.ClusterMetricAvailableReplicas, 2},
			{metrics.ClusterMetricDesiredReplicas, 3},
			{metrics.ClusterMetricUpdatedReplicas, 2},
			{metrics.ClusterMetricLearnerReplicas, 1},
			{metrics.ClusterMetricBackupEnabled, 1},
			{metrics.ClusterMetricBackupLastScheduleTime, now.Unix()},
			{metrics.ClusterMetricBackupLastSuccessfulTime, now.Unix()},
		}

		for _, tc := range testCases {
			t.Run(tc.metricName, func(t *testing.T) {
				gauge := findMetric(tc.metricName)
				require.NotNil(t, gauge, "Metric %s should exist", tc.metricName)
				require.Len(t, gauge.DataPoints, 1, "Should have one data point")

				dp := gauge.DataPoints[0]
				assert.Equal(t, tc.expected, dp.Value, "Metric value should match")
				assert.ElementsMatch(t, expectedAttributes, dp.Attributes.ToSlice(),
					"Metric attributes should match")
			})
		}
	})

	t.Run("Concurrent Access", func(t *testing.T) {
		var wg sync.WaitGroup
		keys := make([]client.ObjectKey, 10)
		for i := 0; i < 10; i++ {
			keys[i] = client.ObjectKey{
				Namespace: "test-ns",
				Name:      fmt.Sprintf("test-cluster-%d", i),
			}
		}

		// Test concurrent creation
		for _, key := range keys {
			wg.Add(1)
			go func(k client.ObjectKey) {
				defer wg.Done()
				metrics, err := manager.GetOrCreateMetrics(k)
				assert.NoError(t, err)
				assert.NotNil(t, metrics)
			}(key)
		}
		wg.Wait()

		// Test concurrent deletion
		for _, key := range keys {
			wg.Add(1)
			go func(k client.ObjectKey) {
				defer wg.Done()
				err := manager.DeleteMetrics(ctx, k)
				assert.NoError(t, err)
			}(key)
		}
		wg.Wait()
	})

	t.Run("Edge Cases", func(t *testing.T) {
		key := client.ObjectKey{Namespace: "test-ns", Name: "test-cluster"}

		// Test cluster with zero values
		zeroCluster := &apiv1.EtcdCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: key.Namespace,
				Name:      key.Name,
			},
			Spec: apiv1.EtcdClusterSpec{
				Replicas: 0,
			},
			Status: apiv1.EtcdClusterStatus{
				Conditions: []apiv1.ClusterCondition{
					{
						Type:   apiv1.ClusterAvailable,
						Status: corev1.ConditionFalse,
					},
				},
			},
		}

		metrics, err := manager.GetOrCreateMetrics(key)
		require.NoError(t, err)
		metrics.Record(ctx, zeroCluster)

		// Test deleting non-existent metrics
		err = manager.DeleteMetrics(ctx, client.ObjectKey{Namespace: "non", Name: "existent"})
		assert.NoError(t, err)

		// Test double deletion
		err = manager.DeleteMetrics(ctx, key)
		assert.NoError(t, err)
		err = manager.DeleteMetrics(ctx, key)
		assert.NoError(t, err)
	})
}
