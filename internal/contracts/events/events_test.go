package events

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestCustomerEvent_JSONMarshaling tests JSON marshaling and unmarshaling of CustomerEvent
func TestCustomerEvent_JSONMarshaling(t *testing.T) {
	// Create a test event
	details := map[string]string{
		"name":  "John Doe",
		"email": "john@example.com",
	}
	event := NewCustomerCreatedEvent("customer-123", details)

	// Marshal to JSON
	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to marshal CustomerEvent to JSON: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, `"type":"customer.created"`) {
		t.Error("JSON should contain event type")
	}
	if !strings.Contains(jsonStr, `"customer_id":"customer-123"`) {
		t.Error("JSON should contain customer ID")
	}
	if !strings.Contains(jsonStr, `"name":"John Doe"`) {
		t.Error("JSON should contain details")
	}

	// Unmarshal back to struct
	var unmarshaled CustomerEvent
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON to CustomerEvent: %v", err)
	}

	// Verify fields
	if unmarshaled.EventType != CustomerCreated {
		t.Errorf("Expected event type %s, got %s", CustomerCreated, unmarshaled.EventType)
	}
	if unmarshaled.EventPayload.CustomerID != "customer-123" {
		t.Errorf("Expected customer ID 'customer-123', got '%s'", unmarshaled.EventPayload.CustomerID)
	}
	if unmarshaled.EventPayload.Details["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", unmarshaled.EventPayload.Details["name"])
	}
}

// TestCustomerEvent_Factory tests the CustomerEventFactory
func TestCustomerEvent_Factory(t *testing.T) {
	// Create original event
	details := map[string]string{"key": "value"}
	originalEvent := NewCustomerUpdatedEvent("customer-456", details)

	// Serialize to JSON
	jsonData, err := originalEvent.ToJSON()
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Use factory to reconstruct
	factory := CustomerEventFactory{}
	reconstructedEvent, err := factory.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to reconstruct event from JSON: %v", err)
	}

	// Verify reconstruction
	if reconstructedEvent.Type() != originalEvent.Type() {
		t.Errorf("Expected type %s, got %s", originalEvent.Type(), reconstructedEvent.Type())
	}
	if reconstructedEvent.GetEntityID() != originalEvent.GetEntityID() {
		t.Errorf("Expected entity ID %s, got %s", originalEvent.GetEntityID(), reconstructedEvent.GetEntityID())
	}
	if reconstructedEvent.EventPayload.Details["key"] != "value" {
		t.Errorf("Expected details key 'value', got '%s'", reconstructedEvent.EventPayload.Details["key"])
	}
}

// TestCustomerEvent_InterfaceCompliance tests that CustomerEvent implements the Event interface
func TestCustomerEvent_InterfaceCompliance(t *testing.T) {
	event := NewCustomerCreatedEvent("test-customer", nil)

	// Test Type() method
	if event.Type() != "customer.created" {
		t.Errorf("Expected type 'customer.created', got '%s'", event.Type())
	}

	// Test Topic() method
	if event.Topic() != "CustomerEvents" {
		t.Errorf("Expected topic 'CustomerEvents', got '%s'", event.Topic())
	}

	// Test Payload() method
	payload := event.Payload()
	customerPayload, ok := payload.(CustomerEventPayload)
	if !ok {
		t.Error("Payload should be CustomerEventPayload")
	}
	if customerPayload.CustomerID != "test-customer" {
		t.Errorf("Expected customer ID 'test-customer', got '%s'", customerPayload.CustomerID)
	}

	// Test GetEntityID() method
	if event.GetEntityID() != "test-customer" {
		t.Errorf("Expected entity ID 'test-customer', got '%s'", event.GetEntityID())
	}

	// Test GetResourceID() method
	if event.GetResourceID() != "test-customer" {
		t.Errorf("Expected resource ID 'test-customer', got '%s'", event.GetResourceID())
	}
}

