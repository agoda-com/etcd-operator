package v1

import (
	"strconv"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClusterLabel  = "etcd.fleet.agoda.com/cluster"
	MemberIDLabel = "etcd.fleet.agoda.com/member-id"
	LearnerLabel  = "etcd.fleet.agoda.com/learner"
)

const (
	RenewAtAnnotation = "etcd.fleet.agoda.com/renew-at"
)

func ClusterLabelValue(cluster client.ObjectKey) string {
	return strings.Join([]string{cluster.Name, cluster.Namespace}, ".")
}

func ParseMemberID(labels map[string]string) uint64 {
	if labels == nil {
		return 0
	}

	sv, ok := labels[MemberIDLabel]
	if !ok {
		return 0
	}

	v, _ := strconv.ParseUint(sv, 16, 64)
	return v
}

func FormatMemberID(id uint64) string {
	return strconv.FormatUint(id, 16)
}

func ParseCluster(labels map[string]string) (client.ObjectKey, bool) {
	if len(labels) == 0 {
		return client.ObjectKey{}, false
	}

	value, ok := labels[ClusterLabel]
	if !ok {
		return client.ObjectKey{}, false
	}

	name, namespace, ok := strings.Cut(value, ".")
	if !ok {
		return client.ObjectKey{}, false
	}

	return client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, true
}

func FormatRenewAt(renewAt time.Time) string {
	return renewAt.Format(time.RFC3339)
}

func ParseRenewAt(annotations map[string]string) time.Time {
	if len(annotations) == 0 {
		return time.Time{}
	}

	ts, _ := time.Parse(time.RFC3339, annotations[RenewAtAnnotation])
	return ts
}
