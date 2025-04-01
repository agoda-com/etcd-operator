package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodBuilder struct {
	*corev1.Pod
}

func (b *Builder) Pod(names ...string) PodBuilder {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.name(names),
			Namespace: b.owner.GetNamespace(),
		},
	}
	b.add(pod)

	return PodBuilder{pod}
}

func (b PodBuilder) PodSpec(spec corev1.PodSpec) PodBuilder {
	b.Spec = spec
	return b
}

func (b PodBuilder) InitContainer(container corev1.Container) PodBuilder {
	b.Spec.InitContainers = append(b.Spec.InitContainers, container)
	return b
}
