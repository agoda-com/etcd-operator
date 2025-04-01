package cluster

import (
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/agoda-com/etcd-operator/pkg/etcd"
)

const (
	BaseConfigFile       = "/etc/etcd/config/base/etcd.json"
	ConfigFile           = "/etc/etcd/config/etcd.json"
	CredentialsDir       = "/etc/etcd/pki"
	ServerCredentialsDir = "/etc/etcd/pki/server"
	PeerCredentialsDir   = "/etc/etcd/pki/peer"
	BucketInfoFile       = "/etc/etcd/cosi/bucket.json"
	DataDir              = "/var/lib/etcd/data"

	DefragSchedule = "0 1 * * *" // 1:00 AM every day
	BackupSchedule = "0 * * * *" // every hour
	JobTTL         = 24 * time.Hour
	ActiveDeadline = 5 * time.Minute
)

var (
	DefaultResources = corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("4"),
		corev1.ResourceMemory: resource.MustParse("16G"),
	}

	InitResources = corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1"),
		corev1.ResourceMemory: resource.MustParse("128M"),
	}
)

func PodSpec(cluster *apiv1.EtcdCluster, config Config) corev1.PodSpec {
	affinity := &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 1,
					PodAffinityTerm: corev1.PodAffinityTerm{
						TopologyKey: "kubernetes.io/hostname",
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								apiv1.ClusterLabel: apiv1.ClusterLabelValue(client.ObjectKeyFromObject(cluster)),
							},
						},
					},
				},
			},
		},
	}

	volumes := []corev1.Volume{
		{
			Name: "base-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cluster.Name,
					},
				},
			},
		},
		{
			Name: "pki",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium:    cluster.Spec.StorageMedium,
					SizeLimit: &cluster.Spec.StorageQuota,
				},
			},
		},
	}

	initContainters := []corev1.Container{
		SidecarContainer(cluster, config),
	}

	containers := []corev1.Container{
		ETCDContainer(cluster, config),
	}

	if cluster.Spec.Restore != nil && cluster.Spec.Restore.SecretName != "" {
		volumes = append(volumes, BucketInfoVolume(cluster))
		initContainters = append(initContainters, RestoreContainer(cluster, config))
	}

	return corev1.PodSpec{
		InitContainers:     initContainters,
		Containers:         containers,
		Affinity:           affinity,
		Volumes:            volumes,
		RuntimeClassName:   cluster.Spec.RuntimeClassName,
		PriorityClassName:  cluster.Spec.PriorityClassName,
		ServiceAccountName: cluster.Name,
	}
}

func ETCDConfig(cluster *apiv1.EtcdCluster, c Config) ([]byte, error) {
	initialState := etcd.InitialStateExisiting
	if cluster.Status.Phase == apiv1.ClusterBootstrap {
		initialState = etcd.InitialStateNew
	}

	etcdConfig := etcd.Config{
		InitialClusterState:     initialState,
		InitialClusterToken:     cluster.Name,
		DataDir:                 DataDir,
		SnapshotCount:           10000,
		AutoCompactionMode:      "revision",
		AutoCompactionRetention: "100",
		ListenClientURLs:        "https://0.0.0.0:2379",
		ListenPeerURLs:          "https://0.0.0.0:2380",
		ListenMetricsURLs:       "http://0.0.0.0:2381",
		ClientTransportSecurity: &etcd.TransportSecurity{
			CertFile:       path.Join(ServerCredentialsDir, etcd.DefaultCertFile),
			KeyFile:        path.Join(ServerCredentialsDir, etcd.DefaultKeyFile),
			TrustedCAFile:  path.Join(ServerCredentialsDir, etcd.DefaultCACertFile),
			ClientCertAuth: true,
			AutoTLS:        false,
		},
		PeerTransportSecurity: &etcd.TransportSecurity{
			CertFile:       path.Join(PeerCredentialsDir, etcd.DefaultCertFile),
			KeyFile:        path.Join(PeerCredentialsDir, etcd.DefaultKeyFile),
			TrustedCAFile:  path.Join(PeerCredentialsDir, etcd.DefaultCACertFile),
			ClientCertAuth: true,
			AutoTLS:        false,
		},
		ExpInitialCorruptCheck:         true,
		ExpWatchProgressNotifyInterval: 5 * time.Second,
	}

	data, err := json.MarshalIndent(etcdConfig, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	return data, nil
}

func ETCDContainer(cluster *apiv1.EtcdCluster, config Config) corev1.Container {
	resources := corev1.ResourceRequirements{
		Requests: maps.Clone(DefaultResources),
		Limits:   maps.Clone(DefaultResources),
	}
	maps.Copy(resources.Requests, cluster.Spec.Resources.Requests)
	maps.Copy(resources.Limits, cluster.Spec.Resources.Limits)

	return corev1.Container{
		Name:  "etcd",
		Image: config.Image + ":" + cluster.Spec.Version,
		Command: []string{
			"etcd",
			"--config-file=" + ConfigFile,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: path.Dir(DataDir),
			},
			{
				Name:      "config",
				MountPath: path.Dir(ConfigFile),
				ReadOnly:  true,
			},
			{
				Name:      "pki",
				MountPath: CredentialsDir,
				ReadOnly:  true,
			},
		},
		StartupProbe: &corev1.Probe{
			FailureThreshold:    24,
			InitialDelaySeconds: 5,
			PeriodSeconds:       5,
			SuccessThreshold:    1,
			TimeoutSeconds:      15,
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/health?serializable=false",
					Port:   intstr.FromInt(2381),
					Scheme: "HTTP",
				},
			},
		},
		LivenessProbe: &corev1.Probe{
			FailureThreshold: 8,
			PeriodSeconds:    5,
			SuccessThreshold: 1,
			TimeoutSeconds:   15,
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/health?exclude=NOSPACE&serializable=true",
					Port:   intstr.FromInt(2381),
					Scheme: "HTTP",
				},
			},
		},
		Resources: resources,
	}
}

