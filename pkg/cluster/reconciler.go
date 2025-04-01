package cluster

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/agoda-com/etcd-operator/pkg/conditions"
	"github.com/agoda-com/etcd-operator/pkg/etcd"
	"github.com/agoda-com/etcd-operator/pkg/metrics"
	"github.com/agoda-com/etcd-operator/pkg/resources"
)

var ErrOperationTimeout = errors.New("operation timeout")

type Reconciler struct {
	kcl       client.Client
	recorder  record.EventRecorder
	tlsCache  *etcd.TLSCache
	config    Config
	telemetry metrics.TelemetryProvider
}

type Config struct {
	Image           string
	ControllerImage string
}

// CreateControllerWithManager creates a new manager
func CreateControllerWithManager(mgr manager.Manager, tlsCache *etcd.TLSCache, config Config, telemetry metrics.TelemetryProvider) error {
	rateLimiter := workqueue.NewTypedItemFastSlowRateLimiter[reconcile.Request](1*time.Second, 5*time.Second, 10)
	reconciler := &Reconciler{
		kcl:       mgr.GetClient(),
		recorder:  mgr.GetEventRecorderFor("etcdcluster"),
		tlsCache:  tlsCache,
		config:    config,
		telemetry: telemetry,
	}
	return builder.ControllerManagedBy(mgr).
		For(&apiv1.EtcdCluster{}).
		Owns(&appsv1.Deployment{}).
		Owns(&cmv1.Certificate{}).
		Owns(&batchv1.CronJob{}).
		WithOptions(controller.Options{
			CacheSyncTimeout: 1 * time.Minute,
			RateLimiter:      rateLimiter,
		}).
		Complete(reconciler)
}

//+kubebuilder:rbac:groups=etcd.fleet.agoda.com,resources=etcdclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=etcd.fleet.agoda.com,resources=etcdclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=etcd.fleet.agoda.com,resources=etcdclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=services;configmaps;pods;serviceaccounts;events;secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=create;get;list;patch;update;watch;delete
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=create;get;list;patch;update;watch;delete
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cert-manager.io,resources=certificates;issuers,verbs=get;list;watch;create;patch;delete

// Reconcile implements the reconcile.Reconciler interface
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	cluster := &apiv1.EtcdCluster{}
	err := r.kcl.Get(ctx, req.NamespacedName, cluster)
	switch {
	case apierrors.IsNotFound(err):
		log.FromContext(ctx).V(3).Info("Cluster not found; skipping reconciliation", "name", req.NamespacedName)
		// Delete metrics for the cluster
		return reconcile.Result{}, r.telemetry.GetClusterMetrics().DeleteMetrics(ctx, req.NamespacedName)
	case err != nil:
		return reconcile.Result{}, err
	}

	// Get metrics for this cluster
	metrics, err := r.telemetry.GetClusterMetrics().GetOrCreateMetrics(req.NamespacedName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("get cluster metrics: %w", err)
	}

	return r.ReconcileCluster(ctx, cluster, metrics)
}

// ReconcileCluster handles the actual reconciliation logic for an EtcdCluster
func (r *Reconciler) ReconcileCluster(ctx context.Context, cluster *apiv1.EtcdCluster, metrics metrics.ClusterMetrics) (reconcile.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(3).Info("Reconciling cluster", "name", cluster.Name)

	if cluster.Status.Phase == apiv1.ClusterFailed {
		return reconcile.Result{}, nil
	}

	base := cluster.DeepCopy()

	// Set up deferred metrics recording that will happen on any return path
	defer metrics.Record(ctx, cluster)

	if cluster.Status.Phase == "" {
		cluster.Status.Phase = apiv1.ClusterBootstrap
	}

	if cluster.Status.SecretName == "" {
		cluster.Status.SecretName = cluster.Name + "-user-root"
	}

	if cluster.Status.Endpoint == "" {
		cluster.Status.Endpoint = fmt.Sprintf("https://%s.%s.svc.cluster.local:2379", cluster.Name, cluster.Namespace)
	}

	err := r.ReconcileResources(ctx, cluster)
	if err != nil {
		logger.V(3).Error(err, "reconcile resources")
		return reconcile.Result{}, fmt.Errorf("reconcile resources: %v", err)
	}

	err = r.ReconcileStatus(ctx, cluster)
	if err != nil {
		logger.V(3).Error(err, "reconcile status")
		return reconcile.Result{}, fmt.Errorf("reconcile status: %v", err)
	}

	// bail if status did not change
	if reflect.DeepEqual(base.Status, cluster.Status) {
		return reconcile.Result{}, nil
	}

	patch := client.MergeFrom(base)
	err = r.kcl.Status().Patch(ctx, cluster, patch)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("patch cluster status: %v", err)
	}

	logger.V(3).Info("patched cluster status")
	return reconcile.Result{}, nil
}

