/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestUnit_Execute_Success(t *testing.T) {
	cb := New(Config{
		MaxParallel: 10,
		Threshold:   3,
		Timeout:     time.Second,
	})

	ctx := context.Background()
	var callCount int32
	fn := func(ctx context.Context) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}

	for i := 0; i < 5; i++ {
		if err := cb.Execute(ctx, fn); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if atomic.LoadInt32(&callCount) != 5 {
		t.Fatalf("expected 5 calls, got %d", callCount)
	}
}

func TestUnit_Execute_OpenOnThreshold(t *testing.T) {
	cb := New(Config{
		MaxParallel: 10,
		Threshold:   2,
		Timeout:     200 * time.Millisecond,
	})
	ctx := context.Background()

	// Две ошибки подряд открывают цепь
	if err := cb.Execute(ctx, failFn("e1")); err == nil {
		t.Fatal("expected error")
	}
	if err := cb.Execute(ctx, failFn("e2")); err == nil {
		t.Fatal("expected error")
	}

	// Третий вызов должен вернуть ErrCircuitOpen
	if err := cb.Execute(ctx, okFn); err != ErrCircuitOpen {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestUnit_Execute_OpenTimeoutHalfOpenSuccess(t *testing.T) {
	timeout := 50 * time.Millisecond
	cb := New(Config{
		MaxParallel: 10,
		Threshold:   1,
		Timeout:     timeout,
	})
	ctx := context.Background()

	// Открываем цепь одной ошибкой
	cb.Execute(ctx, failFn("fail"))

	// Сразу после – ErrCircuitOpen
	if err := cb.Execute(ctx, okFn); err != ErrCircuitOpen {
		t.Fatal("expected ErrCircuitOpen immediately")
	}

	// Ждём таймаут и пробуем – должно получиться (HalfOpen → Closed)
	time.Sleep(timeout + 10*time.Millisecond)

	var called bool
	err := cb.Execute(ctx, func(ctx context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected success after timeout, got %v", err)
	}
	if !called {
		t.Fatal("function was not called")
	}

	// Цепь снова Closed, можно вызывать дальше
	if err := cb.Execute(ctx, okFn); err != nil {
		t.Fatalf("expected subsequent call to succeed, got %v", err)
	}
}

func TestUnit_Execute_HalfOpenFailure(t *testing.T) {
	timeout := 50 * time.Millisecond
	cb := New(Config{
		MaxParallel: 10,
		Threshold:   1,
		Timeout:     timeout,
	})
	ctx := context.Background()

	cb.Execute(ctx, failFn("fail"))
	time.Sleep(timeout + 10*time.Millisecond)

	// Пробный вызов с ошибкой снова размыкает цепь
	err := cb.Execute(ctx, failFn("fail again"))
	if err == nil {
		t.Fatal("expected error from half-open call")
	}

	// Теперь опять ErrCircuitOpen
	if err := cb.Execute(ctx, okFn); err != ErrCircuitOpen {
		t.Fatalf("expected ErrCircuitOpen after failed half-open, got %v", err)
	}
}

func TestUnit_Execute_HalfOpenSingleProbe(t *testing.T) {
	timeout := 50 * time.Millisecond
	cb := New(Config{
		MaxParallel: 10,
		Threshold:   1,
		Timeout:     timeout,
	})
	ctx := context.Background()

	cb.Execute(ctx, failFn("fail"))
	time.Sleep(timeout + 10*time.Millisecond)

	var probeCount int32
	probe := func(ctx context.Context) error {
		atomic.AddInt32(&probeCount, 1)
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			cb.Execute(ctx, probe)
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&probeCount) != 1 {
		t.Fatalf("expected exactly 1 probe execution, got %d", probeCount)
	}
}

func TestUnit_Execute_MaxParallel(t *testing.T) {
	cb := New(Config{
		MaxParallel: 1,
		Threshold:   10,
		Timeout:     time.Second,
	})

	started := make(chan struct{})
	block := make(chan struct{})

	go func() {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			close(started)
			<-block
			return nil
		})
	}()

	<-started // первый вызов занял слот

	// Второй вызов должен сразу вернуть ErrLimitParallelCall
	err := cb.Execute(context.Background(), okFn)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected ErrLimitParallelCall, got %v", err)
	}

	close(block) // разрешаем первому завершиться
	time.Sleep(10 * time.Millisecond)

	// Теперь слот свободен
	if err := cb.Execute(context.Background(), okFn); err != nil {
		t.Fatalf("expected success after slot freed, got %v", err)
	}
}

func TestUnit_Execute_ContextCanceled(t *testing.T) {
	cb := New(Config{
		MaxParallel: 10,
		Threshold:   10,
		Timeout:     time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // отменяем до вызова

	err := cb.Execute(ctx, okFn)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestUnit_Execute_ContextDeadlineExceeded(t *testing.T) {
	cb := New(Config{
		MaxParallel: 10,
		Threshold:   10,
		Timeout:     time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond) // даём таймауту истечь

	err := cb.Execute(ctx, okFn)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestUnit_Execute_ZeroConfigValues(t *testing.T) {
	// Все нулевые значения должны подняться до минимальных
	cb := New(Config{})

	// Проверяем, что порог стал 1
	ctx := context.Background()
	if err := cb.Execute(ctx, failFn("err")); err == nil {
		t.Fatal("expected error")
	}
	// Сразу после одной ошибки – ErrCircuitOpen
	if err := cb.Execute(ctx, okFn); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen after one error with default threshold, got %v", err)
	}
}

// Вспомогательные функции

func okFn(ctx context.Context) error {
	return nil
}

func failFn(msg string) func(context.Context) error {
	return func(ctx context.Context) error {
		return errors.New(msg)
	}
}
