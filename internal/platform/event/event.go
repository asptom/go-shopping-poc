package event

// This package is deprecated. Use go-shopping-poc/internal/contracts/events instead.
// This file exists for backward compatibility during migration.

import "go-shopping-poc/internal/contracts/events"

// Re-export types from contracts for backward compatibility
type Event = events.Event
type EventFactory[T events.Event] = events.EventFactory[T]
