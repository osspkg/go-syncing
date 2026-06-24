/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestUnit_NewGroup(t *testing.T) {
	ctx := context.Background()
	g := NewGroup(ctx)

	if g == nil {
		t.Fatal("expected non-nil Group")
	}
}

func TestUnit_AddDoneWait(t *testing.T) {
	g := NewGroup(context.Background())

	var counter int32
	g.Add(2)
	go func() {
		defer g.Done()
		atomic.AddInt32(&counter, 1)
	}()
	go func() {
		defer g.Done()
		atomic.AddInt32(&counter, 1)
	}()

	g.Wait()
	if c := atomic.LoadInt32(&counter); c != 2 {
		t.Fatalf("counter = %d, want 2", c)
	}
}

func TestUnit_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // avoid context leak if test fails early

	g := NewGroup(ctx)

	started := make(chan struct{})
	done := make(chan struct{})

	g.Background("test-cancel", func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		close(done)
	})

	// wait until goroutine starts
	<-started
	// cancel the group
	g.Cancel()

	select {
	case <-done:
		// ok
	case <-time.After(time.Second):
		t.Fatal("goroutine did not finish after cancel")
	}
}

func TestUnit_BackgroundRunsInGoroutine(t *testing.T) {
	g := NewGroup(context.Background())

	var v int32
	done := make(chan struct{})

	g.Background("bg", func(ctx context.Context) {
		atomic.StoreInt32(&v, 1)
		close(done)
	})

	// background should have been called and done channel closed
	select {
	case <-done:
		if atomic.LoadInt32(&v) != 1 {
			t.Error("expected v to be 1")
		}
	case <-time.After(time.Second):
		t.Fatal("background goroutine did not start")
	}
	// ensure Wait() completes
	g.Wait()
}

func TestUnit_RunIsSynchronous(t *testing.T) {
	g := NewGroup(context.Background())

	var step string
	g.Run("sync-run", func(ctx context.Context) {
		step += "A"
	})
	// After Run returns, step should already contain "A"
	if step != "A" {
		t.Fatalf("step = %q, want %q", step, "A")
	}
}

func TestUnit_OnPanicCalled(t *testing.T) {
	g := NewGroup(context.Background())

	panicErr := make(chan error, 1)
	g.OnPanic(func(err error) {
		panicErr <- err
	})

	g.Background("panic-goroutine", func(ctx context.Context) {
		panic("boom")
	})

	select {
	case err := <-panicErr:
		expected := "panic-goroutine: boom"
		if err.Error() != expected {
			t.Fatalf("panic error = %q, want %q", err.Error(), expected)
		}
	case <-time.After(time.Second):
		t.Fatal("OnPanic was not called")
	}

	// Wait should not hang (recovered)
	g.Wait()
}

func TestUnit_OnPanicNotSet(t *testing.T) {
	// Should not crash even if OnPanic not set
	g := NewGroup(context.Background())

	done := make(chan struct{})
	g.Background("no-panic-handler", func(ctx context.Context) {
		panic("ignored")
	})

	go func() {
		g.Wait()
		close(done)
	}()

	select {
	case <-done:
		// pass
	case <-time.After(time.Second):
		t.Fatal("Wait() blocked after panic without OnPanic")
	}
}

