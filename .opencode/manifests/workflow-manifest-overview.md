# Workflow Manifest Overview

## Explanation of Structure
-  **#Workflow Manifest:** Top-level header for indentification
- **##Project:** Simple single-line value.  Easy to update.
- **##Phases:** Bullet list with pipe-separated fields. This mimics a table for structure but allows flexible content in "Output" (e.g., embed code blocks if needed).
    - Status options: pending, in_progress, completed, failed.
    - Output: Can be a summary, file path, or even a fenced code block for code/debug info.
    - Timestamp: Use ISO for consistency; agents can generate this.

- **##Context**: Key-value bullet list. Values can span lines or include Markdown elements like code blocks, avoiding JSON escaping issues.
- **##Logs:** Optional free-form section for debugging without polluting phases. This isolates problematic content.
