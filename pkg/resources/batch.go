package resources

import (
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type JobBuilder struct{ *batchv1.Job }

type CronJobBuilder struct{ *batchv1.CronJob }

func (b *Builder) Job(names ...string) JobBuilder {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.name(names),
			Namespace: b.owner.GetNamespace(),
		},
	}
	b.add(job)

	return JobBuilder{job}
}

func (b *Builder) CronJob(names ...string) CronJobBuilder {
	job := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.name(names),
			Namespace: b.owner.GetNamespace(),
		},
		Spec: batchv1.CronJobSpec{
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: b.labels,
				},
			},
		},
	}
	b.add(job)

	return CronJobBuilder{job}
}

func (b JobBuilder) PodSpec(spec corev1.PodSpec) JobBuilder {
	b.Spec.Template.Spec = spec
	return b
}

func (b JobBuilder) TTL(ttl time.Duration) JobBuilder {
	b.Spec.TTLSecondsAfterFinished = ptr.To(int32(ttl.Seconds()))
	return b
}

func (b CronJobBuilder) ConcurrencyPolicy(policy batchv1.ConcurrencyPolicy) CronJobBuilder {
	b.Spec.ConcurrencyPolicy = policy
	return b
}

func (b CronJobBuilder) Schedule(schedule string) CronJobBuilder {
	b.Spec.Schedule = schedule
	return b
}

func (b CronJobBuilder) Suspend(suspend bool) CronJobBuilder {
	b.Spec.Suspend = ptr.To(suspend)
	return b
}

func (b CronJobBuilder) PodSpec(spec corev1.PodSpec) CronJobBuilder {
	b.Spec.JobTemplate.Spec.Template.Spec = spec
	return b
}

func (b CronJobBuilder) TTL(ttl time.Duration) CronJobBuilder {
	b.Spec.JobTemplate.Spec.TTLSecondsAfterFinished = ptr.To(int32(ttl.Seconds()))
	return b
}

func (b CronJobBuilder) ActiveDeadline(duration time.Duration) CronJobBuilder {
	b.Spec.JobTemplate.Spec.ActiveDeadlineSeconds = ptr.To(int64(duration.Seconds()))
	return b
}
