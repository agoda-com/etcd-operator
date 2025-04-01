# Defrag

## Spec

Spec: [DefragSpec](/docs/api.md#etcdclusterspecdefrag)

Enable defrag:

```yaml
spec:
  defrag:
    enabled: true    
```

Override schedule / threshold:

```yaml
spec:
  defrag:
    enabled: true
    schedule: "0 6 * * *" 
    threshold:
      size: 128M
      ratio: "0.8"
```

## Trigger Defrag Cronjob

Each cluster has `$CLUSTER-defrag` cronjob created which can be trigerred.

Given cluster name `etcd-test`:

```bash
kubectl --namespace etcd create job etcd-test-defrag --from=cronjob/$CLUSTER-defrag
```