package cluster

import (
	"testing"

	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/agoda-com/etcd-operator/pkg/resources"
)

// Test fixtures
func createTestCluster() *apiv1.EtcdCluster {
	return &apiv1.EtcdCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: apiv1.EtcdClusterSpec{
			Version:  "v3.5.7",
			Replicas: 3,
			Resources: corev1.ResourceList{
				corev1.ResourceCPU:     resource.MustParse("2"),
				corev1.ResourceMemory:  resource.MustParse("4G"),
				corev1.ResourceStorage: resource.MustParse("4G"),
			},
		},
		Status: apiv1.EtcdClusterStatus{
			Phase:      apiv1.ClusterRunning,
			SecretName: "test-cluster-user-root",
			Endpoint:   "https://test-cluster.default.svc.cluster.local:2379",
		},
	}
}

func createTestConfig() Config {
	return Config{
		Image:           "etcd",
		ControllerImage: "etcd-operator",
		BackupEnv: map[string]string{
			"AWS_ACCESS_KEY_ID":     "example",
			"AWS_SECRET_ACCESS_KEY": "example",
			"AWS_BUCKET_NAME":       "example",
		},
	}
}

func TestRestoreContainer(t *testing.T) {
	config := createTestConfig()

	tests := []struct {
		name string
		spec *apiv1.RestoreSpec
	}{
		{
			name: "default",
		},
		{
			name: "key",
			spec: &apiv1.RestoreSpec{
				Key: ptr.To("test-key"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := createTestCluster()
			cluster.Spec.Restore = tt.spec
			// restore container is only injected in Bootstrap phase
			cluster.Status.Phase = apiv1.ClusterBootstrap

			container := RestoreContainer(cluster, config)

			// Convert container to YAML for golden file comparison
			got, err := yaml.Marshal(container)
			if err != nil {
				t.Fatal("marshal:", err)
			}

			golden.Assert(t, string(got), t.Name()+".yaml")
		})
	}
}

func TestBackupCronJob(t *testing.T) {
	config := createTestConfig()
	tests := []struct {
		name string
		spec *apiv1.BackupSpec
	}{
		{
			name: "default",
		},
		{
			name: "schedule",
			spec: &apiv1.BackupSpec{
				Schedule: "@midnight",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := createTestCluster()
			cluster.Spec.Backup = tt.spec

			builder := resources.NewBuilder(cluster)
			cronJob := BackupCronJob(builder, cluster, config)

			// Convert spec to YAML for golden file comparison
			got, err := yaml.Marshal(cronJob)
			if err != nil {
				t.Fatal("marshal:", err)
			}

			golden.Assert(t, string(got), t.Name()+".yaml")
		})
	}
}

func TestDefragCronJob(t *testing.T) {
	config := createTestConfig()
	tests := []struct {
		name string
		spec *apiv1.DefragSpec
	}{
		{
			name: "default",
		},
		{
			name: "schedule",
			spec: &apiv1.DefragSpec{
				Schedule: ptr.To("@midnight"),
			},
		},
		{
			name: "threshold",
			spec: &apiv1.DefragSpec{
				Size:  ptr.To(resource.MustParse("1G")),
				Ratio: ptr.To("0.1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := createTestCluster()
			cluster.Spec.Defrag = tt.spec

			builder := resources.NewBuilder(cluster)
			cronJob := DefragCronJob(builder, cluster, config)

			// Convert spec to YAML for golden file comparison
			got, err := yaml.Marshal(cronJob)
			if err != nil {
				t.Fatal("marshal:", err)
			}

			golden.Assert(t, string(got), t.Name()+".yaml")
		})
	}
}
