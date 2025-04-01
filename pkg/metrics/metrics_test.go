package metrics_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/agoda-com/etcd-operator/pkg/metrics"
)

func TestClusterMetricsManager(t *testing.T) {
	// Create a meter provider for testing
	mp := metric.NewMeterProvider()
	meter := mp.Meter("test")

	// Create a manager
	manager := metrics.NewClusterMetricsManager(meter, "test.namespace")
	require.NotNil(t, manager)

	t.Run("Basic Operations", func(t *testing.T) {
		key := client.ObjectKey{Namespace: "test-ns", Name: "test-cluster"}

		// Get metrics for a new cluster
		metrics1, err := manager.GetOrCreateMetrics(key)
		require.NoError(t, err)
		require.NotNil(t, metrics1)

		// Get metrics again for the same cluster (should return existing)
		metrics2, err := manager.GetOrCreateMetrics(key)
		require.NoError(t, err)
		require.NotNil(t, metrics2)

		// Record some metrics with both instances
		cluster := &apiv1.EtcdCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: key.Namespace,
				Name:      key.Name,
			},
			Spec: apiv1.EtcdClusterSpec{
				Replicas: 3,
			},
			Status: apiv1.EtcdClusterStatus{
				AvailableReplicas: 3,
				Conditions: []apiv1.ClusterCondition{
					{
						Type:   apiv1.ClusterAvailable,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}

		metrics1.Record(context.Background(), cluster)
		metrics2.Record(context.Background(), cluster)

		// Delete metrics
		err = manager.DeleteMetrics(context.Background(), key)
		require.NoError(t, err)

		// Get metrics after deletion (should create new ones)
		metrics3, err := manager.GetOrCreateMetrics(key)
		require.NoError(t, err)
		require.NotNil(t, metrics3)
	})

	t.Run("Concurrent Access", func(t *testing.T) {
		const numGoroutines = 10
		done := make(chan bool)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				key := client.ObjectKey{
					Namespace: "test-ns",
					Name:      "test-cluster",
				}

				// Get or create metrics
				metrics, err := manager.GetOrCreateMetrics(key)
				assert.NoError(t, err)
				assert.NotNil(t, metrics)

				// Record some metrics
				cluster := &apiv1.EtcdCluster{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: key.Namespace,
						Name:      key.Name,
					},
					Spec: apiv1.EtcdClusterSpec{
						Replicas: int32(id),
					},
					Status: apiv1.EtcdClusterStatus{
						AvailableReplicas: int32(id),
						Conditions: []apiv1.ClusterCondition{
							{
								Type:   apiv1.ClusterAvailable,
								Status: corev1.ConditionTrue,
							},
						},
					},
				}
				metrics.Record(context.Background(), cluster)

				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	t.Run("Edge Cases", func(t *testing.T) {
		key := client.ObjectKey{Namespace: "test-ns", Name: "test-cluster"}

		// Test with nil cluster
		metrics, err := manager.GetOrCreateMetrics(key)
		require.NoError(t, err)
		assert.NotPanics(t, func() {
			metrics.Record(context.Background(), nil)
		})

		// Test with empty cluster
		emptyCluster := &apiv1.EtcdCluster{}
		assert.NotPanics(t, func() {
			metrics.Record(context.Background(), emptyCluster)
		})

		// Test deleting non-existent metrics
		err = manager.DeleteMetrics(context.Background(), client.ObjectKey{Namespace: "non", Name: "existent"})
		assert.NoError(t, err)

		// Test double deletion
		err = manager.DeleteMetrics(context.Background(), key)
		assert.NoError(t, err)
		err = manager.DeleteMetrics(context.Background(), key)
		assert.NoError(t, err)
	})
}
