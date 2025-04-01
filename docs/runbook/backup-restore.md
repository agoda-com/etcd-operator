# Backup and Restore

## Backup

Spec: [BackupSpec](/docs/api.md#etcdclusterspecbackup)

If `spec.backup.secretName` is not specified backup will be not enabled.

### Status 

Check backup status:
```bash
kubectl --namespace etcd get etcdcluster etcd-test -o yaml
```

```yaml
status:
  backup:
    enabled: true
    lastScheduleTime: "2024-09-20T04:00:00Z"
    lastSuccessfulTime: "2024-09-20T04:00:36Z"
```

### Operations

#### Enable backup 

```yaml
spec:
  backup:
    enabled: true
    secretName: etcd-backup
```

#### Custom schedule

```yaml
spec:
  backup:
    enabled: true
    secretName: etcd-backup
    schedule: "0 */6 * * *"
```

#### Trigger cronjob

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

When prefix/key are not specified prefix `<namespace>/<name>/` is assumed.
When key is not specified operator uses latest backup object.

### Operations

#### Recreate cluster from latest backup

```yaml
spec:
  restore:
    secretName: etcd-backup
```

#### Use specific backup

```yaml
spec:
  restore:
    secretName: etcd-backup
    key: etcd/etcd-test/manual-backup-123
```