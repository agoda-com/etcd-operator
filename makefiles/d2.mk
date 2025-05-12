ifndef _include_d2_mk
_include_d2_mk := 1

include makefiles/base.mk

### Variables

D2_VERSION ?= v0.6.3
D2_DIR ?= docs/
D2_FLAGS ?= --dark-theme 200 --sketch

### Targets

.PHONY: generate-d2

generate: generate-d2

### Tools

D2_ROOT := $(BINDIR)/d2-$(D2_VERSION)
D2 := $(D2_ROOT)/d2

$(D2):
	GOBIN=$(abspath $(D2_ROOT)) go install oss.terrastruct.com/d2@$(D2_VERSION)

### Implementation

_d2_input := $(wildcard $(addsuffix *.d2,$(D2_DIR)))
_d2_output := $(patsubst %.d2,%.svg,$(_d2_input))

generate-d2: $(D2) $(_d2_output)

%.svg: %.d2
	$(D2) $(D2_FLAGS) $< $@

endif # _include_d2_mk