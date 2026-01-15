# make-debug.mk
# Debug and diagnostic helpers for GNU Make

ifndef MAKE_DEBUG_MK_INCLUDED
MAKE_DEBUG_MK_INCLUDED := 1

# Print during expansion (not recipe execution)
# Usage: $(call dbg,Message)

DBG ?= 0
define dbg
$(if $(call is_true,$(DBG)),$(info [DBG] $(1)))
endef

# Assert variable is non-empty at expansion time
# Usage: $(call assert_nonempty,VAR_NAME)
define assert_nonempty
$(if $($(1)),,$(error [ASSERT] Variable '$(1)' is empty))
endef

# Assert macro argument is non-empty
# Usage: $(call assert_arg,ARG_NAME,VALUE)
define assert_arg
$(if $(strip $(2)),,$(error [ASSERT] Argument '$(1)' is empty))
endef

# Show list contents with indexes (critical for foreach debugging)
# Usage: $(call debug_list,LIST_NAME)
define debug_list
$(warning [DBG] List $(1) contains:)
$(foreach i,$($(1)),$(warning [DBG]  - '$(i)'))
endef

endif # MAKE_DEBUG_MK_INCLUDED
