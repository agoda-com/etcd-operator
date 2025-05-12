package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/agoda-com/etcd-operator/pkg/backup"
	"github.com/agoda-com/etcd-operator/pkg/conditions"
	"github.com/agoda-com/etcd-operator/pkg/etcd"
	"github.com/agoda-com/etcd-operator/pkg/resources"
)

const (
	BaseConfigFile       = "/etc/etcd/config/base/etcd.json"
	ConfigFile           = "/etc/etcd/config/etcd.json"
	CredentialsDir       = "/etc/etcd/pki"
	ServerCredentialsDir = "/etc/etcd/pki/server"
	PeerCredentialsDir   = "/etc/etcd/pki/peer"
	DataDir              = "/var/lib/etcd/data"

	DefragSchedule = "0 1 * * *" // 1:00 AM every day
	BackupSchedule = "0 * * * *" // every hour
	JobTTL         = 24 * time.Hour
	ActiveDeadline = 5 * time.Minute
)

var (
	DefaultResources = corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("4"),
		corev1.ResourceMemory: resource.MustParse("8G"),
	}

	InitResources = corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1"),
		corev1.ResourceMemory: resource.MustParse("128M"),
	}
)

func Deployment(ctx context.Context, builder *resources.Builder, cluster *apiv1.EtcdCluster, config Config) (*appsv1.Deployment, error) {
	// restore requested without key - determine latest backup
	if cluster.Status.Phase == apiv1.ClusterBootstrap && cluster.Spec.Restore != nil && cluster.Spec.Restore.Key == nil {
		prefix := path.Join(cluster.Namespace, cluster.Name)
		if cluster.Spec.Restore.Prefix != nil {
			prefix = *cluster.Spec.Restore.Prefix
		}

		scl, err := backup.NewClient(ctx)
		if err != nil {
			return nil, err
		}

		bucket := config.BackupEnv["AWS_BUCKET_NAME"]
		obj, err := backup.LatestBackup(ctx, scl, bucket, prefix)
		switch {
		case err != nil:
			return nil, err
		case obj == nil || obj.Key == nil:
			conditions.Upsert(&cluster.Status.Conditions, apiv1.ClusterCondition{
				Type:    apiv1.ClusterRestore,
				Status:  corev1.ConditionFalse,
				Reason:  "BackupNotFound",
				Message: "latest backup object not found",
			})
			return nil, nil
		default:
			cluster.Spec.Restore.Key = obj.Key
		}
	}

	etcdData, err := ETCDConfig(cluster, config)
	if err != nil {
		return nil, fmt.Errorf("etcd config: %v", err)
	}

	builder.ConfigMap().
		Data("etcd.json", string(etcdData))

	serviceAccount := builder.ServiceAccount()
	builder.RoleBinding().
		ServiceAccountSubject(serviceAccount.ServiceAccount).
		ClusterRoleRef("etcd-sidecar")

	clusterLabel := apiv1.ClusterLabelValue(client.ObjectKeyFromObject(cluster))

	builder.PodDisruptionBudget().
		Selector(apiv1.ClusterLabel, clusterLabel).
		MaxUnavailable(1)

	// member deployment - bootstrap with single replica
	replicas := cluster.Spec.Replicas
	if cluster.Status.Phase == apiv1.ClusterBootstrap {
		replicas = 1
	}

	deployment := builder.Deployment().
		Replicas(replicas).
		MaxUnavailable(0).
		MaxSurge(1).
		Selector(apiv1.ClusterLabel, clusterLabel).
		PodSpec(PodSpec(cluster, config))

	if cluster.Spec.PodTemplate != nil {
		deployment.
			PodLabels(cluster.Spec.PodTemplate.Labels).
			PodAnnotations(cluster.Spec.PodTemplate.Annotations)
	}

	// cluster service
	builder.Service().
		Selector(apiv1.ClusterLabel, clusterLabel).
		Selector(apiv1.LearnerLabel, "false").
		Port("etcd-client-ssl", 2379, 2379).
		Port("etcd-server-ssl", 2380, 2380).
		Headless(true)

	return deployment.Deployment, nil
}

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

	// set cpu/memory resources
	resources := maps.Clone(DefaultResources)
	for tpe, value := range cluster.Spec.Resources {
		_, existing := resources[tpe]
		if existing {
			resources[tpe] = value
		}
	}

	storageQuota := StorageQuota(cluster)

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
					SizeLimit: ptr.To(storageQuota),
				},
			},
		},
	}

	initContainters := []corev1.Container{
		SidecarContainer(cluster, config),
	}

	containers := []corev1.Container{{
		Name:  "etcd",
		Image: config.Image + ":" + cluster.Spec.Version,
		Command: []string{
			"etcd",
			"--config-file=" + ConfigFile,
		},
		Env: []corev1.EnvVar{
			{
				Name:  "ETCDCTL_CACERT",
				Value: path.Join(ServerCredentialsDir, etcd.DefaultCACertFile),
			},
			{
				Name:  "ETCDCTL_CERT",
				Value: path.Join(ServerCredentialsDir, etcd.DefaultCertFile),
			},
			{
				Name:  "ETCDCTL_KEY",
				Value: path.Join(ServerCredentialsDir, etcd.DefaultKeyFile),
			},
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
		Resources: corev1.ResourceRequirements{
			Requests: resources,
			Limits:   resources,
		},
	}}

	container := RestoreContainer(cluster, config)
	if container != nil {
		initContainters = append(initContainters, *container)
	}

	return corev1.PodSpec{
		InitContainers:     initContainters,
		Containers:         containers,
		Affinity:           affinity,
		Volumes:            volumes,
		ServiceAccountName: cluster.Name,
		PriorityClassName:  config.PriorityClassName,
	}
}

