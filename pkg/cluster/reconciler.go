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
	"github.com/agoda-com/etcd-operator/pkg/resources"
)

var ErrOperationTimeout = errors.New("operation timeout")

type Reconciler struct {
	kcl      client.Client
	recorder record.EventRecorder
	tlsCache *etcd.TLSCache
	config   Config
}

// SetupWithManager creates a new manager
func SetupWithManager(mgr manager.Manager, tlsCache *etcd.TLSCache, config Config) error {
	rateLimiter := workqueue.NewTypedItemFastSlowRateLimiter[reconcile.Request](1*time.Second, 5*time.Second, 10)
	reconciler := &Reconciler{
		kcl:      mgr.GetClient(),
		recorder: mgr.GetEventRecorderFor("etcdcluster"),
		tlsCache: tlsCache,
		config:   config,
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
		Complete(reconcile.AsReconciler(mgr.GetClient(), reconciler))
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

// ReconcileCluster handles the actual reconciliation logic for an EtcdCluster
func (r *Reconciler) Reconcile(ctx context.Context, cluster *apiv1.EtcdCluster) (reconcile.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(3).Info("Reconciling cluster", "name", cluster.Name)

	if cluster.Status.Phase == apiv1.ClusterFailed {
		return reconcile.Result{}, nil
	}

	base := cluster.DeepCopy()

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
	switch {
	case client.IgnoreNotFound(err) != nil:
		return reconcile.Result{}, fmt.Errorf("patch cluster status: %v", err)
	case err == nil:
		logger.V(3).Info("patched cluster status")
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) ReconcileResources(ctx context.Context, cluster *apiv1.EtcdCluster) error {
	// bail if paused or resources were already reconciled
	if cluster.Spec.Pause || cluster.Status.ObservedGeneration == cluster.Generation {
		return nil
	}

	key := client.ObjectKeyFromObject(cluster)
	clusterLabel := apiv1.ClusterLabelValue(key)

	b := resources.NewBuilder(cluster).
		Label("app.kubernetes.io/managed-by", "etcd-operator").
		Label(apiv1.ClusterLabel, clusterLabel)

	// pki
	b.CA("peer-ca")
	b.CA("server-ca")

	secretLabels := map[string]string{
		apiv1.ClusterLabel: clusterLabel,
	}

	b.Certificate("user-root").
		Issuer(cluster.Name, "server-ca").
		Usages(cmv1.UsageClientAuth).
		SecretLabels(secretLabels)

	deployment, err := Deployment(ctx, b, cluster, r.config)
	if deployment == nil {
		return err
	}

	DefragCronJob(b, cluster, r.config)
	BackupCronJob(b, cluster, r.config)

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
	switch {
	// ignore deployment not found
	case apierrors.IsNotFound(err):
	case err != nil:
		return fmt.Errorf("get cluster deployment: %w", err)
	default:
		cluster.Status.UpdatedReplicas = deployment.Status.UpdatedReplicas
	}

	key = client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-backup",
	}
	cronJob := &batchv1.CronJob{}
	err = r.kcl.Get(ctx, key, cronJob)
	switch {
	// cronjob not found - reset backup status
	case apierrors.IsNotFound(err):
		cluster.Status.Backup = nil
	case err != nil:
		return fmt.Errorf("get backup cronjob: %w", err)
	default:
		cluster.Status.Backup = &apiv1.BackupStatus{
			LastScheduleTime:   cronJob.Status.LastScheduleTime,
			LastSuccessfulTime: cronJob.Status.LastSuccessfulTime,
		}
	}

	pods := &corev1.PodList{}
	err = r.kcl.List(ctx, pods, client.MatchingLabels{
		apiv1.ClusterLabel: apiv1.ClusterLabelValue(client.ObjectKeyFromObject(cluster)),
	})
	if err != nil {
		return fmt.Errorf("get cluster pods: %w", err)
	}

	cluster.Status.Replicas = int32(len(pods.Items))
	cluster.Status.ReadyReplicas = 0

	for _, pod := range pods.Items {
		ready := slices.ContainsFunc(pod.Status.Conditions, func(cond corev1.PodCondition) bool {
			return cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue
		})
		if ready {
			cluster.Status.ReadyReplicas++
		}
	}

	switch {
	// wait for bootstrap replica to be available
	case cluster.Status.Phase == apiv1.ClusterBootstrap && cluster.Status.ReadyReplicas == 0:
		return nil
	case cluster.Status.Phase == apiv1.ClusterRunning && cluster.Status.ReadyReplicas == 0:
		conditions.Upsert(&cluster.Status.Conditions, apiv1.ClusterCondition{
			Type:    apiv1.ClusterAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  "ClusterAvailable",
			Message: "cluster has no replicas running",
		})
		r.transition(cluster, apiv1.ClusterFailed)
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

	quorum := cluster.Status.Replicas/2 + 1
	switch {
	case cluster.Status.AvailableReplicas < quorum:
		conditions.Upsert(&cluster.Status.Conditions, apiv1.ClusterCondition{
			Type:    apiv1.ClusterAvailable,
			Status:  corev1.ConditionFalse,
			Reason:  "NoQuorum",
			Message: fmt.Sprintf("%d replicas are less than required quorum %d", cluster.Status.AvailableReplicas, quorum),
		})
	case cluster.Status.AvailableReplicas < cluster.Status.Replicas:
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
