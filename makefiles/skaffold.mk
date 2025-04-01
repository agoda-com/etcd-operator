ifndef _include_skaffold_mk
_include_skaffold_mk := 1

include makefiles/base.mk

### Variables

# GOCOVERDIR := build/coverage
# KUBECONFIG := ~/.kube/config

# Kube context to use
# SKAFFOLD_KUBE_CONTEXT := kind-fleet
SKAFFOLD_KUBE_CONTEXT ?= $(shell kubectl config current-context)

# Namespace to fetch coverage from, comma separated
# SKAFFOLD_NAMESPACE := sandbox
SKAFFOLD_NAMESPACE ?= sandbox

# Populated by skaffold
# SKAFFOLD_RUN_ID := $(shell uuidgen)

# Deployment/Pod selector to pass to fetch-coverage
# FETCH_COVERAGE_SELECTOR := app=makefiles-example

### Implementation

ifneq ($(and $(filter e2e-test,$(MAKECMDGOALS)),$(strip $(GOCOVERDIR))),)
SKAFFOLD_RUN_ID ?= $(shell kubectl get deployment --namespace $(SKAFFOLD_NAMESPACE) --selector $(FETCH_COVERAGE_SELECTOR) -o json | jq -r '.items[0].metadata.labels["skaffold.dev/run-id"]')

.PHONY: coverage-skaffold

$(GOCOVEROUT): coverage-skaffold

coverage-skaffold: $(GOCOVERDIR)
	GOCOVERDIR=$(GOCOVERDIR) \
	KUBECONFIG=$(KUBECONFIG) \
	SKAFFOLD_KUBE_CONTEXT=$(SKAFFOLD_KUBE_CONTEXT) \
	SKAFFOLD_NAMESPACE=$(SKAFFOLD_NAMESPACE) \
	SKAFFOLD_RUN_ID=$(SKAFFOLD_RUN_ID) \
	makefiles/scripts/skaffold/fetch-coverage $(FETCH_COVERAGE_ARGS)
	
endif # SKAFFOLD_CONFIG

endif # _include_skaffold_mk