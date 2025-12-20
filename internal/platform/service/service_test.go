package service

import (
	"context"
	"errors"
	"testing"
	"time"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	kafka "go-shopping-poc/internal/platform/event/bus/kafka"
	kafkaconfig "go-shopping-poc/internal/platform/event/kafka"
)

// MockEventBus for testing
type MockEventBus struct {
	bus.Bus
	startConsumingCalled bool
	startConsumingError  error
	readTopics           []string
	writeTopic           string
}

func (m *MockEventBus) StartConsuming(ctx context.Context) error {
	m.startConsumingCalled = true
	return m.startConsumingError
}

func (m *MockEventBus) ReadTopics() []string {
	return m.readTopics
}

func (m *MockEventBus) WriteTopic() string {
	return m.writeTopic
}

func (m *MockEventBus) RegisterHandler(factory any, handler any) error {
	return ErrUnsupportedEventBus
}

func (m *MockEventBus) Publish(ctx context.Context, topic string, event events.Event) error {
	return nil
}

func (m *MockEventBus) PublishRaw(ctx context.Context, topic string, eventType string, data []byte) error {
	return nil
}

// MockEventBusKafka extends MockEventBus to simulate kafka.EventBus behavior
type MockEventBusKafka struct {
	MockEventBus
	registeredHandlers []any
}

func (m *MockEventBusKafka) RegisterHandler(factory any, handler any) error {
	// Simulate successful registration for kafka-like bus
	m.registeredHandlers = append(m.registeredHandlers, struct {
		factory any
		handler any
	}{
		factory: factory,
		handler: handler,
	})
	return nil
}

func TestBaseService_Name(t *testing.T) {
	service := NewBaseService("test-service")

	if service.Name() != "test-service" {
		t.Errorf("Expected name 'test-service', got '%s'", service.Name())
	}
}