func ETCDConfig(cluster *apiv1.EtcdCluster, c Config) ([]byte, error) {
	initialState := etcd.InitialStateExisiting
	if cluster.Status.Phase == apiv1.ClusterBootstrap {
		initialState = etcd.InitialStateNew
	}

	storageQuota := StorageQuota(cluster)

	etcdConfig := etcd.Config{
		InitialClusterState:     initialState,
		InitialClusterToken:     cluster.Name,
		DataDir:                 DataDir,
		QuotaBackendBytes:       storageQuota.Value(),
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

func RestoreContainer(cluster *apiv1.EtcdCluster, config Config) *corev1.Container {
	// restore requested but backup credentials are not configured
	if cluster.Spec.Restore != nil && len(config.BackupEnv) == 0 {
		conditions.Upsert(&cluster.Status.Conditions, apiv1.ClusterCondition{
			Type:    apiv1.ClusterRestore,
			Status:  corev1.ConditionFalse,
			Reason:  "BackupNotConfigured",
			Message: "backup is not configured",
		})
		return nil
	}

	if cluster.Status.Phase != apiv1.ClusterBootstrap || cluster.Spec.Restore == nil || cluster.Spec.Restore.Key == nil {
		return nil
	}

	conditions.Upsert(&cluster.Status.Conditions, apiv1.ClusterCondition{
		Type:    apiv1.ClusterRestore,
		Status:  corev1.ConditionTrue,
		Reason:  "BackupFound",
		Message: fmt.Sprintf("using backup object %q", *cluster.Spec.Restore.Key),
	})

	return &corev1.Container{
		Name:    "restore",
		Image:   config.ControllerImage,
		Command: []string{"etcd-tools"},
		Args: []string{
			"restore",
			"--config=" + ConfigFile,
			"--key=" + *cluster.Spec.Restore.Key,
		},
		EnvFrom: []corev1.EnvFromSource{{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: cluster.Name + "-backup",
				},
			},
		}},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "config",
				MountPath: path.Dir(ConfigFile),
				ReadOnly:  true,
			},
			{
				Name:      "data",
				MountPath: path.Dir(DataDir),
				ReadOnly:  false,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: InitResources,
			Limits:   InitResources,
		},
	}
}

func BackupPodSpec(cluster *apiv1.EtcdCluster, config Config) corev1.PodSpec {
	prefix := path.Join(cluster.Namespace, cluster.Name)
	credentials := CredentialsSecretVolume(cluster)

	secretName := cluster.Name + "-backup"
	container := corev1.Container{
		Name:    "backup",
		Image:   config.ControllerImage,
		Command: []string{"etcd-tools"},
		Args: []string{
			"backup",
			"--endpoint=" + cluster.Status.Endpoint,
			"--credentials-dir=" + CredentialsDir,
			"--prefix=" + prefix,
		},
		EnvFrom: []corev1.EnvFromSource{{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
			},
		}},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      credentials.Name,
				MountPath: CredentialsDir,
				ReadOnly:  true,
			},
		},
	}

	return corev1.PodSpec{
		RestartPolicy:     corev1.RestartPolicyOnFailure,
		Containers:        []corev1.Container{container},
		Volumes:           []corev1.Volume{credentials},
		PriorityClassName: config.PriorityClassName,
	}
}