// TestCustomerEvent_ConvenienceConstructors tests all convenience constructors
func TestCustomerEvent_ConvenienceConstructors(t *testing.T) {
	details := map[string]string{"test": "data"}

	testCases := []struct {
		name         string
		constructor  func() *CustomerEvent
		expectedType EventType
		entityID     string
		resourceID   string
	}{
		{"NewCustomerCreatedEvent", func() *CustomerEvent { return NewCustomerCreatedEvent("cust-1", details) }, CustomerCreated, "cust-1", "cust-1"},
		{"NewCustomerUpdatedEvent", func() *CustomerEvent { return NewCustomerUpdatedEvent("cust-2", details) }, CustomerUpdated, "cust-2", "cust-2"},
		{"NewAddressAddedEvent", func() *CustomerEvent { return NewAddressAddedEvent("cust-3", "addr-1", details) }, AddressAdded, "cust-3", "addr-1"},
		{"NewAddressUpdatedEvent", func() *CustomerEvent { return NewAddressUpdatedEvent("cust-4", "addr-2", details) }, AddressUpdated, "cust-4", "addr-2"},
		{"NewAddressDeletedEvent", func() *CustomerEvent { return NewAddressDeletedEvent("cust-5", "addr-3", details) }, AddressDeleted, "cust-5", "addr-3"},
		{"NewCardAddedEvent", func() *CustomerEvent { return NewCardAddedEvent("cust-6", "card-1", details) }, CardAdded, "cust-6", "card-1"},
		{"NewCardUpdatedEvent", func() *CustomerEvent { return NewCardUpdatedEvent("cust-7", "card-2", details) }, CardUpdated, "cust-7", "card-2"},
		{"NewCardDeletedEvent", func() *CustomerEvent { return NewCardDeletedEvent("cust-8", "card-3", details) }, CardDeleted, "cust-8", "card-3"},
		{"NewDefaultShippingAddressChangedEvent", func() *CustomerEvent { return NewDefaultShippingAddressChangedEvent("cust-9", "addr-4", details) }, DefaultShippingAddressChanged, "cust-9", "addr-4"},
		{"NewDefaultBillingAddressChangedEvent", func() *CustomerEvent { return NewDefaultBillingAddressChangedEvent("cust-10", "addr-5", details) }, DefaultBillingAddressChanged, "cust-10", "addr-5"},
		{"NewDefaultCreditCardChangedEvent", func() *CustomerEvent { return NewDefaultCreditCardChangedEvent("cust-11", "card-4", details) }, DefaultCreditCardChanged, "cust-11", "card-4"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := tc.constructor()

			if event.EventType != tc.expectedType {
				t.Errorf("Expected event type %s, got %s", tc.expectedType, event.EventType)
			}
			if event.GetEntityID() != tc.entityID {
				t.Errorf("Expected entity ID %s, got %s", tc.entityID, event.GetEntityID())
			}
			if event.GetResourceID() != tc.resourceID {
				t.Errorf("Expected resource ID %s, got %s", tc.resourceID, event.GetResourceID())
			}
			if event.EventPayload.Details["test"] != "data" {
				t.Errorf("Expected details to contain test data")
			}
		})
	}
}

// TestProductEvent_JSONMarshaling tests JSON marshaling and unmarshaling of ProductEvent
func TestProductEvent_JSONMarshaling(t *testing.T) {
	// Create a test event
	details := map[string]string{
		"name":  "Test Product",
		"price": "29.99",
	}
	event := NewProductCreatedEvent("product-123", details)

	// Marshal to JSON
	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to marshal ProductEvent to JSON: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, `"type":"product.created"`) {
		t.Error("JSON should contain event type")
	}
	if !strings.Contains(jsonStr, `"product_id":"product-123"`) {
		t.Error("JSON should contain product ID")
	}
	if !strings.Contains(jsonStr, `"name":"Test Product"`) {
		t.Error("JSON should contain details")
	}

	// Unmarshal back to struct
	var unmarshaled ProductEvent
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON to ProductEvent: %v", err)
	}

	// Verify fields
	if unmarshaled.EventType != ProductCreated {
		t.Errorf("Expected event type %s, got %s", ProductCreated, unmarshaled.EventType)
	}
	if unmarshaled.EventPayload.ProductID != "product-123" {
		t.Errorf("Expected product ID 'product-123', got '%s'", unmarshaled.EventPayload.ProductID)
	}
	if unmarshaled.EventPayload.Details["name"] != "Test Product" {
		t.Errorf("Expected name 'Test Product', got '%s'", unmarshaled.EventPayload.Details["name"])
	}
}

