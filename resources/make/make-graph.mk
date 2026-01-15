# make-graph.mk

ifndef MAKE_GRAPH_MK_INCLUDED
MAKE_GRAPH_MK_INCLUDED := 1

GRAPH_DIR ?= .make-graphs

.PHONY: graph
graph:
	@mkdir -p $(GRAPH_DIR)
	@echo "digraph make {" > $(GRAPH_DIR)/targets.dot
	@echo "  rankdir=LR;" >> $(GRAPH_DIR)/targets.dot
	@$(MAKE) -pn | \
	  awk '/^[a-zA-Z0-9_.-]+:/{ \
	    split($$1,a,":"); tgt=a[1]; \
	    for(i=2;i<=NF;i++) \
	      if($$i !~ /^\\./) \
	        printf "  \"%s\" -> \"%s\";\n", tgt, $$i; \
	  }' >> $(GRAPH_DIR)/targets.dot
	@echo "}" >> $(GRAPH_DIR)/targets.dot
	@echo "Graph written to $(GRAPH_DIR)/targets.dot"

endif # MAKE_GRAPH_MK_INCLUDED