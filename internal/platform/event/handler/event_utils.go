// Package handler provides domain-agnostic event processing utilities.
// These utilities implement common event handling patterns that work with
// any event type, promoting code reuse and consistent behavior.
//
// Available utilities:
//   - EventUtils: Core validation, logging, and processing patterns
//   - EventTypeMatcher: Generic event type matching and filtering
//
// Usage example:
//
//	utils := handler.NewEventUtils()
//	err := utils.HandleEventWithValidation(ctx, event, func(ctx context.Context, event events.Event) error {
//	    // Business logic here
//	    utils.LogEventProcessing(ctx, event.Type(), utils.GetEntityID(event), utils.GetResourceID(event))
//	    return nil
//	})
package handler

import (
	"context"
	"fmt"
	"log"

	"go-shopping-poc/internal/contracts/events"
)

// EventUtils provides reusable event handling utilities
// These utilities are domain-agnostic and can be used with any event type
type EventUtils struct{}

// NewEventUtils creates a new event utilities instance
func NewEventUtils() *EventUtils {
	return &EventUtils{}
}

// ValidateEvent validates common event fields for any event type
// This provides reusable validation patterns that apply to all events
func (u *EventUtils) ValidateEvent(ctx context.Context, event events.Event) error {
	if event == nil {
		log.Printf("[ERROR] Event validation failed: event is nil")
		return fmt.Errorf("event cannot be nil")
	}

	// Generic validation - domain-specific validation should be in service layer
	if event.Type() == "" {
		log.Printf("[ERROR] Event validation failed: missing event type")
		return fmt.Errorf("event type is required")
	}

	if event.Topic() == "" {
		log.Printf("[ERROR] Event validation failed: missing topic")
		return fmt.Errorf("event topic is required")
	}

	log.Printf("[DEBUG] Generic event validation passed for event type %s", event.Type())
	return nil
}

// LogEventProcessing provides standardized logging for event processing
func (u *EventUtils) LogEventProcessing(ctx context.Context, eventType string, entityID string, resourceID string) {
	if resourceID != "" {
		log.Printf("[INFO] Processing %s event: entity=%s, resource=%s", eventType, entityID, resourceID)
	} else {
		log.Printf("[INFO] Processing %s event: entity=%s", eventType, entityID)
	}
}

// LogEventCompletion provides standardized logging for completed event processing
func (u *EventUtils) LogEventCompletion(ctx context.Context, eventType string, entityID string, err error) {
	if err != nil {
		log.Printf("[ERROR] Failed to process %s event for entity %s: %v", eventType, entityID, err)
	} else {
		log.Printf("[INFO] Successfully processed %s event for entity %s", eventType, entityID)
	}
}

// HandleEventWithValidation combines validation and processing with standardized error handling
// This is a reusable pattern for any event handler that needs validation
func (u *EventUtils) HandleEventWithValidation(
	ctx context.Context,
	event events.Event,
	processor func(context.Context, events.Event) error,
) error {
	// Validate the event first
	if err := u.ValidateEvent(ctx, event); err != nil {
		return fmt.Errorf("event validation failed: %w", err)
	}

	// Process the event
	if err := processor(ctx, event); err != nil {
		return fmt.Errorf("event processing failed: %w", err)
	}

	return nil
}

// SafeEventProcessing safely processes events with panic recovery
// This ensures that a single event failure doesn't crash the entire event processor
func (u *EventUtils) SafeEventProcessing(
	ctx context.Context,
	event events.Event,
	processor func(context.Context, events.Event) error,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Panic recovered during event processing: %v", r)
			err = fmt.Errorf("panic during event processing: %v", r)
		}
	}()

	return processor(ctx, event)
}

// GetEntityID extracts the entity ID from the event
// This provides a generic way to get the primary entity ID from any event
func (u *EventUtils) GetEntityID(event events.Event) string {
	if event == nil {
		return ""
	}
	return event.GetEntityID()
}

// GetEventID extracts the entity ID from the event (alias for GetEntityID)
// This provides backward compatibility for existing code
func (u *EventUtils) GetEventID(event events.Event) string {
	return u.GetEntityID(event)
}

// GetEventType extracts the event type from the event
// This provides a generic way to get the event type from any event
func (u *EventUtils) GetEventType(event events.Event) string {
	if event == nil {
		return "unknown"
	}
	return event.Type()
}

// GetEventTopic extracts the topic from the event
// This provides a generic way to get the topic from any event
func (u *EventUtils) GetEventTopic(event events.Event) string {
	if event == nil {
		return "unknown"
	}
	return event.Topic()
}

// GetResourceID extracts the resource ID from the event
// This provides a generic way to get the secondary resource ID from any event
func (u *EventUtils) GetResourceID(event events.Event) string {
	if event == nil {
		return ""
	}
	return event.GetResourceID()
}

// EventTypeMatcher provides generic event type matching utilities
type EventTypeMatcher struct{}

// NewEventTypeMatcher creates a new event type matcher
func NewEventTypeMatcher() *EventTypeMatcher {
	return &EventTypeMatcher{}
}

// MatchEventType checks if an event matches any of the provided event types
// This is useful for handlers that need to process multiple event types
func (m *EventTypeMatcher) MatchEventType(event events.Event, eventTypes ...string) bool {
	if event == nil {
		return false
	}

	eventType := event.Type()
	for _, matchType := range eventTypes {
		if eventType == matchType {
			return true
		}
	}

	return false
}

// IsEventType checks if the event matches a specific event type
// This provides a generic way to check event types without domain knowledge
func (m *EventTypeMatcher) IsEventType(event events.Event, eventType string) bool {
	if event == nil {
		return false
	}
	return event.Type() == eventType
}