func SidecarContainer(cluster *apiv1.EtcdCluster, config Config) corev1.Container {
	return corev1.Container{
		Name:          "sidecar",
		Image:         config.ControllerImage,
		RestartPolicy: ptr.To(corev1.ContainerRestartPolicyAlways),
		Command:       []string{"etcd-sidecar"},
		Args: []string{
			"--base-config=" + BaseConfigFile,
			"--config=" + ConfigFile,
			"--endpoint=" + cluster.Status.Endpoint,
			"--health-address=:8081",
			"--zap-log-level=debug",
		},
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "base-config",
				MountPath: path.Dir(BaseConfigFile),
				ReadOnly:  true,
			},
			{
				Name:      "config",
				MountPath: path.Dir(ConfigFile),
				ReadOnly:  false,
			},
			{
				Name:      "pki",
				MountPath: CredentialsDir,
				ReadOnly:  false,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: InitResources,
			Limits:   InitResources,
		},
		StartupProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Port: intstr.FromInt32(8081),
					Path: "/healthz",
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       5,
			FailureThreshold:    24,
		},
	}
}

func RestoreContainer(cluster *apiv1.EtcdCluster, config Config) corev1.Container {
	resources := corev1.ResourceRequirements{
		Requests: maps.Clone(DefaultResources),
		Limits:   maps.Clone(DefaultResources),
	}
	maps.Copy(resources.Requests, cluster.Spec.Resources.Requests)
	maps.Copy(resources.Limits, cluster.Spec.Resources.Limits)

	args := []string{
		"restore",
		"--config=" + BaseConfigFile,
		"--bucket-info=" + BucketInfoFile,
	}

	spec := cluster.Spec.Restore
	switch {
	case spec.Prefix != "":
		args = append(args, "--prefix="+spec.Prefix)
	case spec.Key != "":
		args = append(args, "--key="+spec.Key)
	case spec.Prefix == "":
		prefix := fmt.Sprintf("%s/%s/", cluster.Namespace, cluster.Name)
		args = append(args, "--prefix="+prefix)
	}

	return corev1.Container{
		Name:    "restore",
		Image:   config.ControllerImage,
		Command: []string{"etcd-tools"},
		Args:    args,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "base-config",
				MountPath: path.Dir(BaseConfigFile),
				ReadOnly:  true,
			},
			{
				Name:      "bucket-info",
				MountPath: path.Dir(BucketInfoFile),
				ReadOnly:  true,
			},
			{
				Name:      "data",
				MountPath: path.Dir(DataDir),
				ReadOnly:  false,
			},
		},
		Resources: resources,
	}
}

func BackupPodSpec(cluster *apiv1.EtcdCluster, config Config) corev1.PodSpec {
	prefix := path.Join(cluster.Namespace, cluster.Name)
	args := []string{
		"backup",
		"--endpoint=" + cluster.Status.Endpoint,
		"--credentials-dir=" + CredentialsDir,
		"--bucket-info=" + BucketInfoFile,
		"--prefix=" + prefix,
	}

	credentials := CredentialsSecretVolume(cluster)
	bucketInfo := BucketInfoVolume(cluster)

	container := corev1.Container{
		Name:    "backup",
		Image:   config.ControllerImage,
		Command: []string{"etcd-tools"},
		Args:    args,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      credentials.Name,
				MountPath: CredentialsDir,
				ReadOnly:  true,
			},
			{
				Name:      bucketInfo.Name,
				MountPath: path.Dir(BucketInfoFile),
				ReadOnly:  true,
			},
		},
	}

	volumes := []corev1.Volume{
		credentials,
		bucketInfo,
	}

	return corev1.PodSpec{
		RestartPolicy:    corev1.RestartPolicyOnFailure,
		Containers:       []corev1.Container{container},
		Volumes:          volumes,
		RuntimeClassName: cluster.Spec.RuntimeClassName,
	}
}

func DefragPodSpec(cluster *apiv1.EtcdCluster, config Config) corev1.PodSpec {
	args := []string{
		"defrag",
		"--endpoint=" + cluster.Status.Endpoint,
		"--credentials-dir=" + CredentialsDir,
	}

	threshold := cluster.Spec.Defrag.Threshold
	if threshold != nil && threshold.Ratio != "" {
		args = append(args, "--ratio="+threshold.Ratio)
	}

	if threshold != nil && !threshold.Size.IsZero() {
		args = append(args, "--unused-size="+threshold.Size.String())
	}

	credentials := CredentialsSecretVolume(cluster)

	container := corev1.Container{
		Name:    "defrag",
		Image:   config.ControllerImage,
		Command: []string{"etcd-tools"},
		Args:    args,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      credentials.Name,
				MountPath: CredentialsDir,
			},
		},
	}

	return corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyOnFailure,
		Containers:    []corev1.Container{container},
		Volumes:       []corev1.Volume{credentials},
	}
}

func BucketInfoVolume(cluster *apiv1.EtcdCluster) corev1.Volume {
	return corev1.Volume{
		Name: "bucket-info",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: cluster.Spec.Backup.SecretName,
				Items: []corev1.KeyToPath{{
					Key:  "BucketInfo",
					Path: path.Base(BucketInfoFile),
				}},
			},
		},
	}
}

func CredentialsSecretVolume(cluster *apiv1.EtcdCluster) corev1.Volume {
	return corev1.Volume{
		Name: "pki",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: cluster.Status.SecretName,
			},
		},
	}
}
