# Defrag

## Spec

Spec: [DefragSpec](/docs/api.md#etcdclusterspecdefrag)

### Suspend defrag

```yaml
spec:
  defrag:
    suspend: true    
```

## Override schedule

```yaml
spec:
  defrag:
    schedule: "0 6 * * *" 
```

## Override threshold

```yaml
spec:
  defrag:
    size: 256M # default 128M
    ratio: "0.8" # default 0.7
```

## Trigger Defrag Cronjob

Each cluster has `$CLUSTER-defrag` cronjob created which can be trigerred.

Given cluster name `etcd-test`:

```bash
kubectl --namespace etcd create job etcd-test-defrag --from=cronjob/$CLUSTER-defrag
```