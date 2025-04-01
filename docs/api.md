# API Reference

Packages:

- [etcd.fleet.agoda.com/v1](#etcdfleetagodacomv1)

# etcd.fleet.agoda.com/v1

Resource Types:

- [EtcdCluster](#etcdcluster)




## EtcdCluster
<sup><sup>[↩ Parent](#etcdfleetagodacomv1 )</sup></sup>






EtcdCluster is the Schema for the etcdclusters API

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>etcd.fleet.agoda.com/v1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>EtcdCluster</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#etcdclusterspec">spec</a></b></td>
        <td>object</td>
        <td>
          EtcdClusterSpec defines the desired state of EtcdCluster<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#etcdclusterstatus">status</a></b></td>
        <td>object</td>
        <td>
          EtcdClusterStatus defines the observed state of EtcdCluster<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### EtcdCluster.spec
<sup><sup>[↩ Parent](#etcdcluster)</sup></sup>



EtcdClusterSpec defines the desired state of EtcdCluster

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>version</b></td>
        <td>string</td>
        <td>
          Version<br/>
          <br/>
            <i>Default</i>: v3.5.7<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#etcdclusterspecbackup">backup</a></b></td>
        <td>object</td>
        <td>
          BackupSpec defines the configuration to backup cluster to<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>commonAnnotations</b></td>
        <td>map[string]string</td>
        <td>
          CommonAnnotations<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>commonLabels</b></td>
        <td>map[string]string</td>
        <td>
          CommonLabels<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#etcdclusterspecdefrag">defrag</a></b></td>
        <td>object</td>
        <td>
          DefragSpec defines the configuration for automated cluster defrag<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>
          PriorityClassName is the pod's priority<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#etcdclusterspecresources">resources</a></b></td>
        <td>object</td>
        <td>
          Compute Resources required by each member of cluster.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#etcdclusterspecrestore">restore</a></b></td>
        <td>object</td>
        <td>
          RestoreSpec defines the configuration to restore cluster from<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>runtimeClassName</b></td>
        <td>string</td>
        <td>
          RuntimeClassName is the pod's runtime class<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storageMedium</b></td>
        <td>string</td>
        <td>
          StorageMedium=Memory creates emptyDir volume on tmpfs<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storageQuota</b></td>
        <td>int or string</td>
        <td>
          StorageQuota sets a size limit on storage<br/>
          <br/>
            <i>Default</i>: 4G<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### EtcdCluster.spec.backup
<sup><sup>[↩ Parent](#etcdclusterspec)</sup></sup>



BackupSpec defines the configuration to backup cluster to

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>schedule</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>secretName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### EtcdCluster.spec.defrag
<sup><sup>[↩ Parent](#etcdclusterspec)</sup></sup>



DefragSpec defines the configuration for automated cluster defrag

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>schedule</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#etcdclusterspecdefragthreshold">threshold</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### EtcdCluster.spec.defrag.threshold
<sup><sup>[↩ Parent](#etcdclusterspecdefrag)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>ratio</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>size</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### EtcdCluster.spec.resources
<sup><sup>[↩ Parent](#etcdclusterspec)</sup></sup>



Compute Resources required by each member of cluster.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#etcdclusterspecresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims,
that are used by this container.

This is an alpha field and requires enabling the
DynamicResourceAllocation feature gate.

This field is immutable. It can only be set for containers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.
If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,
otherwise to an implementation-defined value. Requests cannot exceed Limits.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### EtcdCluster.spec.resources.claims[index]
<sup><sup>[↩ Parent](#etcdclusterspecresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of
the Pod where this field is used. It makes that resource available
inside a container.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>request</b></td>
        <td>string</td>
        <td>
          Request is the name chosen for a request in the referenced claim.
If empty, everything from the claim is made available, otherwise
only the result of this request.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### EtcdCluster.spec.restore
<sup><sup>[↩ Parent](#etcdclusterspec)</sup></sup>



RestoreSpec defines the configuration to restore cluster from

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>prefix</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>secretName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### EtcdCluster.status
<sup><sup>[↩ Parent](#etcdcluster)</sup></sup>



EtcdClusterStatus defines the observed state of EtcdCluster

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>availableReplicas</b></td>
        <td>integer</td>
        <td>
          AvailableReplicas is the number of fully provisioned members.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#etcdclusterstatusbackup">backup</a></b></td>
        <td>object</td>
        <td>
          Backup<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#etcdclusterstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          Latest service status of cluster<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>endpoint</b></td>
        <td>string</td>
        <td>
          Endpoint is the etcd client endpoint<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>learnerReplicas</b></td>
        <td>integer</td>
        <td>
          LearnerReplicas<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#etcdclusterstatusmembersindex">members</a></b></td>
        <td>[]object</td>
        <td>
          Members is the status of each cluster member.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          ObservedGeneration<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>phase</b></td>
        <td>string</td>
        <td>
          Lifecycle phase<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readyReplicas</b></td>
        <td>integer</td>
        <td>
          ReadyReplicas is the number of ready member pods.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas is the number of non-terminated members.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>secretName</b></td>
        <td>string</td>
        <td>
          SecretName is the name of the secret containing the etcd client certificate<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>updatedReplicas</b></td>
        <td>integer</td>
        <td>
          UpdatedReplicas is the number of members that are synced with cluster spec<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>version</b></td>
        <td>string</td>
        <td>
          Version is the observed version of etcd cluster<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### EtcdCluster.status.backup
<sup><sup>[↩ Parent](#etcdclusterstatus)</sup></sup>



Backup

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>lastScheduleTime</b></td>
        <td>string</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>lastSuccessfulTime</b></td>
        <td>string</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### EtcdCluster.status.conditions[index]
<sup><sup>[↩ Parent](#etcdclusterstatus)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### EtcdCluster.status.members[index]
<sup><sup>[↩ Parent](#etcdclusterstatus)</sup></sup>



MemberStatus defines the observed state of EtcdCluster member

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>available</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>endpoint</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>errors</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>lastSuccessfulTime</b></td>
        <td>string</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>role</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>size</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>version</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>
