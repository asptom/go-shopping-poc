# Sub-Makefile for Grafana - loki/alloy/grafana installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/grafana.mk

GRAFANA_NAMESPACE ?= grafana
SERVICES_NAMESPACE ?= shopping
LOKI_CHART ?= grafana/loki
ALLOY_CHART ?= grafana/alloy
GRAFANA_CHART ?= grafana/grafana

# ------------------------------------------------------------------
# Install Loki using Helm
# ------------------------------------------------------------------
$(eval $(call help_entry,loki-install,Loki,Install Loki using Helm))
.PHONY: loki-install
loki-install:
	$(call run,Install Loki using Helm,$@, \
		set -euo pipefail; \
		helm upgrade --install loki $(LOKI_CHART) \
			--namespace $(GRAFANA_NAMESPACE) \
			--values $(PROJECT_HOME)/resources/grafana/values-loki.yaml \
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
			--namespace $(GRAFANA_NAMESPACE) \
			--values $(PROJECT_HOME)/resources/grafana/values-alloy.yaml \
			--set-file alloy.configMap.content=$(PROJECT_HOME)/resources/grafana/alloy-logs.river \
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
			--namespace $(GRAFANA_NAMESPACE) \
			--values $(PROJECT_HOME)/resources/grafana/values-grafana.yaml \
			--wait --timeout 180s; \
	)

# ------------------------------------------------------------------
# Install Grafana Ingress using kubectl
# ------------------------------------------------------------------
$(eval $(call help_entry,grafana-ingress-install,Loki,Install Grafana Ingress using kubectl))
.PHONY: grafana-ingress-install
grafana-ingress-install:
	$(call run,Install Grafana Ingress using kubectl,$@, \
		set -euo pipefail; \
		kubectl apply -f deploy/k8s/platform/grafana/grafana-ingress.yaml; \
	)	

# ------------------------------------------------------------------
# Install Grafana Loki Stack (Loki + Alloy + Grafana)
# ------------------------------------------------------------------
$(eval $(call help_entry,grafana-stack-install,Loki,Install Grafana Loki Stack (Loki + Alloy + Grafana)))
.PHONY: grafana-stack-install
grafana-stack-install: loki-install alloy-install grafana-install grafana-ingress-install

# ------------------------------------------------------------------
# Uninstall Loki
# ------------------------------------------------------------------
$(eval $(call help_entry,loki-uninstall,Loki,Uninstall Loki using Helm))
.PHONY: loki-uninstall
loki-uninstall:
	$(call run,Uninstall Loki using Helm,$@, \
		set -euo pipefail; \
		helm uninstall loki -n $(GRAFANA_NAMESPACE); \
	)

# ------------------------------------------------------------------
# Uninstall Alloy
# ------------------------------------------------------------------
$(eval $(call help_entry,alloy-uninstall,Loki,Uninstall Alloy using Helm))
.PHONY: alloy-uninstall
alloy-uninstall:
	$(call run,Uninstall Alloy using Helm,$@, \
		set -euo pipefail; \
		helm uninstall alloy -n $(GRAFANA_NAMESPACE); \
	)

# ------------------------------------------------------------------
# Uninstall Grafana
# ------------------------------------------------------------------
$(eval $(call help_entry,grafana-uninstall,Loki,Uninstall Grafana using Helm))
.PHONY: grafana-uninstall
grafana-uninstall:
	$(call run,Uninstall Grafana using Helm,$@, \
		set -euo pipefail; \
		helm uninstall grafana -n $(GRAFANA_NAMESPACE); \
	)

# ------------------------------------------------------------------
# Uninstall Grafana Ingress
# ------------------------------------------------------------------

$(eval $(call help_entry,grafana-ingress-uninstall,Loki,Uninstall Grafana Ingress using kubectl))
.PHONY: grafana-ingress-uninstall
grafana-ingress-uninstall:
	$(call run,Uninstall Grafana Ingress using kubectl,$@, \
		set -euo pipefail; \
		kubectl delete -f deploy/k8s/platform/grafana/grafana-ingress.yaml --ignore-not-found; \
	)

# ------------------------------------------------------------------
# Uninstall Grafana Loki Stack (Loki + Alloy + Grafana)
# ------------------------------------------------------------------
$(eval $(call help_entry,grafana-stack-uninstall,Loki,Uninstall Grafana Loki Stack (Loki + Alloy + Grafana)))
.PHONY: grafana-stack-uninstall
grafana-stack-uninstall: loki-uninstall alloy-uninstall grafana-uninstall grafana-ingress-uninstall

# ------------------------------------------------------------------
# Get Grafana credentials
# ------------------------------------------------------------------

$(eval $(call help_entry,grafana-credentials,Loki,Retrieve Grafana credentials from secret))
.PHONY: grafana-credentials
grafana-credentials:
	$(call run,Retrieve Grafana credentials from secret,$@, \
		set -euo pipefail; \
		kubectl get secret --namespace $(GRAFANA_NAMESPACE) grafana -o jsonpath="{.data.admin-user}" | base64 --decode ; echo ""; \
		echo ""; \
		 kubectl get secret --namespace $(GRAFANA_NAMESPACE) grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo ""; \
		echo ""; \
	)