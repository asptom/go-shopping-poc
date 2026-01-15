# output.mk
# Reusable output helpers for Makefiles
# Provides consistent formatting for command output
# Usage: include $(PROJECT_HOME)/resources/make/output.mk

ifndef MAKE_OUTPUT_MK_INCLUDED
MAKE_OUTPUT_MK_INCLUDED := 1

LOG_DIR ?= .logs

define strip #Clear surrounding whitespace
$(strip $(1))
endef

define run
	@mkdir -p $(LOG_DIR)
	@echo ""
	@echo "▶ $(1)"
	@echo "----------------------------------------"
	@set -e; \
	$(3) >$(LOG_DIR)/$(call strip,$(2)).log 2>&1 || \
	( echo "✖ Failed: $(1)"; \
	  echo "  Log: $(LOG_DIR)/$(call strip,$(2)).log"; \
	  exit 1 )
	@echo "✔ Done: $(1)"
	@echo ""
endef

endif # MAKE_OUTPUT_MK_INCLUDED