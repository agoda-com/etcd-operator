/*
Copyright 2024 Agoda.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true

// EtcdClusterList contains a list of EtcdCluster
type EtcdClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EtcdCluster `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=`.spec.replicas`,statuspath=`.status.replicas`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Available",type=integer,JSONPath=`.status.availableReplicas`
// +kubebuilder:printcolumn:name="Learners",type=integer,JSONPath=`.status.learnerReplicas`
// +kubebuilder:printcolumn:name="Updated",type=integer,JSONPath=`.status.updatedReplicas`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// EtcdCluster is the Schema for the etcdclusters API
type EtcdCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EtcdClusterSpec   `json:"spec,omitempty"`
	Status EtcdClusterStatus `json:"status,omitempty"`
}

// EtcdClusterSpec defines the desired state of EtcdCluster
type EtcdClusterSpec struct {
	Pause bool `json:"pause,omitempty"`

	// Replicas
	//
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Version
	//
	// +kubebuilder:default=v3.5.7
	Version string `json:"version"`

	PodTemplate *PodTemplate `json:"podTemplate,omitempty"`

	Restore *RestoreSpec `json:"restore,omitempty"`
	Backup  *BackupSpec  `json:"backup,omitempty"`
	Defrag  *DefragSpec  `json:"defrag,omitempty"`

	// Compute Resources required by each member of cluster.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	Resources corev1.ResourceList `json:"resources,omitempty"`
}

type PodTemplate struct {
	// Labels
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations
	Annotations map[string]string `json:"annotations,omitempty"`
}

// BackupSpec defines the configuration to backup cluster to
type BackupSpec struct {
	Suspend  bool   `json:"suspend,omitempty"`
	Schedule string `json:"schedule,omitempty"`
}

// RestoreSpec defines the configuration to restore cluster from
type RestoreSpec struct {
	Prefix *string `json:"prefix,omitempty"`
	Key    *string `json:"key,omitempty"`
}

// DefragSpec defines the configuration for automated cluster defrag
type DefragSpec struct {
	Suspend  *bool              `json:"suspend,omitempty"`
	Schedule *string            `json:"schedule,omitempty"`
	Size     *resource.Quantity `json:"size,omitempty"`
	// +kubebuilder:validation:Pattern=`^(1\.0|0\.[0-9]+)$`
	Ratio *string `json:"ratio,omitempty"`
}

// EtcdClusterStatus defines the observed state of EtcdCluster
type EtcdClusterStatus struct {
	// Lifecycle phase
	Phase ClusterPhase `json:"phase,omitempty"`

	// ObservedGeneration
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Backup
	Backup *BackupStatus `json:"backup,omitempty"`

	// Replicas is the number of non-terminated members.
	// +kubebuilder:default=0
	Replicas int32 `json:"replicas"`

	// ReadyReplicas is the number of ready member pods.
	// +kubebuilder:default=0
	ReadyReplicas int32 `json:"readyReplicas"`

	// AvailableReplicas is the number of fully provisioned members.
	// +kubebuilder:default=0
	AvailableReplicas int32 `json:"availableReplicas"`

	// LearnerReplicas
	// +kubebuilder:default=0
	LearnerReplicas int32 `json:"learnerReplicas"`

	// UpdatedReplicas is the number of members that are synced with cluster spec
	// +kubebuilder:default=0
	UpdatedReplicas int32 `json:"updatedReplicas"`

	// Version is the observed version of etcd cluster
	Version string `json:"version,omitempty"`

	// Endpoint is the etcd client endpoint
	Endpoint string `json:"endpoint,omitempty"`

	// SecretName is the name of the secret containing the etcd client certificate
	SecretName string `json:"secretName,omitempty"`

	// Latest service status of cluster
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []ClusterCondition `json:"conditions,omitempty"`

	// Members is the status of each cluster member.
	// +listType=map
	// +listMapKey=id
	// +optional
	Members []MemberStatus `json:"members,omitempty"`
}

type ClusterPhase string

var (
	ClusterBootstrap = ClusterPhase("Bootstrap")
	ClusterRunning   = ClusterPhase("Running")
	ClusterFailed    = ClusterPhase("Failed")
)

type MemberRole string

var (
	MemberRoleUnspecified = MemberRole("")
	MemberRoleLearner     = MemberRole("Learner")
	MemberRoleMember      = MemberRole("Member")
	MemberRoleLeader      = MemberRole("Leader")
)

var MemberRoleOrder = []MemberRole{MemberRoleLeader, MemberRoleMember, MemberRoleLearner, MemberRoleUnspecified}

type BackupStatus struct {
	LastSuccessfulTime *metav1.Time `json:"lastSuccessfulTime,omitempty"`
	LastScheduleTime   *metav1.Time `json:"lastScheduleTime,omitempty"`
}

type ClusterCondition struct {
	Type               ClusterConditionType   `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime,omitempty"`
	Reason             string                 `json:"reason,omitempty"`
	Message            string                 `json:"message,omitempty"`
}

type ClusterConditionType string

const (
	ClusterAvailable ClusterConditionType = "Available"
	ClusterScaling   ClusterConditionType = "Scaling"
	ClusterUpgrading ClusterConditionType = "Upgrading"
	ClusterBackup    ClusterConditionType = "Backup"
	ClusterRestore   ClusterConditionType = "Restore"
)

// MemberStatus defines the observed state of EtcdCluster member
type MemberStatus struct {
	ID string `json:"id"`

	Name string `json:"name,omitempty"`

	Endpoint string `json:"endpoint,omitempty"`

	Available bool `json:"available"`

	LastSuccessfulTime *metav1.Time `json:"lastSuccessfulTime,omitempty"`

	Version string `json:"version,omitempty"`

	Role MemberRole `json:"role,omitempty"`

	Size *resource.Quantity `json:"size,omitempty"`

	Errors []string `json:"errors,omitempty"`
}

// GetType implements conditions.Condition
func (c ClusterCondition) GetType() ClusterConditionType {
	return c.Type
}

// GetStatus implements conditions.Condition
func (c ClusterCondition) GetStatus() corev1.ConditionStatus {
	return c.Status
}

// Match implements conditions.Upsertable
func (c ClusterCondition) Match(o ClusterCondition) bool {
	return c.Type == o.Type &&
		c.Status == o.Status &&
		c.Reason == o.Reason &&
		c.Message == o.Message
}

// Touch implements conditions.Upsertable
func (c ClusterCondition) Touch() ClusterCondition {
	if c.LastTransitionTime.IsZero() {
		c.LastTransitionTime = metav1.Now()
	}

	return c
}

func init() {
	SchemeBuilder.Register(&EtcdCluster{}, &EtcdClusterList{})
}
