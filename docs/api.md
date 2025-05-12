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
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 1<br/>
        </td>
        <td>true</td>
      </tr><tr>
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
        <td><b><a href="#etcdclusterspecdefrag">defrag</a></b></td>
        <td>object</td>
        <td>
          DefragSpec defines the configuration for automated cluster defrag<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>pause</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#etcdclusterspecpodtemplate">podTemplate</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>resources</b></td>
        <td>map[string]int or string</td>
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
        <td><b>schedule</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>suspend</b></td>
        <td>boolean</td>
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
        <td><b>ratio</b></td>
        <td>string</td>
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
        <td><b>size</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>suspend</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### EtcdCluster.spec.podTemplate
<sup><sup>[↩ Parent](#etcdclusterspec)</sup></sup>





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
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td>
          Annotations<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>labels</b></td>
        <td>map[string]string</td>
        <td>
          Labels<br/>
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
            <i>Default</i>: 0<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>learnerReplicas</b></td>
        <td>integer</td>
        <td>
          LearnerReplicas<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 0<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>readyReplicas</b></td>
        <td>integer</td>
        <td>
          ReadyReplicas is the number of ready member pods.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 0<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas is the number of non-terminated members.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 0<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>updatedReplicas</b></td>
        <td>integer</td>
        <td>
          UpdatedReplicas is the number of members that are synced with cluster spec<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 0<br/>
        </td>
        <td>true</td>
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
        <td><b>secretName</b></td>
        <td>string</td>
        <td>
          SecretName is the name of the secret containing the etcd client certificate<br/>
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
