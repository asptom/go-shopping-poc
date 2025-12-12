package eventhandlers

import (
	"context"
	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/platform/logging"
)

// OnCustomerCreated handles CustomerCreated events
type OnCustomerCreated struct{}

// NewOnCustomerCreated creates a new CustomerCreated event handler
func NewOnCustomerCreated() *OnCustomerCreated {
	return &OnCustomerCreated{}
}

// Handle processes CustomerCreated events
func (h *OnCustomerCreated) Handle(ctx context.Context, event events.Event) error {
	var customerEvent events.CustomerEvent
	switch e := event.(type) {
	case events.CustomerEvent:
		customerEvent = e
	case *events.CustomerEvent:
		customerEvent = *e
	default:
		logging.Error("Eventreader: Expected CustomerEvent, got %T", event)
		return nil // Don't fail processing, just log and continue
	}

	if customerEvent.EventType != events.CustomerCreated {
		logging.Debug("Eventreader: Ignoring non-CustomerCreated event: %s", customerEvent.EventType)
		return nil
	}

	// Use platform utilities for consistent logging
	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(customerEvent.EventType),
		customerEvent.EventPayload.CustomerID,
		customerEvent.EventPayload.ResourceID)

	// Business logic for handling customer creation
	return h.processCustomerCreated(ctx, customerEvent)
}

// processCustomerCreated contains the actual business logic
func (h *OnCustomerCreated) processCustomerCreated(ctx context.Context, event events.CustomerEvent) error {
	customerID := event.EventPayload.CustomerID
	utils := handler.NewEventUtils()

	// Business logic for handling customer creation
	if err := h.sendWelcomeEmail(ctx, customerID); err != nil {
		utils.LogEventCompletion(ctx, string(event.EventType), customerID, err)
		// Continue processing even if email fails
	}

	if err := h.initializeCustomerPreferences(ctx, customerID); err != nil {
		utils.LogEventCompletion(ctx, string(event.EventType), customerID, err)
		// Continue processing even if preferences fail
	}

	if err := h.updateCustomerAnalytics(ctx, customerID); err != nil {
		utils.LogEventCompletion(ctx, string(event.EventType), customerID, err)
		// Continue processing even if analytics fail
	}

	if err := h.createCustomerProfile(ctx, customerID); err != nil {
		utils.LogEventCompletion(ctx, string(event.EventType), customerID, err)
		// Continue processing even if profile creation fails
	}

	// Log successful completion
	utils.LogEventCompletion(ctx, string(event.EventType), customerID, nil)
	return nil
}

// sendWelcomeEmail sends a welcome email to the new customer
func (h *OnCustomerCreated) sendWelcomeEmail(ctx context.Context, customerID string) error {
	// Check for context cancellation before proceeding
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Simulate email service integration
	// In a real implementation, this would call an email service:
	// emailService := getEmailService()
	// return emailService.SendWelcomeEmail(ctx, customerID)
	_ = customerID // Will be used when email service is implemented

	return nil
}

// initializeCustomerPreferences sets up default preferences for the new customer
func (h *OnCustomerCreated) initializeCustomerPreferences(ctx context.Context, customerID string) error {
	// Check for context cancellation before proceeding
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Simulate preference initialization
	// In a real implementation, this would:
	// - Set default communication preferences
	// - Initialize marketing preferences
	// - Set up default notification settings
	// preferenceService := getPreferenceService()
	// return preferenceService.InitializeDefaults(ctx, customerID)
	_ = customerID // Will be used when preference service is implemented

	return nil
}

// updateCustomerAnalytics updates analytics systems with the new customer data
func (h *OnCustomerCreated) updateCustomerAnalytics(ctx context.Context, customerID string) error {
	// Check for context cancellation before proceeding
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Simulate analytics update
	// In a real implementation, this would:
	// - Update customer acquisition metrics
	// - Track registration source
	// - Update demographic data
	// analyticsService := getAnalyticsService()
	// return analyticsService.TrackNewCustomer(ctx, customerID)
	_ = customerID // Will be used when analytics service is implemented

	return nil
}

// createCustomerProfile creates additional customer profiles in external systems
func (h *OnCustomerCreated) createCustomerProfile(ctx context.Context, customerID string) error {
	// Check for context cancellation before proceeding
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Simulate profile creation in external systems
	// In a real implementation, this would:
	// - Create CRM profile
	// - Set up loyalty program account
	// - Initialize customer support profile
	// crmService := getCRMService()
	// loyaltyService := getLoyaltyService()
	// if err := crmService.CreateCustomerProfile(ctx, customerID); err != nil {
	//     return err
	// }
	// return loyaltyService.CreateAccount(ctx, customerID)
	_ = customerID // Will be used when CRM and loyalty services are implemented

	return nil
}

// EventType returns the event type this handler processes
func (h *OnCustomerCreated) EventType() string {
	return string(events.CustomerCreated)
}

// CreateFactory returns the event factory for this handler
func (h *OnCustomerCreated) CreateFactory() events.EventFactory[events.CustomerEvent] {
	return events.CustomerEventFactory{}
}

// CreateHandler returns the handler function
func (h *OnCustomerCreated) CreateHandler() bus.HandlerFunc[events.CustomerEvent] {
	return func(ctx context.Context, event events.CustomerEvent) error {
		return h.Handle(ctx, event)
	}
}

// Ensure OnCustomerCreated implements the shared interfaces
var _ handler.EventHandler = (*OnCustomerCreated)(nil)
var _ handler.HandlerFactory[events.CustomerEvent] = (*OnCustomerCreated)(nil)
