# ETCD Operator

## Docs

* [API](/docs/api.md)
* [Backup and Restore](/docs/runbook/backup-restore.md)
* [Defrag](/docs/defrag.md)
* [CA Rotation](/docs/ca-rotation.md)

## Deployment 

## Standard
* [config/base](config/base) - cluster-wide operator deployment, does not include RBAC and CRD
* [config/crd](config/crd) - generated Custom Resource Definitions
* [config/rbac](config/rbac) - cluster-wide RBAC

## Profiles
* [config/sandbox](config/sandbox) - single namespace deployment with namespace-scoped RBAC
* [config/e2e](config/e2e) - sandbox with coverage enabled on etcd-operator

## Running locally

### Bootstrap local environment

Operator requires cert-manager and CRDs to be installed in the cluster.

```sh
kustomize build --enable-helm config/bootstrap | kubectl apply -f -
```

### Run

```
skaffold run 
```

### Debug

```
skaffold debug
```

VSCode launch configuration:
```json
{
    "name": "Skaffold Debug",
    "type": "go",
    "request": "attach",
    "mode": "remote",
    "host": "localhost",
    "port": 56268,
    "substitutePath": [
        {
            "from": "${workspaceFolder}",
            "to": "/workspace",
        },
    ],
```

### Testing

Unit tests only (marked with `t.Short()`):
```sh
make test
```

Unit and integration tests:
```sh
make integration-test
```

End-to-end tests:
```sh
make e2e-test
```

Run end-to-end tests on dev cluster:
```sh
make e2e-test
```

Coverage:
```sh
make test coverage
make integration-test coverage
make e2e-test coverage
```

Output coverage report:
```sh
CODECOV_HTMLFILE=build/coverage.html make integration-test coverage
open build/coverage.html
```
