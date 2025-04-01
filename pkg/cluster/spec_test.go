package cluster_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/agoda-com/etcd-operator/pkg/cluster"
)

// Test fixtures
func createTestCluster() *apiv1.EtcdCluster {
	return &apiv1.EtcdCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: apiv1.EtcdClusterSpec{
			Version:      "v3.5.7",
			Replicas:     3,
			StorageQuota: resource.MustParse("4G"),
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("4G"),
				},
			},
		},
		Status: apiv1.EtcdClusterStatus{
			Phase:      apiv1.ClusterRunning,
			SecretName: "test-cluster-user-root",
			Endpoint:   "https://test-cluster.default.svc.cluster.local:2379",
		},
	}
}

func createTestConfig() cluster.Config {
	return cluster.Config{
		Image:           "etcd",
		ControllerImage: "etcd-controller",
	}
}

func TestRestoreContainer(t *testing.T) {
	tests := []struct {
		name    string
		cluster *apiv1.EtcdCluster
		config  cluster.Config
	}{
		{
			name: "restore_container_with_prefix",
			cluster: func() *apiv1.EtcdCluster {
				c := createTestCluster()
				c.Spec.Restore = &apiv1.RestoreSpec{
					SecretName: "backup-secret",
					Prefix:     "test-prefix",
				}
				return c
			}(),
			config: createTestConfig(),
		},
		{
			name: "restore_container_with_key",
			cluster: func() *apiv1.EtcdCluster {
				c := createTestCluster()
				c.Spec.Restore = &apiv1.RestoreSpec{
					SecretName: "backup-secret",
					Key:        "test-key",
				}
				return c
			}(),
			config: createTestConfig(),
		},
		{
			name: "restore_container_with_default_prefix",
			cluster: func() *apiv1.EtcdCluster {
				c := createTestCluster()
				c.Spec.Restore = &apiv1.RestoreSpec{
					SecretName: "backup-secret",
				}
				return c
			}(),
			config: createTestConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := cluster.RestoreContainer(tt.cluster, tt.config)

			// Convert container to YAML for golden file comparison
			got, err := yaml.Marshal(container)
			require.NoError(t, err)

			golden.Assert(t, string(got), tt.name+".yaml")
		})
	}
}

func TestBackupPodSpec(t *testing.T) {
	tests := []struct {
		name    string
		cluster *apiv1.EtcdCluster
		config  cluster.Config
	}{
		{
			name: "backup_pod_spec_basic",
			cluster: func() *apiv1.EtcdCluster {
				c := createTestCluster()
				c.Spec.Backup = apiv1.BackupSpec{
					SecretName: "backup-secret",
				}
				return c
			}(),
			config: createTestConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := cluster.BackupPodSpec(tt.cluster, tt.config)

			// Convert spec to YAML for golden file comparison
			got, err := yaml.Marshal(spec)
			require.NoError(t, err)

			golden.Assert(t, string(got), tt.name+".yaml")
		})
	}
}

func TestDefragPodSpec(t *testing.T) {
	tests := []struct {
		name    string
		cluster *apiv1.EtcdCluster
		config  cluster.Config
	}{
		{
			name:    "defrag_pod_spec_basic",
			cluster: createTestCluster(),
			config:  createTestConfig(),
		},
		{
			name: "defrag_pod_spec_with_threshold",
			cluster: func() *apiv1.EtcdCluster {
				c := createTestCluster()
				c.Spec.Defrag = apiv1.DefragSpec{
					Threshold: &apiv1.DefragThreshold{
						Size:  resource.MustParse("1G"),
						Ratio: "0.1",
					},
				}
				return c
			}(),
			config: createTestConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := cluster.DefragPodSpec(tt.cluster, tt.config)

			// Convert spec to YAML for golden file comparison
			got, err := yaml.Marshal(spec)
			require.NoError(t, err)

			golden.Assert(t, string(got), tt.name+".yaml")
		})
	}
}

func TestBucketInfoVolume(t *testing.T) {
	testCluster := createTestCluster()
	testCluster.Spec.Backup.SecretName = "backup-secret"

	volume := cluster.BucketInfoVolume(testCluster)

	// Convert volume to YAML for golden file comparison
	got, err := yaml.Marshal(volume)
	require.NoError(t, err)

	golden.Assert(t, string(got), "bucket_info_volume.yaml")
}
