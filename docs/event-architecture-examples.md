# Event Architecture Examples

## New Typed Event Handler Pattern

```go
// 1. Define your event struct
type CustomerEvent struct {
    ID           string                `json:"id"`
    EventType    EventType            `json:"type"`
    Timestamp    time.Time             `json:"timestamp"`
    EventPayload CustomerEventPayload  `json:"payload"`
}

// 2. Implement event.Event interface (value receivers)
func (e CustomerEvent) Type() string { return string(e.EventType) }
func (e CustomerEvent) Topic() string { return "customer.changes" }
func (e CustomerEvent) Payload() any { return e.EventPayload }
func (e CustomerEvent) ToJSON() ([]byte, error) { return json.Marshal(e) }

// 3. Create EventFactory
type CustomerEventFactory struct{}
func (f CustomerEventFactory) FromJSON(data []byte) (CustomerEvent, error) {
    var event CustomerEvent
    err := json.Unmarshal(data, &event)
    return event, err
}

// 4. Use in EventReader
factory := CustomerEventFactory{}
handler := eventbus.HandlerFunc[CustomerEvent](func(ctx context.Context, evt CustomerEvent) error {
    log.Printf("Processing %s for customer %s", evt.Type(), evt.EventPayload.CustomerID)
    return nil
})

eventbus.SubscribeTyped(eventBus, factory, handler)
```

## Legacy Pattern (Still Supported)

```go
// Old way still works for backward compatibility
handler := &CustomerEventHandler{Callback: myCallback}
bus.Subscribe("customer.created", handler)
```

## Key Benefits

1. **Type Safety**: Compile-time checking of event types
2. **No Registry**: Eliminates global state and init() complexity
3. **No Adapters**: Direct type handling without conversion layers
4. **Performance**: No reflection or runtime type discovery
5. **Testability**: Easy to create mock events and handlers