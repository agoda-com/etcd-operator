package resources

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Builder struct {
	owner       client.Object
	children    []client.Object
	delete      []client.Object
	prefix      string
	labels      map[string]string
	annotations map[string]string
}

func NewBuilder(owner client.Object) *Builder {
	return &Builder{
		owner:       owner,
		prefix:      owner.GetName(),
		labels:      map[string]string{},
		annotations: map[string]string{},
	}
}

func (b *Builder) NoPrefix() *Builder {
	b.prefix = ""
	return b
}

func (b *Builder) Prefix(prefix string) *Builder {
	b.prefix = prefix
	return b
}

func (b *Builder) Label(key, value string) *Builder {
	b.labels[key] = value
	return b
}

func (b *Builder) Labels(labels map[string]string) *Builder {
	maps.Copy(b.labels, labels)
	return b
}

func (b *Builder) Annotation(key, value string) *Builder {
	b.annotations[key] = value
	return b
}

func (b *Builder) Annotations(annotations map[string]string) *Builder {
	maps.Copy(b.annotations, annotations)
	return b
}

func (b *Builder) Delete(obj client.Object) *Builder {
	b.delete = append(b.delete, obj)
	return b
}

func (b *Builder) Build(scheme *runtime.Scheme) error {
	for _, child := range b.children {
		if child.GetNamespace() != "" {
			err := controllerutil.SetControllerReference(b.owner, child, scheme)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *Builder) Apply(ctx context.Context, cl client.Client) error {
	err := b.Build(cl.Scheme())
	if err != nil {
		return err
	}

	for _, child := range b.children {
		err := NormalizeGVK(child, cl.Scheme())
		if err != nil {
			return fmt.Errorf("normalize %T: %w", child, err)
		}

		err = cl.Patch(ctx, child, client.Apply, client.FieldOwner("etcd-operator"), client.ForceOwnership)
		if err != nil {
			return fmt.Errorf("patch %T: %w", child, err)
		}
	}

	for _, obj := range b.delete {
		err := cl.Delete(ctx, obj)
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("delete %T: %w", obj, err)
		}
	}

	return nil
}

func (b *Builder) add(child client.Object) {
	labels := maps.Clone(b.labels)
	maps.Copy(labels, child.GetLabels())
	child.SetLabels(labels)

	annotations := maps.Clone(b.annotations)
	maps.Copy(annotations, child.GetAnnotations())
	child.SetAnnotations(annotations)

	b.children = append(b.children, child)
}

func (b *Builder) name(names []string) string {
	if b.prefix != "" {
		return strings.Join(append([]string{b.prefix}, names...), "-")
	}
	return strings.Join(names, "-")
}

func NormalizeGVK(object client.Object, scheme *runtime.Scheme) error {
	if object == nil {
		return nil
	}

	kind := object.GetObjectKind()
	if !kind.GroupVersionKind().Empty() {
		return nil
	}

	gvk, err := apiutil.GVKForObject(object, scheme)
	if err != nil {
		return err
	}
	kind.SetGroupVersionKind(gvk)

	return nil
}

func DefaultLabels(obj client.Object, defaults map[string]string) {
	switch {
	case len(defaults) == 0:
		return
	case len(obj.GetLabels()) == 0:
		obj.SetLabels(defaults)
	case len(obj.GetLabels()) != 0:
		merged := maps.Clone(defaults)
		maps.Copy(merged, obj.GetLabels())
		obj.SetLabels(merged)
	}
}

func GetLabel(obj client.Object, name string) string {
	labels := obj.GetLabels()
	if labels == nil {
		return ""
	}

	return labels[name]
}

func RemoveOwnerReference(owner metav1.Object, controlled metav1.Object) {
	refs := controlled.GetOwnerReferences()
	uid := owner.GetUID()
	// noop
	if len(refs) == 0 || uid == "" {
		return
	}

	i := slices.IndexFunc(refs, func(ref metav1.OwnerReference) bool {
		return ref.UID == uid
	})
	// ref not found
	if i == -1 {
		return
	}

	controlled.SetOwnerReferences(slices.Delete(refs, i, i+1))
}
