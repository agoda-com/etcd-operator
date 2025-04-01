# controller-gen args
CONTROLLER_GEN_VERSION = v0.17.1
CONTROLLER_GEN_ARGS := \
	paths={./api/...,./pkg/...} \
	crd \
	object:headerFile=hack/boilerplate.go.txt \
	rbac:roleName=etcd-operator \
	output:crd:artifacts:config=config/crd

CRDOC_ARGS := \
	--resources=config/crd \
	--output=docs/api.md

# deployment namespace
SKAFFOLD_NAMESPACE ?= fleet

# explicit package path for coverage
GOCOVERPKG := github.com/agoda-com/etcd-operator/pkg/...
GOTESTARGS := -test.timeout=30m
GOMUTESTARGS := ./pkg
GOLANGCILINT_VERSION := v1.64.8

include makefiles/go.mk
include makefiles/controller.mk
include makefiles/d2.mk

.PHONY: generate fetch-coverage

generate: config/rbac/role.yaml config/e2e/test-role.yaml config/e2e/role.yaml config/e2e/role-binding.yaml

fetch-coverage: $(GOCOVERDIR)
	GOCOVERDIR=$(GOCOVERDIR) \
	SKAFFOLD_NAMESPACE=$(SKAFFOLD_NAMESPACE) \
	SKAFFOLD_RUN_ID=$(SKAFFOLD_RUN_ID) \
	makefiles/scripts/skaffold/fetch-coverage app=etcd-operator

.PHONY: config/rbac/role.yaml config/e2e/test-role.yaml

config/rbac/role.yaml:
	$(CONTROLLER_GEN) > config/rbac/role.yaml \
		paths=./pkg/... \
		rbac:roleName=etcd-operator \
		output:rbac:stdout

config/e2e/test-role.yaml:
	$(CONTROLLER_GEN) > config/e2e/test-role.yaml \
		paths=./e2e/... \
		rbac:roleName=etcd-test \
		output:rbac:stdout

config/e2e/role.yaml: config/rbac/role.yaml
	mkdir -p config/e2e
	yq -r '.kind = "Role" | .rules = .rules' config/rbac/role.yaml >config/e2e/role.yaml

config/e2e/role-binding.yaml: config/rbac/role-binding.yaml
	mkdir -p config/e2e
	yq -r '.kind = "RoleBinding"' config/rbac/role-binding.yaml >config/e2e/role-binding.yaml
