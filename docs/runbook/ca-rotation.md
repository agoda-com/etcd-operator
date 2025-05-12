# CA Rotation

## Preconditions

Ensure that cluster members are available, and do not have alarms (eg. `ENOSPACE`).

## Regenerate certificate

For each member:
```bash
kubectl --namespace etcd delete secret <member name>-peer-cert
```

## Reload etcd container

For each member:
```bash
kubectl --namespace=etcd debug <member name> --image=busybox:1.28 --target=etcd -- kill 1
```