// TestProductEvent_Factory tests the ProductEventFactory
func TestProductEvent_Factory(t *testing.T) {
	// Create original event
	details := map[string]string{"category": "electronics"}
	originalEvent := NewProductUpdatedEvent("product-789", details)

	// Serialize to JSON
	jsonData, err := originalEvent.ToJSON()
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Use factory to reconstruct
	factory := ProductEventFactory{}
	reconstructedEvent, err := factory.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to reconstruct event from JSON: %v", err)
	}

	// Verify reconstruction
	if reconstructedEvent.Type() != originalEvent.Type() {
		t.Errorf("Expected type %s, got %s", originalEvent.Type(), reconstructedEvent.Type())
	}
	if reconstructedEvent.GetEntityID() != originalEvent.GetEntityID() {
		t.Errorf("Expected entity ID %s, got %s", originalEvent.GetEntityID(), reconstructedEvent.GetEntityID())
	}
	if reconstructedEvent.EventPayload.Details["category"] != "electronics" {
		t.Errorf("Expected details category 'electronics', got '%s'", reconstructedEvent.EventPayload.Details["category"])
	}
}

// TestProductEvent_InterfaceCompliance tests that ProductEvent implements the Event interface
func TestProductEvent_InterfaceCompliance(t *testing.T) {
	event := NewProductCreatedEvent("test-product", nil)

	// Test Type() method
	if event.Type() != "product.created" {
		t.Errorf("Expected type 'product.created', got '%s'", event.Type())
	}

	// Test Topic() method
	if event.Topic() != "ProductEvents" {
		t.Errorf("Expected topic 'ProductEvents', got '%s'", event.Topic())
	}

	// Test Payload() method
	payload := event.Payload()
	productPayload, ok := payload.(ProductEventPayload)
	if !ok {
		t.Error("Payload should be ProductEventPayload")
	}
	if productPayload.ProductID != "test-product" {
		t.Errorf("Expected product ID 'test-product', got '%s'", productPayload.ProductID)
	}

	// Test GetEntityID() method
	if event.GetEntityID() != "test-product" {
		t.Errorf("Expected entity ID 'test-product', got '%s'", event.GetEntityID())
	}

	// Test GetResourceID() method
	if event.GetResourceID() != "test-product" {
		t.Errorf("Expected resource ID 'test-product', got '%s'", event.GetResourceID())
	}
}

// TestProductEvent_ConvenienceConstructors tests all convenience constructors
func TestProductEvent_ConvenienceConstructors(t *testing.T) {
	details := map[string]string{"batch": "batch-123"}

	testCases := []struct {
		name         string
		constructor  func() *ProductEvent
		expectedType ProductEventType
		entityID     string
		resourceID   string
	}{
		{"NewProductCreatedEvent", func() *ProductEvent { return NewProductCreatedEvent("prod-1", details) }, ProductCreated, "prod-1", "prod-1"},
		{"NewProductUpdatedEvent", func() *ProductEvent { return NewProductUpdatedEvent("prod-2", details) }, ProductUpdated, "prod-2", "prod-2"},
		{"NewProductDeletedEvent", func() *ProductEvent { return NewProductDeletedEvent("prod-3", details) }, ProductDeleted, "prod-3", "prod-3"},
		{"NewProductImageAddedEvent", func() *ProductEvent { return NewProductImageAddedEvent("prod-4", "img-1", details) }, ProductImageAdded, "prod-4", "img-1"},
		{"NewProductImageUpdatedEvent", func() *ProductEvent { return NewProductImageUpdatedEvent("prod-5", "img-2", details) }, ProductImageUpdated, "prod-5", "img-2"},
		{"NewProductImageDeletedEvent", func() *ProductEvent { return NewProductImageDeletedEvent("prod-6", "img-3", details) }, ProductImageDeleted, "prod-6", "img-3"},
		{"NewProductIngestionStartedEvent", func() *ProductEvent { return NewProductIngestionStartedEvent("batch-1", details) }, ProductIngestionStarted, "", "batch-1"},
		{"NewProductIngestionCompletedEvent", func() *ProductEvent { return NewProductIngestionCompletedEvent("batch-2", details) }, ProductIngestionCompleted, "", "batch-2"},
		{"NewProductIngestionFailedEvent", func() *ProductEvent { return NewProductIngestionFailedEvent("batch-3", details) }, ProductIngestionFailed, "", "batch-3"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := tc.constructor()

			if event.EventType != tc.expectedType {
				t.Errorf("Expected event type %s, got %s", tc.expectedType, event.EventType)
			}
			if event.GetEntityID() != tc.entityID {
				t.Errorf("Expected entity ID '%s', got '%s'", tc.entityID, event.GetEntityID())
			}
			if event.GetResourceID() != tc.resourceID {
				t.Errorf("Expected resource ID '%s', got '%s'", tc.resourceID, event.GetResourceID())
			}
			if tc.name != "NewProductIngestionStartedEvent" && tc.name != "NewProductIngestionCompletedEvent" && tc.name != "NewProductIngestionFailedEvent" {
				if event.EventPayload.Details["batch"] != "batch-123" {
					t.Errorf("Expected details to contain batch data")
				}
			}
		})
	}
}