func TestUnit_PanicErrorFormat(t *testing.T) {
	g := NewGroup(context.Background())
	errCh := make(chan error, 1)
	g.OnPanic(func(err error) {
		errCh <- err
	})

	g.Background("worker-1", func(ctx context.Context) {
		panic(errors.New("custom error"))
	})

	select {
	case err := <-errCh:
		if err.Error() != "worker-1: custom error" {
			t.Errorf("unexpected error string: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("OnPanic not called")
	}
	g.Wait()
}

func TestUnit_ContextPropagation(t *testing.T) {
	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()

	g := NewGroup(parentCtx)

	var childCtx context.Context
	done := make(chan struct{})

	g.Background("ctx-test", func(ctx context.Context) {
		childCtx = ctx
		close(done)
	})

	select {
	case <-done:
		// continue
	case <-time.After(time.Second):
		t.Fatal("background not started")
	}

	// child context should be derived from parent
	if childCtx != g.(*_group).globalCtx {
		t.Error("context passed to call is not the group's global context")
	}

	// Cancelling parent should cancel child
	parentCancel()
	select {
	case <-childCtx.Done():
		// ok
	case <-time.After(time.Second):
		t.Error("child context was not cancelled when parent was cancelled")
	}
}

func TestUnit_WaitAfterCancel(t *testing.T) {
	g := NewGroup(context.Background())

	var steps int32
	g.Background("work", func(ctx context.Context) {
		<-ctx.Done()
		atomic.AddInt32(&steps, 1)
	})

	g.Cancel() // cancels and waits for goroutine to finish

	if atomic.LoadInt32(&steps) != 1 {
		t.Fatal("goroutine did not finish after Cancel")
	}
}

func TestUnit_MultipleBackgroundAndWait(t *testing.T) {
	g := NewGroup(context.Background())
	const n = 10
	var counter int32

	for i := 0; i < n; i++ {
		g.Background("worker", func(ctx context.Context) {
			atomic.AddInt32(&counter, 1)
		})
	}
	g.Wait()

	if c := atomic.LoadInt32(&counter); c != n {
		t.Fatalf("counter = %d, want %d", c, n)
	}
}

func TestUnit_AddNegativePanics(t *testing.T) {
	// sync.WaitGroup panics when delta + counter < 0 after Add or Done.
	// We just verify that Add(-1) panics when counter is 0.
	g := NewGroup(context.Background())
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative delta")
		}
	}()
	g.Add(-1)
}

func TestUnit_ConcurrentAddAndWait(t *testing.T) {
	// Not a real race but ensures no deadlock when adding while waiting.
	g := NewGroup(context.Background())
	var wg sync.WaitGroup
	const n = 100
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.Add(1)
			go func() {
				g.Done()
			}()
		}()
	}
	wg.Wait()
	g.Wait()
}

func TestUnit_RunThenCancel(t *testing.T) {
	// Run blocks until the function returns, but we can cancel context from another goroutine.
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := NewGroup(parentCtx)

	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		g.Run("long-run", func(ctx context.Context) {
			close(started)
			<-ctx.Done()
			close(done)
		})
	}()

	<-started
	cancel() // cancel parent context

	select {
	case <-done:
		// Run returned after context cancelled
	case <-time.After(time.Second):
		t.Fatal("Run did not finish after parent cancel")
	}
}

func TestUnit_OnPanicConcurrency(t *testing.T) {
	// Multiple panics should each trigger OnPanic callback.
	g := NewGroup(context.Background())
	var (
		panicCount int32
		mu         sync.Mutex
		errorsList []string
	)
	g.OnPanic(func(err error) {
		atomic.AddInt32(&panicCount, 1)
		mu.Lock()
		errorsList = append(errorsList, err.Error())
		mu.Unlock()
	})

	const n = 5
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("p%d", i)
		g.Background(name, func(ctx context.Context) {
			panic(name)
		})
	}
	g.Wait()

	if c := atomic.LoadInt32(&panicCount); c != n {
		t.Fatalf("OnPanic called %d times, want %d", c, n)
	}
	if len(errorsList) != n {
		t.Fatalf("recorded %d errors, want %d", len(errorsList), n)
	}
}

func TestUnit_DoubleCancel(t *testing.T) {
	g := NewGroup(context.Background())
	g.Cancel()
	// second cancel should not panic
	g.Cancel()
}

func TestUnit_BackgroundAfterCancel(t *testing.T) {
	g := NewGroup(context.Background())
	g.Cancel()

	// Adding work after cancel should still work, but context is already cancelled.
	done := make(chan struct{})
	g.Background("post-cancel", func(ctx context.Context) {
		select {
		case <-ctx.Done():
			close(done)
		default:
			// should be done immediately
		}
	})

	select {
	case <-done:
		// ok
	case <-time.After(time.Second):
		t.Fatal("post-cancel goroutine did not notice cancelled context")
	}
	g.Wait()
}