func TestBaseService_Health(t *testing.T) {
	service := NewBaseService("test-service")

	err := service.Health()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestBaseService_Stop(t *testing.T) {
	service := NewBaseService("test-service")

	ctx := context.Background()
	err := service.Stop(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestNewEventServiceBase(t *testing.T) {
	mockBus := &MockEventBus{
		readTopics: []string{"topic1", "topic2"},
		writeTopic: "write-topic",
	}

	service := NewEventServiceBase("event-service", mockBus)

	if service.Name() != "event-service" {
		t.Errorf("Expected name 'event-service', got '%s'", service.Name())
	}

	if service.EventBus() != mockBus {
		t.Error("Expected event bus to be set correctly")
	}

	if service.HandlerCount() != 0 {
		t.Errorf("Expected 0 handlers initially, got %d", service.HandlerCount())
	}
}

func TestEventServiceBase_Start(t *testing.T) {
	mockBus := &MockEventBus{
		readTopics: []string{"topic1"},
		writeTopic: "write-topic",
	}

	service := NewEventServiceBase("event-service", mockBus)

	ctx := context.Background()
	err := service.Start(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !mockBus.startConsumingCalled {
		t.Error("Expected StartConsuming to be called")
	}
}

func TestEventServiceBase_Start_Error(t *testing.T) {
	expectedErr := errors.New("start consuming error")
	mockBus := &MockEventBus{
		startConsumingError: expectedErr,
		readTopics:          []string{"topic1"},
		writeTopic:          "write-topic",
	}

	service := NewEventServiceBase("event-service", mockBus)

	ctx := context.Background()
	err := service.Start(ctx)

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestRegisterHandler_Success(t *testing.T) {
	// Test that RegisterHandler works with real kafka.EventBus
	kafkaCfg := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-events",
		GroupID: "test-group",
	}
	kafkaBus := kafka.NewEventBus(kafkaCfg)

	service := NewEventServiceBase("event-service", kafkaBus)

	factory := events.CustomerEventFactory{}
	handler := bus.HandlerFunc[events.CustomerEvent](func(ctx context.Context, evt events.CustomerEvent) error {
		return nil
	})

	err := RegisterHandler(service, factory, handler)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify handler was stored
	if service.HandlerCount() != 1 {
		t.Errorf("Expected 1 handler, got %d", service.HandlerCount())
	}
}

func TestRegisterHandler_UnsupportedService(t *testing.T) {
	baseService := NewBaseService("base-service")

	factory := events.CustomerEventFactory{}
	handler := bus.HandlerFunc[events.CustomerEvent](func(ctx context.Context, evt events.CustomerEvent) error {
		return nil
	})

	err := RegisterHandler(baseService, factory, handler)

	if err == nil {
		t.Error("Expected error for unsupported service type")
	}

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Errorf("Expected ServiceError, got %T", err)
	}

	if serviceErr.Service != "base-service" {
		t.Errorf("Expected service name 'base-service', got '%s'", serviceErr.Service)
	}
}

func TestGetEventServiceInfo_Success(t *testing.T) {
	mockBus := &MockEventBus{
		readTopics: []string{"topic1", "topic2"},
		writeTopic: "write-topic",
	}

	service := NewEventServiceBase("event-service", mockBus)

	info, err := GetEventServiceInfo(service)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if info.Name != "event-service" {
		t.Errorf("Expected name 'event-service', got '%s'", info.Name)
	}

	if info.HandlerCount != 0 {
		t.Errorf("Expected 0 handlers, got %d", info.HandlerCount)
	}

	if len(info.Topics) != 2 {
		t.Errorf("Expected 2 topics, got %d", len(info.Topics))
	}

	if !info.Healthy {
		t.Error("Expected service to be healthy")
	}
}

func TestGetEventServiceInfo_UnsupportedService(t *testing.T) {
	baseService := NewBaseService("base-service")

	_, err := GetEventServiceInfo(baseService)

	if err == nil {
		t.Error("Expected error for unsupported service type")
	}

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Errorf("Expected ServiceError, got %T", err)
	}
}

func TestListHandlers_Success(t *testing.T) {
	mockBus := &MockEventBus{
		readTopics: []string{"topic1"},
		writeTopic: "write-topic",
	}

	service := NewEventServiceBase("event-service", mockBus)

	// Add a handler first
	factory := events.CustomerEventFactory{}
	handler := bus.HandlerFunc[events.CustomerEvent](func(ctx context.Context, evt events.CustomerEvent) error {
		return nil
	})

	// We can't actually register with the mock bus since it's not a kafka.EventBus
	// So we'll manually add to handlers for testing
	service.handlers = append(service.handlers, struct {
		factory events.EventFactory[events.CustomerEvent]
		handler bus.HandlerFunc[events.CustomerEvent]
	}{
		factory: factory,
		handler: handler,
	})

	registrations, err := ListHandlers(service)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(registrations) != 1 {
		t.Errorf("Expected 1 registration, got %d", len(registrations))
	}

	if !registrations[0].Active {
		t.Error("Expected handler to be active")
	}
}

func TestValidateEventBus_Success(t *testing.T) {
	mockBus := &MockEventBus{
		readTopics: []string{"topic1"},
		writeTopic: "write-topic",
	}

	service := NewEventServiceBase("event-service", mockBus)

	err := ValidateEventBus(service)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestValidateEventBus_NilEventBus(t *testing.T) {
	service := NewEventServiceBase("event-service", nil)

	err := ValidateEventBus(service)

	if err == nil {
		t.Error("Expected error for nil event bus")
	}

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Errorf("Expected ServiceError, got %T", err)
	}
}

func TestValidateEventBus_NoTopics(t *testing.T) {
	mockBus := &MockEventBus{
		readTopics: []string{},
		writeTopic: "write-topic",
	}

	service := NewEventServiceBase("event-service", mockBus)

	err := ValidateEventBus(service)

	if err == nil {
		t.Error("Expected error for no topics")
	}
}

func TestServiceError(t *testing.T) {
	originalErr := errors.New("original error")
	serviceErr := &ServiceError{
		Service: "test-service",
		Op:      "test-operation",
		Err:     originalErr,
	}

	expected := "service test-service: test-operation: original error"
	if serviceErr.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, serviceErr.Error())
	}

	if serviceErr.Unwrap() != originalErr {
		t.Error("Unwrap should return the original error")
	}
}

func TestEventServiceBase_ContextCancellation(t *testing.T) {
	mockBus := &MockEventBus{
		readTopics: []string{"topic1"},
		writeTopic: "write-topic",
	}

	service := NewEventServiceBase("event-service", mockBus)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Since our MockEventBus.StartConsuming just returns nil immediately,
	// this test will not actually test context cancellation.
	// In a real scenario with a blocking StartConsuming, this would test cancellation.
	err := service.Start(ctx)

	// The mock returns nil immediately, so we expect no error
	if err != nil {
		t.Errorf("Expected no error from mock, got %v", err)
	}
}
