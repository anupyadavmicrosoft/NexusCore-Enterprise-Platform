package proxy

import (
	"sync"
	"time"

	"github.com/nexuscore/api-gateway/internal/metrics"
)

type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

type CircuitBreaker struct {
	name            string
	mu              sync.RWMutex
	state           State
	failureCount    int
	successCount    int
	totalCount      int
	failureRate     float64
	cooldown        time.Duration
	lastStateChange time.Time
	halfOpenLimit   int
}

func NewCircuitBreaker(name string, failureRate float64, cooldown time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:            name,
		state:           StateClosed,
		failureRate:     failureRate,
		cooldown:        cooldown,
		lastStateChange: time.Now(),
		halfOpenLimit:   5,
	}
	metrics.CircuitBreakerState.WithLabelValues(name).Set(float64(StateClosed))
	return cb
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen {
		if time.Since(cb.lastStateChange) > cb.cooldown {
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.failureCount = 0
			cb.totalCount = 0
			cb.lastStateChange = time.Now()
			metrics.CircuitBreakerState.WithLabelValues(cb.name).Set(float64(StateHalfOpen))
			return true
		}
		return false
	}
	return true
}

func (cb *CircuitBreaker) RecordResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalCount++
	if success {
		cb.successCount++
	} else {
		cb.failureCount++
	}

	if cb.state == StateHalfOpen {
		if !success {
			// Instant reversion to open on any half-open failure
			cb.state = StateOpen
			cb.lastStateChange = time.Now()
			metrics.CircuitBreakerState.WithLabelValues(cb.name).Set(float64(StateOpen))
		} else if cb.successCount >= cb.halfOpenLimit {
			// Successfully closed the breaker
			cb.state = StateClosed
			cb.failureCount = 0
			cb.successCount = 0
			cb.totalCount = 0
			cb.lastStateChange = time.Now()
			metrics.CircuitBreakerState.WithLabelValues(cb.name).Set(float64(StateClosed))
		}
	} else if cb.state == StateClosed {
		// Minimum sample size before evaluating failure rate
		if cb.totalCount >= 10 {
			rate := float64(cb.failureCount) / float64(cb.totalCount)
			if rate >= cb.failureRate {
				cb.state = StateOpen
				cb.lastStateChange = time.Now()
				metrics.CircuitBreakerState.WithLabelValues(cb.name).Set(float64(StateOpen))
			}
			// Reset counts periodically to slide the window
			if cb.totalCount >= 100 {
				cb.totalCount = 0
				cb.failureCount = 0
				cb.successCount = 0
			}
		}
	}
}

func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
