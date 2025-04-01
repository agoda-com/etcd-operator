package sidecar

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	etcdv3 "go.etcd.io/etcd/client/v3"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
)

func (s *Sidecar) Sync(ctx context.Context, ecl *etcdv3.Client) error {
	status, err := ecl.Status(ctx, "https://127.0.0.1:2379")
	if err != nil {
		return fmt.Errorf("query status: %w", err)
	}

	pod := &s.pod
	labels := maps.Clone(pod.Labels)
	labels[apiv1.LearnerLabel] = strconv.FormatBool(status.IsLearner)
	labels[apiv1.MemberIDLabel] = strconv.FormatUint(status.Header.MemberId, 16)

	// update labels if changed
	if !maps.Equal(labels, pod.Labels) {
		patch := client.StrategicMergeFrom(s.pod.DeepCopy())
		pod.Labels = labels
		err = s.kcl.Patch(ctx, pod, patch)
		if err != nil {
			return fmt.Errorf("patch pod labels: %w", err)
		}
	}

	// bail if not leader
	if status.Leader != status.Header.MemberId {
		return nil
	}

	members, err := ecl.MemberList(ctx)
	if err != nil {
		return fmt.Errorf("query member list: %w", err)
	}

	err = s.Prune(ctx, ecl, members)
	if err != nil {
		return fmt.Errorf("prune: %w", err)
	}

	err = s.Promote(ctx, ecl, members)
	if err != nil {
		return fmt.Errorf("promote: %w", err)
	}

	return nil
}

// Prune removes members which do not have associated pod
func (s *Sidecar) Prune(ctx context.Context, ecl etcdv3.Cluster, members *etcdv3.MemberListResponse) error {
	if !s.config.Prune {
		return nil
	}

	logger := log.FromContext(ctx)

	pods := &corev1.PodList{}
	err := s.kcl.List(ctx, pods, client.InNamespace(s.config.Namespace), client.MatchingLabels{
		apiv1.ClusterLabel: s.pod.Labels[apiv1.ClusterLabel],
	})
	switch {
	case apierrors.IsTooManyRequests(err):
		logger.V(3).Info("prune: too many requests")
		return nil
	case err != nil:
		return err
	}

	// find member without pod
	i := slices.IndexFunc(members.Members, func(member *etcdserverpb.Member) bool {
		// skip unstarted
		if member.Name == "" {
			return false
		}

		return !slices.ContainsFunc(pods.Items, func(pod corev1.Pod) bool {
			return pod.Name == member.Name || apiv1.ParseMemberID(pod.Labels) == member.ID
		})
	})
	if i == -1 {
		return nil
	}

	member := members.Members[i]
	_, err = ecl.MemberRemove(ctx, member.ID)
	switch {
	case err != nil && !errors.Is(err, rpctypes.ErrMemberNotFound):
		return err
	case err == nil:
		logger.Info("removed member", "id", apiv1.FormatMemberID(member.ID))
	}

	return nil
}

func (s *Sidecar) Promote(ctx context.Context, ecl etcdv3.Cluster, members *etcdv3.MemberListResponse) error {
	i := slices.IndexFunc(members.Members, func(member *etcdserverpb.Member) bool {
		return member.IsLearner && member.Name != ""
	})
	if i == -1 {
		return nil
	}

	learner := members.Members[i]
	logger := log.FromContext(ctx, "learner", learner.Name, "id", apiv1.FormatMemberID(learner.ID))

	_, err := ecl.MemberPromote(ctx, learner.ID)
	switch {
	case errors.Is(err, rpctypes.ErrMemberLearnerNotReady):
		logger.Info("waiting for learner to catch up")
		return nil
	case err != nil:
		return fmt.Errorf("promote member: %w", err)
	}

	logger.Info("promoted learner")
	return nil
}
