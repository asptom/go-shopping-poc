#!/usr/bin/env bash
set -euo pipefail

# Print documented .PHONY targets and their descriptions
# Supports: .PHONY: target1 target2 ## description
# - Handles multi-line .PHONY continuations using backslashes
# - Groups targets by the prefix before the first '-' (uppercased)

awk -F'##' '
/^[[:space:]]*\.PHONY:/ {
  # Accumulate continuation lines ending with backslash
  cur = $0
  while (match(cur, /\\[[:space:]]*$/)) {
    sub(/\\[[:space:]]*$/, "", cur)
    if (getline nextline <= 0) break
    sub(/^[[:space:]]+/, "", nextline)
    cur = cur " " nextline
  }

  line = cur
  # split description and left-hand targets
  n = split(line, parts, /##/)
  left = parts[1]
  desc = (n>1 ? parts[2] : "")
  # strip the leading .PHONY: and whitespace
  sub(/^[[:space:]]*\.PHONY:[[:space:]]*/, "", left)
  gsub(/^[[:space:]]+|[[:space:]]+$/, "", left)
  gsub(/^[[:space:]]+|[[:space:]]+$/, "", desc)

  # split targets and record them grouped
  m = split(left, targets, / +/)
  for (i=1;i<=m;i++) {
    t = targets[i]
    gsub(/\\/, "", t) # remove stray backslashes
    if (t == "") next
    split(t, p, "-")
    g = toupper(p[1])
    if (!(g in seen)) { seen[g]=1; groups[++gcount]=g }
    items[g, ++count[g]] = t "\t" desc
  }
}
END {
  for (gi=1; gi<=gcount; gi++) {
    g = groups[gi]
    print "[" g "]"
    for (i=1; i<=count[g]; i++) {
      split(items[g,i], arr, "\t")
      printf "  %-25s %s\n", arr[1], arr[2]
    }
    print ""
  }
}' Makefile resources/make/*.mk || true
