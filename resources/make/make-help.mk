# help.mk
# Reusable help system with dynamic categories for Makefiles
# Provides consistent help messages for Makefiles
# Usage: include $(PROJECT_HOME)/resources/make/make-help.mk

ifndef MAKE_HELP_MK_INCLUDED
MAKE_HELP_MK_INCLUDED := 1

HELP_TARGETS    ?=
HELP_CATEGORIES ?=

# help_entry(target, category, description)
define help_entry
# register target (deduplicated)
$(if $(filter $(1),$(HELP_TARGETS)),, \
  $(eval HELP_TARGETS += $(1)))

# register category (deduplicated)
$(if $(filter $(2),$(HELP_CATEGORIES)),, \
  $(eval HELP_CATEGORIES += $(2)))

# store metadata (normalize target name)
$(eval HELP_DESC_$(subst -,_,$(1)) := $(3))
$(eval HELP_CAT_$(subst -,_,$(1))  := $(2))
endef

endif # MAKE_HELP_MK_INCLUDED
