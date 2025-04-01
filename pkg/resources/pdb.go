package resources

import (
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type PdbBuilder struct{ *policyv1.PodDisruptionBudget }

func (b *Builder) PodDisruptionBudget(name ...string) *PdbBuilder {
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.name(name),
			Namespace: b.owner.GetNamespace(),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{},
			},
		},
	}
	b.children = append(b.children, pdb)
	return &PdbBuilder{pdb}
}

func (b *PdbBuilder) MaxUnavailable(v int32) *PdbBuilder {
	b.Spec.MaxUnavailable = &intstr.IntOrString{IntVal: v}
	return b
}

func (b *PdbBuilder) Selector(label, value string) *PdbBuilder {
	b.Spec.Selector.MatchLabels[label] = value
	return b
}

func (b *PdbBuilder) UnhealthyPodEvictionPolicy(v policyv1.UnhealthyPodEvictionPolicyType) *PdbBuilder {
	b.Spec.UnhealthyPodEvictionPolicy = &v
	return b
}
