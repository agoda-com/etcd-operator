package sidecar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/agoda-com/etcd-operator/pkg/etcd"
)

func (s *Sidecar) Configure(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("configure")

	etcdConfig := &s.etcdConfig
	err := etcd.LoadConfig(s.config.ConfigFile, etcdConfig)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		err = etcd.LoadConfig(s.config.BaseConfigFile, etcdConfig)
		if err != nil {
			return fmt.Errorf("load base-config: %w", err)
		}
	case err != nil:
		return fmt.Errorf("load config: %w", err)
	}

	err = s.GenerateCredentials(ctx)
	if err != nil {
		return fmt.Errorf("generate credentials: %w", err)
	}

	etcdConfig.Name = s.pod.Name
	etcdConfig.AdvertiseClientURLs = fmt.Sprintf("https://%s:2379", s.pod.Status.PodIP)
	etcdConfig.InitialAdvertisePeerURLs = fmt.Sprintf("https://%s:2380", s.pod.Status.PodIP)

	pod := &s.pod
	patch := client.StrategicMergeFrom(pod.DeepCopy())
	labels := maps.Clone(pod.Labels)
	switch s.etcdConfig.InitialClusterState {
	// bootstrap
	case etcd.InitialStateNew:
		s.etcdConfig.InitialCluster = fmt.Sprintf("%s=https://%s:2380", s.pod.Name, s.pod.Status.PodIP)
		labels[apiv1.LearnerLabel] = "false"
	// add learner if we're joining existing cluster
	case etcd.InitialStateExisiting:
		member, err := s.AddLearner(ctx)
		if err != nil {
			return fmt.Errorf("add learner: %w", err)
		}

		labels[apiv1.LearnerLabel] = strconv.FormatBool(member.IsLearner)
		labels[apiv1.MemberIDLabel] = apiv1.FormatMemberID(member.ID)
	default:
		return fmt.Errorf("invalid initial-cluster-state %q", s.etcdConfig.InitialClusterState)
	}

	data, err := json.MarshalIndent(s.etcdConfig, "", "\t")
	if err != nil {
		return err
	}

	if !maps.Equal(labels, pod.Labels) {
		pod.Labels = labels
		err = s.kcl.Patch(ctx, pod, patch)
		if err != nil {
			return fmt.Errorf("patch pod: %w", err)
		}
	}

	err = os.WriteFile(s.config.ConfigFile, data, 0644)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	logger.Info("config written", "path", s.config.ConfigFile)

	return nil
}

func (s *Sidecar) AddLearner(ctx context.Context) (*etcdserverpb.Member, error) {
	logger := log.FromContext(ctx, "endpoint", s.config.Endpoint)

	tlsConfig := &s.tlsConfig
	ecl, err := etcd.Connect(ctx, tlsConfig, s.config.Endpoint)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, ecl.Close())
	}()

	// check if the member already exists
	members, err := ecl.MemberList(ctx)
	if err != nil {
		return nil, fmt.Errorf("query member list: %w", err)
	}
	i := slices.IndexFunc(members.Members, func(member *etcdserverpb.Member) bool {
		return member.Name == s.pod.Name
	})
	if i != -1 {
		return members.Members[i], nil
	}

	// retry loop till learner is admitted
	var member *etcdserverpb.Member
	err = wait.PollUntilContextCancel(ctx, s.config.Interval, true, func(ctx context.Context) (done bool, err error) {
		ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		resp, err := ecl.MemberAddAsLearner(ctx, []string{s.etcdConfig.InitialAdvertisePeerURLs})
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			logger.Info("add learner: timeout")
			return false, nil
		case errors.Is(err, rpctypes.ErrUnhealthy):
			logger.Info("add learner: waiting for cluster to be healthy")
			return false, nil
		case errors.Is(err, rpctypes.ErrTooManyLearners):
			logger.Info("add learner: waiting for cluster to allow learner to join")
			return false, nil
		case err != nil:
			return true, err
		}

		memberID := apiv1.FormatMemberID(resp.Header.MemberId)
		logger.Info("added learner", "id", memberID)

		endpoints := []string{
			s.pod.Name + "=" + s.etcdConfig.InitialAdvertisePeerURLs,
		}
		for _, member := range resp.Members {
			if member.Name != "" && len(member.PeerURLs) != 0 {
				endpoints = append(endpoints, member.Name+"="+member.PeerURLs[0])
			}
		}
		s.etcdConfig.InitialCluster = strings.Join(endpoints, ",")

		logger.Info("initial cluster", "endpoints", endpoints)

		member = resp.Member

		return true, nil
	})

	return member, err
}
