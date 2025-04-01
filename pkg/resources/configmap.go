package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMapBuilder struct {
	*corev1.ConfigMap
}

func (b *Builder) ConfigMap(names ...string) ConfigMapBuilder {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.name(names),
			Namespace: b.owner.GetNamespace(),
		},
	}
	b.add(cm)

	return ConfigMapBuilder{
		ConfigMap: cm,
	}
}

func (b ConfigMapBuilder) Data(key, value string) ConfigMapBuilder {
	if b.ConfigMap.Data == nil {
		b.ConfigMap.Data = map[string]string{}
	}

	b.ConfigMap.Data[key] = value

	return b
}
