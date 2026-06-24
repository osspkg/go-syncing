/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestUnit_New_MinimumCapacityOne(t *testing.T) {
	c := NewControl(0) // count 0 -> max(0,1) = 1
	sem := c.(*sem)
	if cap(sem.c) != 1 {
		t.Fatalf("expected capacity 1, got %d", cap(sem.c))
	}
}

func TestUnit_Acquire_NoContention(t *testing.T) {
	c := NewControl(2)
	ctx := context.Background()

	if !c.Acquire(ctx) {
		t.Fatal("first acquire should succeed")
	}
	if !c.Acquire(ctx) {
		t.Fatal("second acquire should succeed")
	}
	// now semaphore is full; further acquire would block
}

func TestUnit_Acquire_BlocksWhenFull(t *testing.T) {
	c := NewControl(1)
	ctx := context.Background()
	if !c.Acquire(ctx) {
		t.Fatal("acquire should succeed")
	}

	acquired := make(chan bool, 1)
	go func() {
		acquired <- c.Acquire(ctx)
	}()

	// The goroutine should be blocked; ensure it doesn't return immediately.
	select {
	case <-acquired:
		t.Fatal("acquire should have blocked but returned true")
	case <-time.After(50 * time.Millisecond):
		// expected: still blocked
	}
}

func TestUnit_Release_UnblocksAcquire(t *testing.T) {
	c := NewControl(1)
	ctx := context.Background()
	if !c.Acquire(ctx) {
		t.Fatal("first acquire failed")
	}

	acquired := make(chan bool, 1)
	go func() {
		acquired <- c.Acquire(ctx)
	}()

	// Give the goroutine time to block
	time.Sleep(10 * time.Millisecond)

	c.Release()

	select {
	case ok := <-acquired:
		if !ok {
			t.Fatal("acquire after release should succeed")
		}
	case <-time.After(time.Second):
		t.Fatal("acquire did not unblock after release")
	}
}

func TestUnit_Release_OnEmptyDoesNothing(t *testing.T) {
	c := NewControl(2)
	// Release on empty semaphore should not panic or change capacity.
	c.Release()
	c.Release()
	c.Release()

	// Still possible to acquire up to capacity.
	ctx := context.Background()
	if !c.Acquire(ctx) {
		t.Fatal("acquire after empty release should succeed")
	}
	if !c.Acquire(ctx) {
		t.Fatal("second acquire should succeed")
	}
}

func TestUnit_ContextCancellation_BeforeAcquire(t *testing.T) {
	c := NewControl(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	if c.Acquire(ctx) {
		t.Fatal("acquire with cancelled context should return false")
	}
	// The semaphore should still be free.
	ctx2 := context.Background()
	if !c.Acquire(ctx2) {
		t.Fatal("acquire after cancelled attempt should succeed")
	}
}

func TestUnit_ContextCancellation_WhileWaiting(t *testing.T) {
	c := NewControl(1)
	// Fill the semaphore
	if !c.Acquire(context.Background()) {
		t.Fatal("first acquire failed")
	}

	ctx, cancel := context.WithCancel(context.Background())
	acquired := make(chan bool, 1)
	go func() {
		acquired <- c.Acquire(ctx)
	}()

	// Wait for goroutine to start blocking
	time.Sleep(10 * time.Millisecond)

	cancel() // cancel while waiting

	select {
	case ok := <-acquired:
		if ok {
			t.Fatal("acquire should return false after context cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("acquire did not return after context cancellation")
	}

	// Verify that the slot was NOT taken by the cancelled acquire
	c.Release()
	if !c.Acquire(context.Background()) {
		t.Fatal("semaphore should be available after cancelled wait")
	}
}

func TestUnit_Concurrency_RespectsCapacity(t *testing.T) {
	const capacity = 3
	const goroutines = 10
	c := NewControl(capacity)

	ctx := context.Background()
	var wg sync.WaitGroup
	maxConcurrent := 0
	var mu sync.Mutex
	current := 0

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if c.Acquire(ctx) {
				mu.Lock()
				current++
				if current > maxConcurrent {
					maxConcurrent = current
				}
				mu.Unlock()

				time.Sleep(10 * time.Millisecond) // simulate work

				mu.Lock()
				current--
				mu.Unlock()
				c.Release()
			}
		}()
	}

	wg.Wait()

	if maxConcurrent > capacity {
		t.Fatalf("max concurrent acquisitions %d exceeded capacity %d", maxConcurrent, capacity)
	}
	if maxConcurrent < 1 {
		t.Fatal("no acquisitions succeeded")
	}
}

func TestUnit_InterfaceImplementation(t *testing.T) {
	// Compile-time check (this test mainly ensures the type satisfies Control)
	var _ Control = NewControl(1)
}
