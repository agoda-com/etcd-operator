package metrics

import (
	"context"
	"fmt"
	"sync/atomic"

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

func Register(meter metric.Meter, cluster *apiv1.EtcdCluster) (*Observer, error) {
	observer := &Observer{
		attributes: attribute.NewSet(
			attribute.String("fleet.etcd.cluster.namespace", cluster.Namespace),
			attribute.String("fleet.etcd.cluster.name", cluster.Name),
		),
	}

	observer.cluster.Store(cluster)

	gauges := []struct {
		name        string
		description string
		dest        *metric.Int64ObservableGauge
	}{
		{
			name:        "fleet.etcd.cluster.desired_replicas",
			description: "Number of desired replicas",
			dest:        &observer.desiredReplicas,
		},
		{
			name:        "fleet.etcd.cluster.replicas",
			description: "Number of replicas",
			dest:        &observer.replicas,
		},
		{
			name:        "fleet.etcd.cluster.ready_replicas",
			description: "Number of ready replicas",
			dest:        &observer.readyReplicas,
		},
		{
			name:        "fleet.etcd.cluster.updated_replicas",
			description: "Number of updated replicas",
			dest:        &observer.updatedReplicas,
		},
		{
			name:        "fleet.etcd.cluster.available_replicas",
			description: "Number of available replicas",
			dest:        &observer.availableReplicas,
		},
		{
			name:        "fleet.etcd.cluster.learner_replicas",
			description: "Number of learner replicas",
			dest:        &observer.learnerReplicas,
		},
		{
			name:        "fleet.etcd.cluster.backup.last_schedule_time",
			description: "Last backup schedule time",
			dest:        &observer.backupLastScheduleTime,
		},
		{
			name:        "fleet.etcd.cluster.backup.last_successful_time",
			description: "Last backup successful time",
			dest:        &observer.backupLastSuccessfulTime,
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

	registration, err := meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		cluster := observer.cluster.Load()
		if cluster == nil {
			return nil
		}

		opts := []metric.ObserveOption{
			metric.WithAttributeSet(observer.attributes),
		}

		o.ObserveInt64(observer.desiredReplicas, int64(cluster.Spec.Replicas), opts...)
		o.ObserveInt64(observer.replicas, int64(cluster.Status.Replicas), opts...)
		o.ObserveInt64(observer.readyReplicas, int64(cluster.Status.ReadyReplicas), opts...)
		o.ObserveInt64(observer.updatedReplicas, int64(cluster.Status.UpdatedReplicas), opts...)
		o.ObserveInt64(observer.availableReplicas, int64(cluster.Status.AvailableReplicas), opts...)
		o.ObserveInt64(observer.learnerReplicas, int64(cluster.Status.LearnerReplicas), opts...)

		if cluster.Status.Backup != nil && cluster.Status.Backup.LastScheduleTime != nil {
			o.ObserveInt64(observer.backupLastScheduleTime, cluster.Status.Backup.LastScheduleTime.Unix(), opts...)
		}

		if cluster.Status.Backup != nil && cluster.Status.Backup.LastSuccessfulTime != nil {
			o.ObserveInt64(observer.backupLastSuccessfulTime, cluster.Status.Backup.LastSuccessfulTime.Unix(), opts...)
		}

		return nil
	}, observables...)
	if err != nil {
		return nil, fmt.Errorf("register callback: %w", err)
	}

	observer.registration = registration

	return observer, nil
}

func (o *Observer) Update(cluster *apiv1.EtcdCluster) {
	o.cluster.Store(cluster)
}

func (o *Observer) Unregister() error {
	o.cluster.Store(nil)
	return o.registration.Unregister()
}
