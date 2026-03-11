# Logging Standard

This standard defines concise, consistent logging patterns for both platform and domain services.

## Goals

- Make logs easy to scan under pressure.
- Keep messages short and structured.
- Support fast debugging of requests, events, and async workflows.
- Avoid leaking sensitive data.

## Core Rules

1. Use structured logs (`slog`) only.
2. Message text should be concise and action-oriented.
3. Put variability in fields, not in message text.
4. Reuse consistent field keys across all services.
5. Never log secrets, card numbers, CVVs, auth tokens, or full PII payloads.

## Message Style

- Prefer: `verb + object + context`
  - Good: `"Create cart failed"`
  - Good: `"Checkout started"`
  - Bad: `"Something went wrong in checkout method with id"`
- Keep message text stable to support search and alerting.

## Standard Fields

Use these keys consistently where relevant:

- `component`: package-level component name (for example `cart_handler`, `order_service`).
- `operation`: short operation name (`create_cart`, `checkout`, `publish_event`).
- `request_id`: request correlation id if available.
- `event_id`, `event_type`: for event handlers/publishers.
- `cart_id`, `customer_id`, `order_id`, `product_id`: domain identifiers.
- `status`: domain status transitions.
- `duration_ms`: operation duration for expensive calls.
- `error`: canonical error string at failure points.

## Log Levels

- `Debug`: high-volume details for local debugging.
- `Info`: important lifecycle and business milestones.
- `Warn`: recoverable failures, retries, degraded behavior.
- `Error`: operation failed and user/business flow impacted.

## Required Coverage Points

Each service should have logs for:

1. Process lifecycle:
   - service start, stop, health degradation.
2. Boundary operations:
   - HTTP request handling, database boundaries, event publish/consume.
3. Domain decision failures:
   - validation failures, invalid transitions, missing dependencies.
4. Retries and async processing:
   - retry attempt count, backoff, terminal failure.

## Sensitive Data Policy

- Never log raw card data (`card_number`, `card_cvv`).
- Never log credentials, tokens, secrets, DSNs with secrets.
- For customer/contact data, log identifiers only when possible.
- If field inclusion is required for support, mask by default.

## Anti-Patterns

- Logging entire request bodies by default.
- String-concatenated logs instead of structured fields.
- Duplicate logs for same failure in a single call stack.
- Inconsistent field naming (`cartId` in one package, `cart_id` in another).

## Review Checklist

- Message text is concise and stable.
- Field keys follow standard names.
- Level matches severity.
- No sensitive values are emitted.
- Logs include enough context to trace failures across services.
