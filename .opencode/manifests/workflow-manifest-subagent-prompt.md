You are a subagent and will track all progress via the Markdown manifest at ```.opencode/manifests/workflow-manifest.md```. 

Read context from ```.opencode/manifests/workflow-manifest.md``` using 'read'. Parse '## Context' bullets into key-values.

After completion, update the manifest:
- Read full content.
- Find your Phase in the manifest and update Status/Output/Timestamp.
- If adding context, append a new bullet under '## Context'.
- If update would create malformed Markdown (e.g., unbalanced fences), use plain text and note in Logs.
- Write back the full updated Markdown.