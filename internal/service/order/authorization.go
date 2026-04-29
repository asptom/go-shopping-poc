package order

import (
	"context"
	"fmt"
	"time"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/auth"
)

// VerifyCustomerIdentity resolves a JWT to a CustomerIdentity.
// Fast path: lookup in the in-memory cache (~0.1 ms).
// Slow path: synchronous Kafka request/response (~50-200 ms) only on cache miss.
func (s *OrderService) VerifyCustomerIdentity(ctx context.Context, claims *auth.Claims) (*CustomerIdentity, error) {
	if claims.Email == "" {
		return nil, fmt.Errorf("missing email in token")
	}
	if claims.Subject == "" {
		return nil, fmt.Errorf("missing sub in token")
	}

	// Fast path: cache lookup
	if identity, ok := s.identityCache.Get(claims.Subject); ok {
		return &identity, nil
	}

	// Slow path: request customer service via Kafka
	return s.verifyViaKafka(ctx, claims)
}

func (s *OrderService) verifyViaKafka(ctx context.Context, claims *auth.Claims) (*CustomerIdentity, error) {
	requestID := fmt.Sprintf("verify-%s-%d", claims.Subject, time.Now().UnixNano())
	ch := make(chan verificationResult, 1)

	s.mu.Lock()
	s.verificationCallbacks[requestID] = ch
	s.mu.Unlock()

	reqEvent := events.NewCustomerIdentityVerificationRequestedEvent(
		requestID, claims.Email, claims.Subject,
	)
	if err := s.infrastructure.EventBus.Publish(ctx, reqEvent.Topic(), reqEvent); err != nil {
		s.mu.Lock()
		delete(s.verificationCallbacks, requestID)
		s.mu.Unlock()
		return nil, fmt.Errorf("failed to publish verification request: %w", err)
	}

	select {
	case result := <-ch:
		// Cache the result for next time (only on success)
		if result.identity != nil {
			s.identityCache.Set(claims.Subject, *result.identity)
		}
		return result.identity, result.err
	case <-time.After(5 * time.Second):
		s.mu.Lock()
		delete(s.verificationCallbacks, requestID)
		s.mu.Unlock()
		return nil, fmt.Errorf("identity verification timed out after 5s")
	case <-ctx.Done():
		s.mu.Lock()
		delete(s.verificationCallbacks, requestID)
		s.mu.Unlock()
		return nil, ctx.Err()
	}
}
