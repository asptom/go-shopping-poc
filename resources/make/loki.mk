# Sub-Makefile for LOki/Grafana installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/loki.mk

LOKI_NAMESPACE ?= loki
SERVICES_NAMESPACE ?= shopping
LOKI_CHART ?= grafana/loki

# ------------------------------------------------------------------
# Install Loki using Helm
# ------------------------------------------------------------------
$(eval $(call help_entry,loki-install,Loki,Install Loki using Helm))
.PHONY: loki-install
loki-install:
	$(call run,Install Loki using Helm,$@, \
		set -euo pipefail; \
		helm upgrade --install loki $(LOKI_CHART) \
			--namespace $(LOKI_NAMESPACE) \
			--values $(PROJECT_HOME)/resources/loki/values-loki.yaml \
			--wait --timeout 180s; \
	)

# ------------------------------------------------------------------
# Install Alloy using Helm
# ------------------------------------------------------------------
$(eval $(call help_entry,alloy-install,Loki,Install Alloy using Helm))
.PHONY: alloy-install
alloy-install:
	$(call run,Install Alloy using Helm,$@, \
		set -euo pipefail; \
		helm upgrade --install alloy grafana/alloy \
			--namespace $(SERVICES_NAMESPACE) \
			--values $(PROJECT_HOME)/resources/loki/values-alloy.yaml \
			--set-file alloy.configMap.content=$(PROJECT_HOME)/resources/loki/alloy-logs.river \
			--wait --timeout 180s; \
	)

# ------------------------------------------------------------------
# Install Grafana using Helm
# ------------------------------------------------------------------
$(eval $(call help_entry,grafana-install,Loki,Install Grafana using Helm))
.PHONY: grafana-install
grafana-install:
	$(call run,Install Grafana using Helm,$@, \
		set -euo pipefail; \
		helm upgrade --install grafana grafana/grafana \
			--namespace $(SERVICES_NAMESPACE) \
			--values $(PROJECT_HOME)/resources/loki/values-grafana.yaml \
			--wait --timeout 180s; \
	)

# ------------------------------------------------------------------
# Install Grafana Loki Stack (Loki + Alloy + Grafana)
# ------------------------------------------------------------------
$(eval $(call help_entry,loki-stack-install,Loki,Install Grafana Loki Stack (Loki + Alloy + Grafana)))
.PHONY: loki-stack-install
loki-stack-install: loki-install alloy-install grafana-install

# ------------------------------------------------------------------
# Uninstall Loki
# ------------------------------------------------------------------
$(eval $(call help_entry,loki-uninstall,Loki,Uninstall Loki using Helm))
.PHONY: loki-uninstall
loki-uninstall:
	$(call run,Uninstall Loki using Helm,$@, \
		set -euo pipefail; \
		helm uninstall loki -n $(LOKI_NAMESPACE); \
	)

# ------------------------------------------------------------------
# Uninstall Alloy
# ------------------------------------------------------------------
$(eval $(call help_entry,alloy-uninstall,Loki,Uninstall Alloy using Helm))
.PHONY: alloy-uninstall
alloy-uninstall:
	$(call run,Uninstall Alloy using Helm,$@, \
		set -euo pipefail; \
		helm uninstall alloy -n $(SERVICES_NAMESPACE); \
	)

# ------------------------------------------------------------------
# Uninstall Grafana
# ------------------------------------------------------------------
$(eval $(call help_entry,grafana-uninstall,Loki,Uninstall Grafana using Helm))
.PHONY: grafana-uninstall
grafana-uninstall:
	$(call run,Uninstall Grafana using Helm,$@, \
		set -euo pipefail; \
		helm uninstall grafana -n $(SERVICES_NAMESPACE); \
	)

# ------------------------------------------------------------------
# Uninstall Grafana Loki Stack (Loki + Alloy + Grafana)
# ------------------------------------------------------------------
$(eval $(call help_entry,loki-stack-uninstall,Loki,Uninstall Grafana Loki Stack (Loki + Alloy + Grafana)))
.PHONY: loki-stack-uninstall
loki-stack-uninstall: loki-uninstall alloy-uninstall grafana-uninstall

# ------------------------------------------------------------------
# Get Grafana credentials
# ------------------------------------------------------------------

$(eval $(call help_entry,grafana-credentials,Loki,Retrieve Grafana credentials from secret))
.PHONY: grafana-credentials
grafana-credentials:
	$(call run,Retrieve Grafana credentials from secret,$@, \
		set -euo pipefail; \
		kubectl get secret --namespace $(SERVICES_NAMESPACE) grafana -o jsonpath="{.data.admin-user}" | base64 --decode ; echo ""; \
		echo ""; \
		 kubectl get secret --namespace $(SERVICES_NAMESPACE) grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo ""; \
		echo ""; \
	)

#-------------------------------------------------------------------
# Expose Grafana port using port-forwarding
#-------------------------------------------------------------------
$(eval $(call help_entry,grafana-port-forward,Loki,Expose Grafana port using port-forwarding))
.PHONY: grafana-port-forward
grafana-port-forward:
	$(call run,Expose Grafana port using port-forwarding,$@, \
		set -euo pipefail; \
		export POD_NAME=$(kubectl get pods --namespace $(SERVICES_NAMESPACE) -l "app.kubernetes.io/name=grafana,app.kubernetes.io/instance=grafana" -o jsonpath="{.items[0].metadata.name}"); \
     	kubectl --namespace $(SERVICES_NAMESPACE) port-forward $$POD_NAME 3000;\
	)