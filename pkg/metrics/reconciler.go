package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
)

type Reconciler struct {
	kcl   client.Client
	meter metric.Meter

	// no lock required - workqueue is stingy preventing concurrent reconciles of same resource
	// https://pkg.go.dev/k8s.io/client-go/util/workqueue
	observers map[client.ObjectKey]*Observer
}

func SetupWithManager(mgr manager.Manager, meterProvider metric.MeterProvider) error {
	meter := meterProvider.Meter("github.com/agoda-com/etcd-operator/pkg/metrics")
	reconciler := &Reconciler{
		kcl:       mgr.GetClient(),
		meter:     meter,
		observers: map[client.ObjectKey]*Observer{},
	}

	return builder.ControllerManagedBy(mgr).
		For(&apiv1.EtcdCluster{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 4,
		}).
		Complete(reconciler)
}

func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	observer := r.observers[req.NamespacedName]

	cluster := &apiv1.EtcdCluster{}
	err := r.kcl.Get(ctx, req.NamespacedName, cluster)
	switch {
	// passthrough error
	case client.IgnoreNotFound(err) != nil:
		return reconcile.Result{}, err
	// unregister if cluster not found or deleted
	case (err != nil || !cluster.DeletionTimestamp.IsZero()) && observer != nil:
		err = observer.Unregister()
		return reconcile.Result{}, err
	// register observer
	case observer == nil:
		observer, err = Register(r.meter, cluster)
		if err != nil {
			return reconcile.Result{}, err
		}
		r.observers[req.NamespacedName] = observer
	}

	observer.Update(cluster)

	return reconcile.Result{}, nil
}
