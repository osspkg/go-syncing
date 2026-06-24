/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package circuitbreaker

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"go.osspkg.com/syncing"
)

var (
	ErrCircuitOpen       = errors.New("circuit breaker: is open")
	ErrLimitParallelCall = errors.New("circuit breaker: limit parallel call")
)

const (
	stateClosed   int32 = 0
	stateHalfOpen int32 = 1
	stateOpen     int32 = 2
)

type CircuitBreaker struct {
	state         int32
	failureCount  int32
	threshold     int32
	timeout       int64
	lastFailure   int64
	halfOpenToken int32 // 0 - blocked, 1 - can run 1 call
	parallel      syncing.Control
}

type Config struct {
	MaxParallel uint64
	Threshold   int32
	Timeout     time.Duration
}

func New(c Config) *CircuitBreaker {
	return &CircuitBreaker{
		state:     stateClosed,
		threshold: max(c.Threshold, 1),
		timeout:   int64(max(c.Timeout, time.Millisecond)),
		parallel:  syncing.NewControl(max(c.MaxParallel, 1)),
	}
}

func (cb *CircuitBreaker) Execute(
	ctx context.Context,
	call func(ctx context.Context) error,
) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(cb.timeout))
	defer cancel()

	if !cb.parallel.Acquire(ctx) {
		if err := ctx.Err(); err != nil {
			return err
		}
		return ErrLimitParallelCall
	}
	defer cb.parallel.Release()

	now := time.Now().UnixNano()
	state := atomic.LoadInt32(&cb.state)

	// -------------------   Open → HalfOpen ?   -------------------
	if state == stateOpen {
		last := atomic.LoadInt64(&cb.lastFailure)
		if now-last < cb.timeout {
			return ErrCircuitOpen
		}

		if atomic.CompareAndSwapInt32(&cb.state, stateOpen, stateHalfOpen) {
			atomic.StoreInt32(&cb.halfOpenToken, 1) // даём токен
			state = stateHalfOpen
		} else {
			state = atomic.LoadInt32(&cb.state)
		}
	}

	// -------------------   HalfOpen   -------------------
	if state == stateHalfOpen {
		if !atomic.CompareAndSwapInt32(&cb.halfOpenToken, 1, 0) {
			return ErrCircuitOpen
		}

		if err := call(ctx); err != nil {
			atomic.StoreInt64(&cb.lastFailure, now)
			atomic.StoreInt32(&cb.state, stateOpen)
			return err
		}

		atomic.StoreInt32(&cb.failureCount, 0)
		atomic.StoreInt32(&cb.state, stateClosed)

		return nil
	}

	// -------------------   Closed   -------------------

	if err := call(ctx); err != nil {
		newCount := atomic.AddInt32(&cb.failureCount, 1)
		if newCount >= cb.threshold &&
			atomic.CompareAndSwapInt32(&cb.state, stateClosed, stateOpen) {

			atomic.StoreInt64(&cb.lastFailure, now)
		}

		return err
	}

	atomic.StoreInt32(&cb.failureCount, 0)

	return nil
}
