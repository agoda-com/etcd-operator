package metrics

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/google/go-cmp/cmp"
)

var update = flag.Bool("update", false, "update golden files")

func TestObserver(t *testing.T) {
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
	ts := time.Date(2025, 5, 12, 17, 22, 0, 0, time.UTC)
	cluster := &apiv1.EtcdCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "etcd",
			Name:      "test-cluster",
		},
		Spec: apiv1.EtcdClusterSpec{
			Version:  "v3.5.14",
			Replicas: 3,
		},
		Status: apiv1.EtcdClusterStatus{
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
		},
	}
	observer, err := Register(meter, cluster)
	if err != nil {
		t.Fatal("register:", err)
	}
	t.Cleanup(func() {
		err := observer.Unregister()
		if err != nil {
			t.Error("unregister:", err)
		}
	})

	metrics := &metricdata.ResourceMetrics{}
	err = reader.Collect(t.Context(), metrics)
	if err != nil {
		t.Fatal("collect:", err)
	}

	assertMetrics(t, metrics.ScopeMetrics)
}

func assertMetrics(t testing.TB, actual []metricdata.ScopeMetrics) {
	t.Helper()

	name := filepath.Join("testdata", t.Name(), "metrics.golden.yaml")
	if *update {
		data, err := yaml.Marshal(mapMetrics(t, actual))
		if err != nil {
			t.Fatal("marshal golden:", err)
		}

		err = os.MkdirAll(filepath.Dir(name), 0777)
		if err != nil {
			t.Fatal("mkdir:", err)
		}

		err = os.WriteFile(name, data, 0666)
		if err != nil {
			t.Fatal("write golden:", err)
		}

		return
	}

	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal("read golden:", err)
	}

	expected := []Metric{}
	err = yaml.Unmarshal(data, &expected)
	if err != nil {
		t.Fatal("unmarshal golden:", err)
	}

	diff := cmp.Diff(expected, mapMetrics(t, actual))
	if diff != "" {
		t.Error("diff:", diff)
	}
}

type Metric struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	DataPoints  []DataPoint `json:"dataPoints"`
}

type DataPoint struct {
	Attributes map[string]string `json:"attributes"`
	Value      int64             `json:"value"`
}

func mapMetrics(t testing.TB, data []metricdata.ScopeMetrics) []Metric {
	t.Helper()

	res := []Metric{}
	for _, sm := range data {
		for _, metric := range sm.Metrics {
			gauge, ok := metric.Data.(metricdata.Gauge[int64])
			if !ok {
				t.Errorf("unsupported metric %q", metric.Name)
			}

			dataPoints := []DataPoint{}
			for _, dp := range gauge.DataPoints {
				dataPoints = append(dataPoints, mapDataPoint(dp))
			}

			res = append(res, Metric{
				Name:        metric.Name,
				Description: metric.Description,
				DataPoints:  dataPoints,
			})
		}
	}

	return res
}

func mapDataPoint(dp metricdata.DataPoint[int64]) DataPoint {
	attributes := map[string]string{}
	for _, attr := range dp.Attributes.ToSlice() {
		attributes[string(attr.Key)] = attr.Value.AsString()
	}

	return DataPoint{
		Attributes: attributes,
		Value:      dp.Value,
	}
}