func (r *Reconciler) ReconcileResources(ctx context.Context, cluster *apiv1.EtcdCluster) error {
	// bail if resources were already reconciled
	if cluster.Status.ObservedGeneration == cluster.Generation {
		return nil
	}

	key := client.ObjectKeyFromObject(cluster)
	clusterLabel := apiv1.ClusterLabelValue(key)

	b := resources.NewBuilder(cluster).
		Label("app.kubernetes.io/managed-by", "etcd-operator").
		Label(apiv1.ClusterLabel, clusterLabel).
		Labels(cluster.Spec.CommonLabels).
		Annotations(cluster.Spec.CommonAnnotations)

	// pki
	b.CA("peer-ca")
	b.CA("server-ca")

	b.Certificate("user-root").
		Issuer(cluster.Name, "server-ca").
		Usages(cmv1.UsageClientAuth)

	etcdData, err := ETCDConfig(cluster, r.config)
	if err != nil {
		return fmt.Errorf("etcd config: %v", err)
	}

	serviceAccount := b.ServiceAccount()
	b.RoleBinding().
		ServiceAccountSubject(serviceAccount.ServiceAccount).
		ClusterRoleRef("etcd-sidecar")

	b.PodDisruptionBudget().
		Selector(apiv1.ClusterLabel, clusterLabel).
		MaxUnavailable(1)

	// cluster service
	b.Service().
		Selector(apiv1.ClusterLabel, clusterLabel).
		Selector(apiv1.LearnerLabel, "false").
		Port("etcd-client-ssl", 2379, 2379).
		Port("etcd-server-ssl", 2380, 2380).
		Headless(true)

	b.ConfigMap().
		Data("etcd.json", string(etcdData))

	// member deployment - bootstrap with single replica
	replicas := cluster.Spec.Replicas
	if cluster.Status.Phase == apiv1.ClusterBootstrap {
		replicas = 1
	}
	b.Deployment().
		Replicas(replicas).
		MaxUnavailable(0).
		MaxSurge(1).
		Selector(apiv1.ClusterLabel, clusterLabel).
		PodSpec(PodSpec(cluster, r.config))

	// defrag cron job
	defragSchedule := cluster.Spec.Defrag.Schedule
	if defragSchedule == "" {
		defragSchedule = DefragSchedule
	}

	b.CronJob("defrag").
		Suspend(!cluster.Spec.Defrag.Enabled).
		ConcurrencyPolicy(batchv1.ForbidConcurrent).
		Schedule(defragSchedule).
		TTL(JobTTL).
		ActiveDeadline(ActiveDeadline).
		PodSpec(DefragPodSpec(cluster, r.config))

	// backup cron job
	if cluster.Spec.Backup.SecretName != "" {
		schedule := cluster.Spec.Backup.Schedule
		if schedule == "" {
			schedule = BackupSchedule
		}

		b.CronJob("backup").
			Suspend(!cluster.Spec.Backup.Enabled).
			ConcurrencyPolicy(batchv1.ForbidConcurrent).
			Schedule(schedule).
			TTL(JobTTL).
			ActiveDeadline(ActiveDeadline).
			PodSpec(BackupPodSpec(cluster, r.config))
	}

	err = b.Apply(ctx, r.kcl)
	if err != nil {
		return fmt.Errorf("apply cluster resources: %w", err)
	}

	cluster.Status.ObservedGeneration = cluster.Generation

	return nil
}

