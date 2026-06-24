/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import (
	"context"
	"testing"
	"time"
)

func TestUnit_Limiter_BasicTap(t *testing.T) {
	ctx := context.Background()
	lim := NewLimiter(ctx, 2, 100*time.Millisecond)

	// Initially two tokens should be available
	if !lim.Tap(ctx) {
		t.Error("first Tap should succeed")
	}
	if !lim.Tap(ctx) {
		t.Error("second Tap should succeed")
	}

	// Third Tap should fail immediately because no tokens left and context not cancelled
	ctxShort, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()
	if lim.Tap(ctxShort) {
		t.Error("third Tap should fail due to exhausted tokens")
	}
}

func TestUnit_Limiter_RefillAfterInterval(t *testing.T) {
	ctx := context.Background()
	const count = 2
	interval := 50 * time.Millisecond
	lim := NewLimiter(ctx, count, interval)

	// exhaust tokens
	for i := 0; i < count; i++ {
		if !lim.Tap(ctx) {
			t.Fatalf("initial token %d should be available", i)
		}
	}

	// Wait slightly longer than the interval to trigger a ticker refill
	time.Sleep(interval + 20*time.Millisecond)

	// Now tokens should be refilled (up to count)
	for i := 0; i < count; i++ {
		if !lim.Tap(ctx) {
			t.Errorf("token %d after refill should be available", i)
		}
	}
}

func TestUnit_Limiter_ContextCancelGlobal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	lim := NewLimiter(ctx, 5, time.Second)

	// Take one token first to ensure limiter is alive
	if !lim.Tap(context.Background()) {
		t.Fatal("first Tap should succeed")
	}

	cancel() // cancel limiter's base context

	// Now Tap should fail because lim.ctx is cancelled
	if lim.Tap(context.Background()) {
		t.Error("Tap should fail after global context cancel")
	}
}

func TestUnit_Limiter_ContextCancelPerTap(t *testing.T) {
	ctx := context.Background()
	lim := NewLimiter(ctx, 5, time.Second)

	// Create a cancelled context for a single Tap
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if lim.Tap(cancelCtx) {
		t.Error("Tap with cancelled context should fail")
	}
}

func TestUnit_Limiter_CountZero(t *testing.T) {
	ctx := context.Background()
	// Passing count <= 0 should be treated as 1
	lim := NewLimiter(ctx, 0, 100*time.Millisecond)

	// One token should be available
	if !lim.Tap(ctx) {
		t.Error("first Tap with effective count=1 should succeed")
	}
	// Second should fail
	ctxShort, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()
	if lim.Tap(ctxShort) {
		t.Error("second Tap should fail because only one token exists")
	}
}

func TestUnit_Limiter_ConcurrentSafety(t *testing.T) {
	ctx := context.Background()
	lim := NewLimiter(ctx, 100, 100*time.Millisecond)

	// Run many goroutines trying to tap concurrently
	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			// This may succeed or fail depending on tokens, but should never panic
			lim.Tap(context.Background())
		}()
	}
	// wait for all to finish
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestUnit_Limiter_TapWithTimeout(t *testing.T) {
	ctx := context.Background()
	lim := NewLimiter(ctx, 1, 200*time.Millisecond)

	// Exhaust the single token
	if !lim.Tap(ctx) {
		t.Fatal("first Tap should succeed")
	}

	// Try to Tap with a short timeout that expires before any refill
	ctxTimeout, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if lim.Tap(ctxTimeout) {
		t.Error("Tap should fail because no tokens and timeout fires")
	}
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Tap blocked too long: %v", elapsed)
	}
}

func TestUnit_Limiter_RefillDoesNotExceedCapacity(t *testing.T) {
	ctx := context.Background()
	const count = 5
	interval := 50 * time.Millisecond
	lim := NewLimiter(ctx, count, interval)

	// Wait long enough for several refills
	time.Sleep(2*interval + 20*time.Millisecond)

	// Try to take count tokens; they should be available but not more than count
	for i := 0; i < count; i++ {
		if !lim.Tap(ctx) {
			t.Errorf("token %d should be available after multiple intervals", i)
		}
	}
	// Next should fail immediately
	ctxShort, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()
	if lim.Tap(ctxShort) {
		t.Error("no more than count tokens should be available")
	}
}
