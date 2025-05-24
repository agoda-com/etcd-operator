package metrics

import (
	"context"
	"fmt"
	"sync/atomic"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
)

type Observer struct {
	cluster      atomic.Pointer[apiv1.EtcdCluster]
	attributes   attribute.Set
	registration metric.Registration

	desiredReplicas, replicas, readyReplicas, updatedReplicas, availableReplicas, learnerReplicas metric.Int64ObservableGauge
	backupLastScheduleTime, backupLastSuccessfulTime                                              metric.Int64ObservableGauge
}

func Register(meter metric.Meter, key client.ObjectKey) (*Observer, error) {
	o := &Observer{
		attributes: attribute.NewSet(
			attribute.String("fleet.etcd.cluster.namespace", key.Namespace),
			attribute.String("fleet.etcd.cluster.name", key.Name),
		),
	}

	gauges := []struct {
		name        string
		description string
		dest        *metric.Int64ObservableGauge
	}{
		{
			name:        "fleet.etcd.cluster.desired_replicas",
			description: "Number of desired replicas",
			dest:        &o.desiredReplicas,
		},
		{
			name:        "fleet.etcd.cluster.replicas",
			description: "Number of replicas",
			dest:        &o.replicas,
		},
		{
			name:        "fleet.etcd.cluster.ready_replicas",
			description: "Number of ready replicas",
			dest:        &o.readyReplicas,
		},
		{
			name:        "fleet.etcd.cluster.updated_replicas",
			description: "Number of updated replicas",
			dest:        &o.updatedReplicas,
		},
		{
			name:        "fleet.etcd.cluster.available_replicas",
			description: "Number of available replicas",
			dest:        &o.availableReplicas,
		},
		{
			name:        "fleet.etcd.cluster.learner_replicas",
			description: "Number of learner replicas",
			dest:        &o.learnerReplicas,
		},
		{
			name:        "fleet.etcd.cluster.backup.last_schedule_time",
			description: "Last backup schedule time",
			dest:        &o.backupLastScheduleTime,
		},
		{
			name:        "fleet.etcd.cluster.backup.last_successful_time",
			description: "Last backup successful time",
			dest:        &o.backupLastSuccessfulTime,
		},
	}

	var observables []metric.Observable
	for _, gauge := range gauges {
		observable, err := meter.Int64ObservableGauge(gauge.name, metric.WithDescription(gauge.description))
		if err != nil {
			return nil, fmt.Errorf("gauge %q: %w", gauge.name, err)
		}

		*gauge.dest = observable
		observables = append(observables, observable)
	}

	registration, err := meter.RegisterCallback(o.Observe, observables...)
	if err != nil {
		return nil, fmt.Errorf("register callback: %w", err)
	}

	o.registration = registration

	return o, nil
}

func (o *Observer) Observe(ctx context.Context, observer metric.Observer) error {
	cluster := o.cluster.Load()
	if cluster == nil {
		return nil
	}

	opts := []metric.ObserveOption{
		metric.WithAttributeSet(o.attributes),
	}

	observer.ObserveInt64(o.desiredReplicas, int64(cluster.Spec.Replicas), opts...)
	observer.ObserveInt64(o.replicas, int64(cluster.Status.Replicas), opts...)
	observer.ObserveInt64(o.readyReplicas, int64(cluster.Status.ReadyReplicas), opts...)
	observer.ObserveInt64(o.updatedReplicas, int64(cluster.Status.UpdatedReplicas), opts...)
	observer.ObserveInt64(o.availableReplicas, int64(cluster.Status.AvailableReplicas), opts...)
	observer.ObserveInt64(o.learnerReplicas, int64(cluster.Status.LearnerReplicas), opts...)

	if cluster.Status.Backup != nil && cluster.Status.Backup.LastScheduleTime != nil {
		observer.ObserveInt64(o.backupLastScheduleTime, cluster.Status.Backup.LastScheduleTime.Unix(), opts...)
	}

	if cluster.Status.Backup != nil && cluster.Status.Backup.LastSuccessfulTime != nil {
		observer.ObserveInt64(o.backupLastSuccessfulTime, cluster.Status.Backup.LastSuccessfulTime.Unix(), opts...)
	}

	return nil
}

func (o *Observer) Update(cluster *apiv1.EtcdCluster) {
	o.cluster.Store(cluster)
}

func (o *Observer) Unregister() error {
	o.cluster.Store(nil)
	return o.registration.Unregister()
}
