package e2e

import (
	"slices"
	"strings"
	"testing"
	"time"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/agoda-com/etcd-operator/pkg/conditions"
)

func TestCluster(t *testing.T) {
	ctx := t.Context()
	kcl := kubeClient(t, client.Options{})

	cluster := createCluster(t, kcl, 3*time.Minute, apiv1.EtcdClusterSpec{
		Version:   "v3.5.14",
		Replicas:  3,
		Resources: resources,
	})

	t.Run("scale", func(t *testing.T) {
		scale(t, kcl, cluster, 5)
		poll(t, kcl, cluster, 2*time.Minute, available)

		t.Logf("cluster %q scaled up to 5 members", cluster.Name)

		scale(t, kcl, cluster, 3)
		poll(t, kcl, cluster, time.Minute, available)

		t.Logf("cluster %q scaled down to 3 members", cluster.Name)
	})

	t.Run("evict pod", func(t *testing.T) {
		members := cluster.Status.Members
		if len(members) == 0 {
			t.Fatalf("cluster %q has no members", cluster.Name)
		}

		name := members[0].Name
		key := client.ObjectKey{
			Namespace: cluster.Namespace,
			Name:      name,
		}
		pod := &corev1.Pod{}
		err := kcl.Get(ctx, key, pod)
		if err != nil {
			t.Fatal("get pod:", err)
		}

		err = kcl.SubResource("eviction").Create(ctx, pod, &policyv1.Eviction{})
		if err != nil {
			t.Fatal("evict pod:", err)
		}
		t.Log("evict pod", key)

		poll(t, kcl, cluster, 2*time.Minute, func(cluster *apiv1.EtcdCluster) bool {
			if !available(cluster) {
				return false
			}

			return !slices.ContainsFunc(cluster.Status.Members, func(member apiv1.MemberStatus) bool {
				return member.Name == name
			})
		})

		t.Logf("cluster %q recovered from pod termination", cluster.Name)
	})

	t.Run("upgrade", func(t *testing.T) {
		version := "v3.5.18"
		t.Logf("upgrading cluster %q from version %q to %q", cluster.Name, cluster.Spec.Version, version)

		patch := client.MergeFrom(cluster.DeepCopy())
		cluster.Spec.Version = version
		err := kcl.Patch(ctx, cluster, patch)
		if err != nil {
			t.Error("patch cluster:", err)
		}

		version = strings.TrimPrefix(version, "v")
		poll(t, kcl, cluster, 5*time.Minute, func(cluster *apiv1.EtcdCluster) bool {
			if !conditions.StatusTrue(cluster.Status.Conditions, apiv1.ClusterAvailable) {
				return false
			}

			return !slices.ContainsFunc(cluster.Status.Members, func(member apiv1.MemberStatus) bool {
				return member.Version != version
			})
		})

		t.Logf("cluster %q upgraded to %q", cluster.Name, version)
	})
}