func DefragCronJob(builder *resources.Builder, cluster *apiv1.EtcdCluster, config Config) *batchv1.CronJob {
	schedule := DefragSchedule
	if cluster.Spec.Defrag != nil && cluster.Spec.Defrag.Schedule != nil {
		schedule = *cluster.Spec.Defrag.Schedule
	}

	suspend := false
	if cluster.Spec.Defrag != nil && cluster.Spec.Defrag.Suspend != nil {
		suspend = *cluster.Spec.Defrag.Suspend
	}

	cronJob := builder.CronJob("defrag").
		Suspend(suspend).
		ConcurrencyPolicy(batchv1.ForbidConcurrent).
		Schedule(schedule).
		TTL(JobTTL).
		ActiveDeadline(ActiveDeadline).
		PodSpec(DefragPodSpec(cluster, config))

	if cluster.Spec.PodTemplate != nil {
		cronJob.
			PodLabels(cluster.Spec.PodTemplate.Labels).
			PodAnnotations(cluster.Spec.PodTemplate.Annotations)
	}

	return cronJob.CronJob
}

func BackupCronJob(builder *resources.Builder, cluster *apiv1.EtcdCluster, config Config) *batchv1.CronJob {
	// if backup is not configured set status condition and mark cronjob for deletion
	if len(config.BackupEnv) == 0 {
		conditions.Upsert(&cluster.Status.Conditions, apiv1.ClusterCondition{
			Type:    apiv1.ClusterBackup,
			Status:  corev1.ConditionFalse,
			Reason:  "BackupNotConfigured",
			Message: "backup is not configured",
		})

		builder.Delete(&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cluster.Namespace,
				Name:      cluster.Name + "-backup",
			},
		})

		return nil
	}

	// if backup is configured create secret with s3 credentials and cronjob
	builder.Secret("backup").
		StringData(config.BackupEnv)

	schedule := BackupSchedule
	if cluster.Spec.Backup != nil && cluster.Spec.Backup.Schedule != "" {
		schedule = cluster.Spec.Backup.Schedule
	}

	suspend := false
	if cluster.Spec.Backup != nil {
		suspend = cluster.Spec.Backup.Suspend
	}

	cronJob := builder.CronJob("backup").
		Suspend(suspend).
		ConcurrencyPolicy(batchv1.ForbidConcurrent).
		Schedule(schedule).
		TTL(JobTTL).
		ActiveDeadline(ActiveDeadline).
		PodSpec(BackupPodSpec(cluster, config))

	if cluster.Spec.PodTemplate != nil {
		cronJob.
			PodLabels(cluster.Spec.PodTemplate.Labels).
			PodAnnotations(cluster.Spec.PodTemplate.Annotations)
	}

	return cronJob.CronJob
}

func DefragPodSpec(cluster *apiv1.EtcdCluster, config Config) corev1.PodSpec {
	args := []string{
		"defrag",
		"--endpoint=" + cluster.Status.Endpoint,
		"--credentials-dir=" + CredentialsDir,
	}

	if cluster.Spec.Defrag != nil && cluster.Spec.Defrag.Ratio != nil {
		ratio := *cluster.Spec.Defrag.Ratio
		args = append(args, "--ratio="+ratio)
	}

	if cluster.Spec.Defrag != nil && cluster.Spec.Defrag.Size != nil {
		size := cluster.Spec.Defrag.Size.String()
		args = append(args, "--unused-size="+size)
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
		RestartPolicy:     corev1.RestartPolicyOnFailure,
		Containers:        []corev1.Container{container},
		Volumes:           []corev1.Volume{credentials},
		PriorityClassName: config.PriorityClassName,
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

func StorageQuota(cluster *apiv1.EtcdCluster) resource.Quantity {
	// storage == memory
	storageQuota := DefaultResources[corev1.ResourceMemory]
	if cluster.Spec.Resources == nil {
		return storageQuota
	}

	// use spec.resources.storage if set
	_, ok := cluster.Spec.Resources[corev1.ResourceStorage]
	if ok {
		return cluster.Spec.Resources[corev1.ResourceStorage]
	}

	// use spec.resources.memory if set
	_, ok = cluster.Spec.Resources[corev1.ResourceMemory]
	if ok {
		return cluster.Spec.Resources[corev1.ResourceMemory]
	}

	// no override - use default
	return storageQuota
}
