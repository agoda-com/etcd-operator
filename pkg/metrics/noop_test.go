package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/agoda-com/etcd-operator/pkg/metrics"
)

func TestNoopTelemetryProvider(t *testing.T) {
	t.Run("NewNoopTelemetryProvider", func(t *testing.T) {
		provider := metrics.NewNoopTelemetryProvider()
		require.NotNil(t, provider, "Provider should not be nil")

		clusterMetrics := provider.GetClusterMetrics()
		require.NotNil(t, clusterMetrics, "Cluster metrics should not be nil")
	})

	t.Run("GetClusterMetrics", func(t *testing.T) {
		provider := metrics.NewNoopTelemetryProvider()
		metrics1 := provider.GetClusterMetrics()
		metrics2 := provider.GetClusterMetrics()

		assert.Equal(t, metrics1, metrics2, "Multiple calls to GetClusterMetrics should return the same instance")
	})

	t.Run("Shutdown", func(t *testing.T) {
		provider := metrics.NewNoopTelemetryProvider()
		ctx := context.Background()

		err := provider.Shutdown(ctx)
		assert.NoError(t, err, "Shutdown should not return an error")

		// Test with cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err = provider.Shutdown(ctx)
		assert.NoError(t, err, "Shutdown should not return an error even with cancelled context")
	})

	t.Run("Record Metrics", func(t *testing.T) {
		provider := metrics.NewNoopTelemetryProvider()
		clusterMetrics := provider.GetClusterMetrics()

		key := client.ObjectKey{
			Namespace: "test-namespace",
			Name:      "test-cluster",
		}

		// Create a test cluster
		cluster := &apiv1.EtcdCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
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
					LastScheduleTime:   &metav1.Time{Time: time.Now()},
					LastSuccessfulTime: &metav1.Time{Time: time.Now()},
				},
			},
		}

		// These calls should not panic
		assert.NotPanics(t, func() {
			metrics, err := clusterMetrics.GetOrCreateMetrics(key)
			require.NoError(t, err)
			metrics.Record(context.Background(), cluster)
		}, "Recording metrics should not panic")

		// Test with nil cluster
		assert.NotPanics(t, func() {
			metrics, err := clusterMetrics.GetOrCreateMetrics(key)
			require.NoError(t, err)
			metrics.Record(context.Background(), nil)
		}, "Recording nil cluster should not panic")

		// Test with cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.NotPanics(t, func() {
			metrics, err := clusterMetrics.GetOrCreateMetrics(key)
			require.NoError(t, err)
			metrics.Record(ctx, cluster)
		}, "Recording with cancelled context should not panic")
	})
}
