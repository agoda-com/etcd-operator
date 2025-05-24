package metrics

import (
	"context"
	"os"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
)

func TestReconciler(t *testing.T) {
	kcl := setupEnvtest(t)

	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(
		metric.WithReader(reader),
	)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := provider.Shutdown(ctx)
		if err != nil {
			t.Error("shutdown meter provider:", err)
		}
	})

	meter := provider.Meter("test-meter")
	reconciler := &Reconciler{
		kcl:       kcl,
		meter:     meter,
		observers: map[client.ObjectKey]*Observer{},
	}

	ts := time.Date(2025, 5, 12, 17, 22, 0, 0, time.UTC)
	cluster := &apiv1.EtcdCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-cluster",
		},
		Spec: apiv1.EtcdClusterSpec{
			Version:  "v3.5.14",
			Replicas: 3,
		},
	}

	t.Run("cluster with empty status", func(t *testing.T) {
		err := kcl.Create(t.Context(), cluster)
		if err != nil {
			t.Fatal("create cluster:", err)
		}

		metrics := collect(t, reconciler, reader, cluster)
		assertMetrics(t, metrics.ScopeMetrics)
	})

	t.Run("cluster with status", func(t *testing.T) {
		patch := client.MergeFrom(cluster.DeepCopy())
		cluster.Status = apiv1.EtcdClusterStatus{
			Phase:             apiv1.ClusterRunning,
			Replicas:          3,
			ReadyReplicas:     2,
			AvailableReplicas: 2,
			LearnerReplicas:   1,
			UpdatedReplicas:   3,
			Backup: &apiv1.BackupStatus{
				LastScheduleTime:   ptr.To(metav1.NewTime(ts)),
				LastSuccessfulTime: ptr.To(metav1.NewTime(ts)),
			},
		}

		err := kcl.Status().Patch(t.Context(), cluster, patch)
		if err != nil {
			t.Fatal("patch cluster status:", err)
		}

		metrics := collect(t, reconciler, reader, cluster)
		assertMetrics(t, metrics.ScopeMetrics)
	})

	t.Run("no metrics for deleted cluster", func(t *testing.T) {
		err := kcl.Delete(t.Context(), cluster)
		if err != nil {
			t.Fatal("delete cluster:", err)
		}

		metrics := collect(t, reconciler, reader, cluster)
		if len(metrics.ScopeMetrics) != 0 {
			t.Error("expected metrics to be empty after cluster deletion")
		}
	})
}

func setupEnvtest(t testing.TB) client.Client {
	t.Helper()

	if testing.Short() || os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.SkipNow()
	}

	scheme := runtime.NewScheme()
	builder := runtime.NewSchemeBuilder(
		kscheme.AddToScheme,
		apiv1.AddToScheme,
	)
	err := builder.AddToScheme(scheme)
	if err != nil {
		t.Fatal("scheme:", err)
	}

	env := &envtest.Environment{
		Scheme: scheme,
		CRDDirectoryPaths: []string{
			"../../config/crd",
		},
	}
	kubeconfig, err := env.Start()
	if err != nil {
		t.Fatal("start envtest: ", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Log("stop envtest: ", err)
		}
	})

	kcl, err := client.New(kubeconfig, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		t.Fatal("k8s client:", err)
	}

	return kcl
}

func collect(t testing.TB, reconciler *Reconciler, reader *metric.ManualReader, cluster *apiv1.EtcdCluster) *metricdata.ResourceMetrics {
	t.Helper()

	key := client.ObjectKeyFromObject(cluster)
	_, err := reconciler.Reconcile(t.Context(), reconcile.Request{
		NamespacedName: key,
	})
	if err != nil {
		t.Fatal("reconcile:", err)
	}

	metrics := &metricdata.ResourceMetrics{}
	err = reader.Collect(t.Context(), metrics)
	if err != nil {
		t.Fatal("collect metrics:", err)
	}

	return metrics
}