// TestEvent_TypeSafety tests type safety of events
func TestEvent_TypeSafety(t *testing.T) {
	// Test that CustomerEvent and ProductEvent are different types
	customerEvent := NewCustomerCreatedEvent("cust-1", nil)
	productEvent := NewProductCreatedEvent("prod-1", nil)

	// They should have different types
	if customerEvent.Type() == productEvent.Type() {
		t.Error("Customer and Product events should have different types")
	}

	// They should have different topics
	if customerEvent.Topic() == productEvent.Topic() {
		t.Error("Customer and Product events should have different topics")
	}

	// Test that we can assign to Event interface
	var customerAsEvent Event = customerEvent
	var productAsEvent Event = productEvent

	if customerAsEvent.Type() != "customer.created" {
		t.Error("Customer event interface should return correct type")
	}
	if productAsEvent.Type() != "product.created" {
		t.Error("Product event interface should return correct type")
	}
}

// TestEvent_JSONRoundTrip tests complete JSON round-trip for both event types
func TestEvent_JSONRoundTrip(t *testing.T) {
	testCases := []struct {
		name  string
		event Event
	}{
		{"CustomerEvent", NewCustomerCreatedEvent("cust-123", map[string]string{"key": "value"})},
		{"ProductEvent", NewProductCreatedEvent("prod-123", map[string]string{"key": "value"})},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal to JSON
			jsonData, err := tc.event.ToJSON()
			if err != nil {
				t.Fatalf("Failed to marshal %s: %v", tc.name, err)
			}

			// Verify it's valid JSON
			var jsonMap map[string]interface{}
			if err := json.Unmarshal(jsonData, &jsonMap); err != nil {
				t.Fatalf("Generated JSON is invalid: %v", err)
			}

			// Verify required fields exist
			if _, exists := jsonMap["id"]; !exists {
				t.Error("JSON should contain 'id' field")
			}
			if _, exists := jsonMap["type"]; !exists {
				t.Error("JSON should contain 'type' field")
			}
			if _, exists := jsonMap["timestamp"]; !exists {
				t.Error("JSON should contain 'timestamp' field")
			}
			if _, exists := jsonMap["payload"]; !exists {
				t.Error("JSON should contain 'payload' field")
			}
		})
	}
}

