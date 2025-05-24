package metrics

import (
	"context"
	"sync"

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

	mtx       sync.Mutex
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
		Named("metrics").
		For(&apiv1.EtcdCluster{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 4,
		}).
		Complete(reconciler)
}

func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	key := req.NamespacedName
	cluster := &apiv1.EtcdCluster{}
	err := r.kcl.Get(ctx, key, cluster)
	switch {
	// passthrough error
	case client.IgnoreNotFound(err) != nil:
		return reconcile.Result{}, err
	// delete observer if cluster not found or deleted
	case err != nil || !cluster.DeletionTimestamp.IsZero():
		err = r.Delete(key)
		return reconcile.Result{}, err
	}

	observer, err := r.GetOrCreate(key)
	if err != nil {
		return reconcile.Result{}, err
	}

	observer.Update(cluster)

	return reconcile.Result{}, nil
}

func (r *Reconciler) GetOrCreate(key client.ObjectKey) (*Observer, error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	observer, ok := r.observers[key]
	if ok {
		return observer, nil
	}

	observer, err := Register(r.meter, key)
	if err != nil {
		return nil, err
	}

	r.observers[key] = observer

	return observer, nil
}

func (r *Reconciler) Delete(key client.ObjectKey) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	observer, ok := r.observers[key]
	if !ok {
		return nil
	}

	delete(r.observers, key)

	return observer.Unregister()
}