func (r *Reconciler) ReconcileStatus(ctx context.Context, cluster *apiv1.EtcdCluster) error {
	key := client.ObjectKeyFromObject(cluster)
	deployment := &appsv1.Deployment{}
	err := r.kcl.Get(ctx, key, deployment)
	if err != nil {
		return fmt.Errorf("get cluster deployment: %v", err)
	}

	cluster.Status.Replicas = deployment.Status.Replicas
	cluster.Status.ReadyReplicas = deployment.Status.ReadyReplicas
	cluster.Status.UpdatedReplicas = deployment.Status.UpdatedReplicas

	switch {
	case cluster.Status.Phase == apiv1.ClusterRunning && cluster.Status.ReadyReplicas == 0:
		conditions.Upsert(&cluster.Status.Conditions, apiv1.ClusterCondition{
			Type:    apiv1.ClusterAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  "ClusterAvailable",
			Message: "cluster has no replicas running",
		})
		r.transition(cluster, apiv1.ClusterFailed)
		return nil
	case cluster.Status.ReadyReplicas == 0:
		return nil
	}

	key = client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      cluster.Status.SecretName,
	}
	tlsConfig, err := r.tlsCache.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("tls config: %v", err)
	}

	ctx, cancel := context.WithTimeoutCause(ctx, 30*time.Second, ErrOperationTimeout)
	defer cancel()

	// setup etcd client
	ecl, err := etcd.Connect(ctx, tlsConfig, cluster.Status.Endpoint)
	if err != nil {
		return fmt.Errorf("connect to cluster %v: %w", key, err)
	}
	defer func() {
		err = errors.Join(err, ecl.Close())
	}()

	resp, err := ecl.MemberList(ctx)
	if err != nil {
		conditions.Upsert(&cluster.Status.Conditions, apiv1.ClusterCondition{
			Type:    apiv1.ClusterAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  "NoConnection",
			Message: err.Error(),
		})
		return nil
	}

	cluster.Status.LearnerReplicas = 0
	cluster.Status.AvailableReplicas = 0

	// map member list response to status
	var wg sync.WaitGroup
	cluster.Status.Members = make([]apiv1.MemberStatus, len(resp.Members))
	for i, member := range resp.Members {
		status := apiv1.MemberStatus{
			ID:   apiv1.FormatMemberID(member.ID),
			Name: member.Name,
		}

		switch {
		case member.IsLearner:
			status.Role = apiv1.MemberRoleLearner
			cluster.Status.LearnerReplicas++
		case len(member.ClientURLs) != 0:
			status.Endpoint = member.ClientURLs[0]
		}

		if status.Endpoint == "" {
			cluster.Status.Members[i] = status
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeoutCause(ctx, 5*time.Second, ErrOperationTimeout)
			defer cancel()

			resp, err := ecl.Status(ctx, status.Endpoint)
			if err != nil {
				return
			}

			switch resp.Leader {
			case resp.Header.MemberId:
				status.Role = apiv1.MemberRoleLeader
			default:
				status.Role = apiv1.MemberRoleMember
			}

			status.Version = resp.Version
			status.Available = len(status.Errors) == 0
			status.Errors = resp.Errors

			status.Size = resource.NewQuantity(resp.DbSize, resource.DecimalSI)

			cluster.Status.Members[i] = status
		}()
	}
	wg.Wait()

	for _, member := range cluster.Status.Members {
		if member.Available {
			cluster.Status.AvailableReplicas++
		}
	}

	// sort members by role and name
	slices.SortFunc(cluster.Status.Members, func(l, r apiv1.MemberStatus) int {
		if l.Role == r.Role {
			return strings.Compare(l.Name, r.Name)
		}

		li := slices.Index(apiv1.MemberRoleOrder, l.Role)
		ri := slices.Index(apiv1.MemberRoleOrder, r.Role)

		return cmp.Compare(li, ri)
	})

	// bootstrap completed, reconcile resources
	if cluster.Status.Phase == apiv1.ClusterBootstrap && cluster.Status.AvailableReplicas >= 1 {
		cluster.Status.ObservedGeneration = 0
		r.transition(cluster, apiv1.ClusterRunning)
		return nil
	}

	quorum := cluster.Spec.Replicas/2 + 1
	switch {
	case cluster.Status.AvailableReplicas < quorum:
		conditions.Upsert(&cluster.Status.Conditions, apiv1.ClusterCondition{
			Type:    apiv1.ClusterAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  "NoQuorum",
			Message: fmt.Sprintf("%d replicas are less than required quorum %d", cluster.Status.AvailableReplicas, quorum),
		})
	case cluster.Status.AvailableReplicas < cluster.Spec.Replicas:
		conditions.Upsert(&cluster.Status.Conditions, apiv1.ClusterCondition{
			Type:    apiv1.ClusterAvailable,
			Status:  corev1.ConditionTrue,
			Reason:  "Degraded",
			Message: fmt.Sprintf("%d out of %d replicas are available", cluster.Status.AvailableReplicas, cluster.Spec.Replicas),
		})
	default:
		conditions.Upsert(&cluster.Status.Conditions, apiv1.ClusterCondition{
			Type:   apiv1.ClusterAvailable,
			Status: corev1.ConditionTrue,
			Reason: "ClusterAvailable",
		})
	}
	return nil
}

func (r *Reconciler) transition(cluster *apiv1.EtcdCluster, phase apiv1.ClusterPhase) {
	r.recorder.Eventf(cluster, corev1.EventTypeNormal, string(phase), fmt.Sprintf("Transition from %s to %s", cluster.Status.Phase, phase))
	cluster.Status.Phase = phase
}
