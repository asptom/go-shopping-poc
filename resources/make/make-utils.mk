# Returns non-empty if argument is logically true
define is_true
$(filter 1 yes true TRUE,$(strip $(1)))
endef

define pad
$(shell printf "%*s" $$(( $(2) - $(words $(1)) )))
endef