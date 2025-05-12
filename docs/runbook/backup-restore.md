# Backup and Restore

## Backup

Spec: [BackupSpec](/docs/api.md#etcdclusterspecbackup)

Backup is enabled by default as long as `etcd-operator` deployment is configured with AWS credentials using environment variables.

Required environment variables:
- `AWS_DEFAULT_REGION`
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_BUCKET_NAME`

Status condition `Backup` indicates if backup is enabled.

### Status 

Check backup status:
```bash
kubectl --namespace etcd get etcdcluster etcd-test -o yaml
```

```yaml
status:
  backup:
    lastScheduleTime: "2024-09-20T04:00:00Z"
    lastSuccessfulTime: "2024-09-20T04:00:36Z"
```

### Operations

### Suspend backup

```yaml
spec:
  backup:
    suspend: true
```

### Custom schedule

```yaml
spec:
  backup:
    schedule: "0 */6 * * *"
```

### Trigger cronjob

Given cluster name `etcd-test`:

```bash
kubectl --namespace etcd create job --from=cronjob/etcd-test-backup --output name
```

Then wait for the job to Complete:

```bash
kubectl --namespace etcd wait --for=condition=Complete job/etcd-test-backup-mkbmk --timeout 5m
```

## Restore

Spec: [RestoreSpec](/docs/api.md#etcdclusterspecrestore)

Restore only happens when new cluster is created as its executed in `Bootstrap` phase.

When prefix/key are not specified prefix `<namespace>/<name>/` is assumed.

When key is not specified operator uses latest backup object, if no backup is not found condition `Restore` with status `False` will be set and k8s resources will be not created.

### Recreate cluster from latest backup

```yaml
spec:
  restore: {}
```

### Restore from other cluster latest backup

```yaml
spec:
  restore:
    prefix: etcd/etcd-other
```

### Use specific backup

```yaml
spec:
  restore:
    key: etcd/etcd-test/manual-backup-123
```