// TestEvent_InvalidJSON tests handling of invalid JSON
func TestEvent_InvalidJSON(t *testing.T) {
	// Test CustomerEventFactory with invalid JSON
	customerFactory := CustomerEventFactory{}
	_, err := customerFactory.FromJSON([]byte("invalid json"))
	if err == nil {
		t.Error("CustomerEventFactory should return error for invalid JSON")
	}

	// Test ProductEventFactory with invalid JSON
	productFactory := ProductEventFactory{}
	_, err = productFactory.FromJSON([]byte("invalid json"))
	if err == nil {
		t.Error("ProductEventFactory should return error for invalid JSON")
	}

	// Test with empty JSON
	_, err = customerFactory.FromJSON([]byte("{}"))
	if err != nil {
		t.Errorf("CustomerEventFactory should handle empty JSON object, got error: %v", err)
	}

	_, err = productFactory.FromJSON([]byte("{}"))
	if err != nil {
		t.Errorf("ProductEventFactory should handle empty JSON object, got error: %v", err)
	}
}

// TestEvent_UniqueIDs tests that events get unique IDs
func TestEvent_UniqueIDs(t *testing.T) {
	event1 := NewCustomerCreatedEvent("cust-1", nil)
	event2 := NewCustomerCreatedEvent("cust-1", nil)

	if event1.ID == event2.ID {
		t.Error("Events should have unique IDs")
	}

	if event1.ID == "" {
		t.Error("Event ID should not be empty")
	}

	if event2.ID == "" {
		t.Error("Event ID should not be empty")
	}

	// Verify IDs are valid UUIDs
	if _, err := uuid.Parse(event1.ID); err != nil {
		t.Errorf("Event ID should be valid UUID: %v", err)
	}
	if _, err := uuid.Parse(event2.ID); err != nil {
		t.Errorf("Event ID should be valid UUID: %v", err)
	}
}

// TestEvent_Timestamps tests that events have reasonable timestamps
func TestEvent_Timestamps(t *testing.T) {
	before := time.Now()
	event := NewCustomerCreatedEvent("cust-1", nil)
	after := time.Now()

	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Errorf("Event timestamp should be between %v and %v, got %v", before, after, event.Timestamp)
	}

	// Test that timestamp is not zero
	if event.Timestamp.IsZero() {
		t.Error("Event timestamp should not be zero")
	}
}

// TestEvent_EmptyDetails tests events with empty details
func TestEvent_EmptyDetails(t *testing.T) {
	// Test with nil details
	event1 := NewCustomerCreatedEvent("cust-1", nil)
	if event1.EventPayload.Details != nil {
		t.Error("Details should be nil when passed nil")
	}

	// Test with empty map
	event2 := NewCustomerCreatedEvent("cust-2", map[string]string{})
	if event2.EventPayload.Details == nil {
		t.Error("Details should not be nil when passed empty map")
	}
	if len(event2.EventPayload.Details) != 0 {
		t.Error("Details should be empty when passed empty map")
	}

	// Test marshaling/unmarshaling with nil details
	jsonData, err := event1.ToJSON()
	if err != nil {
		t.Fatalf("Failed to marshal event with nil details: %v", err)
	}

	factory := CustomerEventFactory{}
	reconstructed, err := factory.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to unmarshal event with nil details: %v", err)
	}

	if reconstructed.EventPayload.Details != nil {
		t.Error("Reconstructed event should have nil details")
	}
}

// TestEvent_EmptyIDs tests events with empty entity/resource IDs
func TestEvent_EmptyIDs(t *testing.T) {
	// Test ingestion events that have empty product IDs
	event := NewProductIngestionStartedEvent("batch-1", nil)

	if event.GetEntityID() != "" {
		t.Errorf("Expected empty entity ID, got '%s'", event.GetEntityID())
	}
	if event.GetResourceID() != "batch-1" {
		t.Errorf("Expected resource ID 'batch-1', got '%s'", event.GetResourceID())
	}

	// Test marshaling/unmarshaling
	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to marshal event with empty entity ID: %v", err)
	}

	factory := ProductEventFactory{}
	reconstructed, err := factory.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to unmarshal event with empty entity ID: %v", err)
	}

	if reconstructed.GetEntityID() != "" {
		t.Errorf("Reconstructed event should have empty entity ID, got '%s'", reconstructed.GetEntityID())
	}
	if reconstructed.GetResourceID() != "batch-1" {
		t.Errorf("Reconstructed event should have resource ID 'batch-1', got '%s'", reconstructed.GetResourceID())
	}
}
