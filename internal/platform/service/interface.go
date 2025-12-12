// Package service provides the foundational infrastructure for all service types.
// It implements Clean Architecture by providing reusable service lifecycle management
// that supports event-driven, HTTP, gRPC, and custom service implementations.
//
// Key interfaces:
//   - Service: Common lifecycle interface (Start, Stop, Health, Name)
//   - EventService: Event-specific extension with handler management
//
// Usage patterns:
//   - Event-driven services: Embed EventServiceBase
//   - HTTP services: Embed BaseService + add HTTP server
//   - Custom services: Implement Service interface directly
//
// Example usage:
//
//	// Event-driven service
//	type MyService struct {
//	    *service.EventServiceBase
//	    // domain-specific fields
//	}
//
//	func NewMyService(eventBus bus.Bus) *MyService {
//	    return &MyService{
//	        EventServiceBase: service.NewEventServiceBase("my-service", eventBus),
//	    }
//	}
package service

import (
	"context"
	"errors"
	"fmt"

	bus "go-shopping-poc/internal/platform/event/bus"
)

// Service defines the common interface for all service types
// This provides the basic lifecycle management that all services need
type Service interface {
	// Start begins the service operation
	Start(ctx context.Context) error

	// Stop gracefully shuts down the service
	Stop(ctx context.Context) error

	// Health returns the current health status of the service
	Health() error

	// Name returns the service name for identification
	Name() string
}

// EventService extends Service with event-specific functionality
type EventService interface {
	Service
	EventBus() bus.Bus
	HandlerCount() int
}

// BaseService provides a base implementation that can be embedded in specific services
type BaseService struct {
	name string
}

// NewBaseService creates a new base service with the given name
func NewBaseService(name string) *BaseService {
	return &BaseService{
		name: name,
	}
}

// Start provides a default no-op implementation - override in specific services
func (s *BaseService) Start(ctx context.Context) error {
	return nil
}

// Name returns the service name
func (s *BaseService) Name() string {
	return s.name
}

// Health returns nil (healthy) by default - override in specific services
func (s *BaseService) Health() error {
	return nil
}

// Stop provides a default no-op implementation - override if cleanup is needed
func (s *BaseService) Stop(ctx context.Context) error {
	return nil
}

// Common errors
var (
	ErrUnsupportedEventBus = errors.New("unsupported event bus type")
)

// ServiceError represents a service-specific error with context
type ServiceError struct {
	Service string
	Op      string
	Err     error
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("service %s: %s: %v", e.Service, e.Op, e.Err)
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}
