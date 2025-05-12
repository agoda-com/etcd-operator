package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type ServiceBuilder struct{ *corev1.Service }

func (sb ServiceBuilder) Selector(label, value string) ServiceBuilder {
	sb.Spec.Selector[label] = value
	return sb
}

func (sb ServiceBuilder) Port(name string, port int32, target int) ServiceBuilder {
	sb.Spec.Ports = append(sb.Spec.Ports, corev1.ServicePort{
		Name:       name,
		Port:       port,
		TargetPort: intstr.FromInt(target),
	})
	return sb
}

func (sb ServiceBuilder) Headless(publishNotReady bool) ServiceBuilder {
	sb.Spec.ClusterIP = corev1.ClusterIPNone
	sb.Spec.PublishNotReadyAddresses = publishNotReady
	return sb
}

func (b *Builder) Service(name ...string) ServiceBuilder {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.name(name),
			Namespace: b.owner.GetNamespace(),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{},
		},
	}
	b.add(svc)

	return ServiceBuilder{svc}
}
