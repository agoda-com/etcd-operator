package resources

import (
	"maps"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

type DeploymentBuilder struct{ *appsv1.Deployment }

func (b *Builder) Deployment(names ...string) DeploymentBuilder {
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.name(names),
			Namespace: b.owner.GetNamespace(),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
			},
		},
	}
	b.add(d)

	return DeploymentBuilder{
		Deployment: d,
	}
}

func (b DeploymentBuilder) MaxUnavailable(unavailable int32) DeploymentBuilder {
	if b.Spec.Strategy.RollingUpdate == nil {
		b.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{}
	}

	b.Spec.Strategy.RollingUpdate.MaxUnavailable = ptr.To(intstr.FromInt32(unavailable))

	return b
}

func (b DeploymentBuilder) MaxSurge(surge int32) DeploymentBuilder {
	if b.Spec.Strategy.RollingUpdate == nil {
		b.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{}
	}

	b.Spec.Strategy.RollingUpdate.MaxSurge = ptr.To(intstr.FromInt32(surge))

	return b
}

func (b DeploymentBuilder) Replicas(replicas int32) DeploymentBuilder {
	b.Spec.Replicas = ptr.To(replicas)

	return b
}

func (b DeploymentBuilder) Selector(label, value string) DeploymentBuilder {
	b.Spec.Selector.MatchLabels[label] = value
	b.Spec.Template.Labels[label] = value

	return b
}

func (b DeploymentBuilder) PodLabel(key, value string) DeploymentBuilder {
	b.Spec.Template.Labels[key] = value

	return b
}

func (b DeploymentBuilder) PodAnnotations(annotations map[string]string) DeploymentBuilder {
	if b.Spec.Template.Annotations == nil {
		b.Spec.Template.Annotations = make(map[string]string)
	}

	maps.Copy(b.Spec.Template.Annotations, annotations)

	return b
}

func (b DeploymentBuilder) PodSpec(spec corev1.PodSpec) DeploymentBuilder {
	b.Spec.Template.Spec = spec

	return b
}
