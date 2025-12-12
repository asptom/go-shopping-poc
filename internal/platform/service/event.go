package service

import (
	"errors"
)

// EventHandlerRegistration represents a registered event handler
type EventHandlerRegistration struct {
	EventType string
	Topic     string
	Active    bool
}

// EventServiceInfo provides information about an event service
type EventServiceInfo struct {
	Name         string
	HandlerCount int
	Topics       []string
	Healthy      bool
}

// GetEventServiceInfo returns information about the event service
func GetEventServiceInfo(s Service) (*EventServiceInfo, error) {
	if eventService, ok := s.(EventService); ok {
		topics := eventService.EventBus().ReadTopics()
		healthy := eventService.Health() == nil

		return &EventServiceInfo{
			Name:         eventService.Name(),
			HandlerCount: eventService.HandlerCount(),
			Topics:       topics,
			Healthy:      healthy,
		}, nil
	}

	return nil, &ServiceError{
		Service: s.Name(),
		Op:      "GetEventServiceInfo",
		Err:     ErrUnsupportedEventBus,
	}
}

// ListHandlers returns a list of registered handler information
func ListHandlers(s Service) ([]EventHandlerRegistration, error) {
	if _, ok := s.(EventService); ok {
		// For services that embed EventServiceBase, we can access the handlers
		if esb, ok := s.(*EventServiceBase); ok {
			var registrations []EventHandlerRegistration

			// For now, we can't easily extract the event type from the stored handlers
			// In a future enhancement, we could store more metadata
			for i := 0; i < len(esb.handlers); i++ {
				registrations = append(registrations, EventHandlerRegistration{
					EventType: "unknown", // Would need enhancement to extract actual type
					Topic:     "unknown", // Would need enhancement to extract topic
					Active:    true,
				})
			}

			return registrations, nil
		}
	}

	return nil, &ServiceError{
		Service: s.Name(),
		Op:      "ListHandlers",
		Err:     ErrUnsupportedEventBus,
	}
}

// ValidateEventBus checks if the event bus is properly configured
func ValidateEventBus(s Service) error {
	if eventService, ok := s.(EventService); ok {
		eventBus := eventService.EventBus()
		if eventBus == nil {
			return &ServiceError{
				Service: s.Name(),
				Op:      "ValidateEventBus",
				Err:     ErrUnsupportedEventBus,
			}
		}

		// Check if we can read topics
		topics := eventBus.ReadTopics()
		if len(topics) == 0 {
			return &ServiceError{
				Service: s.Name(),
				Op:      "ValidateEventBus",
				Err:     errors.New("no read topics configured"),
			}
		}

		return nil
	}

	return &ServiceError{
		Service: s.Name(),
		Op:      "ValidateEventBus",
		Err:     ErrUnsupportedEventBus,
	}
}
