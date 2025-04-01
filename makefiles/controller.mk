ifndef _include_envtest_mk
_include_envtest_mk := 1

include makefiles/go.mk

### Variables
CONTROLLER_GEN_VERSION ?= v0.12.0
CONTROLLER_GEN_ARGS ?=

CRDOC_VERSION ?= v0.6.3
CRDOC_ARGS ?=

ENVTEST_KUBE_VERSION ?= 1.27

### Targets

### Tools
CONTROLLER_GEN_ROOT := $(BINDIR)/controller-gen-$(CONTROLLER_GEN_VERSION)
CONTROLLER_GEN := $(CONTROLLER_GEN_ROOT)/controller-gen

$(CONTROLLER_GEN):
	@mkdir -p $(CONTROLLER_GEN_ROOT)
	GOBIN=$(abspath $(CONTROLLER_GEN_ROOT)) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)

CRDOC_ROOT := $(BINDIR)/crdoc-$(CRDOC_VERSION)
CRDOC := $(CRDOC_ROOT)/crdoc

$(CRDOC):
	@mkdir -p $(CRDOC_ROOT)
	GOBIN=$(abspath $(CRDOC_ROOT)) go install fybrik.io/crdoc@$(CRDOC_VERSION)

### Implementation

_envtest := go run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
integration-test-go: export KUBEBUILDER_ASSETS = $(shell $(_envtest) use -p path $(ENVTEST_KUBE_VERSION))

ifneq ($(strip $(CONTROLLER_GEN_ARGS)),)
.PHONY: generate-controller

generate: generate-controller 

generate-controller: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) $(CONTROLLER_GEN_ARGS)
endif # CONTROLLER_GEN_ARGS

ifneq ($(strip $(CRDOC_ARGS)),)
.PHONY: generate-crdoc

generate: generate-crdoc

generate-crdoc: $(CRDOC)
	$(CRDOC) $(CRDOC_ARGS)
endif # CRDOC_ARGS

endif # _include_envtest_mk