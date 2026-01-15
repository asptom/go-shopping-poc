# make-doctor.mk
# Self-diagnostics for Makefiles

ifndef MAKE_DOCTOR_MK_INCLUDED
MAKE_DOCTOR_MK_INCLUDED := 1

.PHONY: doctor
doctor:
	@echo ""
	@echo "🩺 Make Doctor"
	@echo "----------------------------------------"
	@echo ""
	@echo "SERVICES:"
	@$(foreach s,$(SERVICES),printf "  - '%s'\n" "$(s)";)
	@echo ""
	@echo "SERVICE_HAS_DB flags:"
	@$(foreach s,$(SERVICES), \
		printf "  %-20s = '%s'\n" "$(s)" "$(SERVICE_HAS_DB_$(s))"; \
	)
	@echo ""
	@echo "HELP_TARGETS:"
	@$(foreach t,$(HELP_TARGETS),printf "  - '%s'\n" "$(t)";)
	@echo ""
	@echo "HELP_CATEGORIES:"
	@$(foreach c,$(HELP_CATEGORIES),printf "  - '%s'\n" "$(c)";)
	@echo ""
	@echo "LOG_DIR: $(LOG_DIR)"
	@echo ""

endif # MAKE_DOCTOR_MK_INCLUDED
