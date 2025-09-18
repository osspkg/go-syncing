/*
 *  Copyright (c) 2024-2025 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import (
	"sync"
	"sync/atomic"
)

type item struct {
	mux   sync.RWMutex
	count int32
}

func (l *item) Inc() {
	atomic.AddInt32(&l.count, 1)
}

func (l *item) Dec() {
	atomic.AddInt32(&l.count, -1)
}

func (l *item) Count() int32 {
	return atomic.LoadInt32(&l.count)
}

func (l *item) Lock() {
	l.mux.Lock()
}

func (l *item) Unlock() {
	l.mux.Unlock()
}

// ---------------------------------------------------------------------------------------------------

type (
	funnel[T comparable] struct {
		global sync.Mutex
		locks  map[T]*item
	}

	Funnel[T comparable] interface {
		Valve(name T, call func())
	}
)

func NewFunnel[T comparable]() Funnel[T] {
	return &funnel[T]{
		locks: make(map[T]*item, 10),
	}
}

func (l *funnel[T]) Valve(name T, call func()) {
	l.global.Lock()
	nl, exist := l.locks[name]
	if !exist {
		nl = &item{}
		l.locks[name] = nl
	}
	nl.Inc()
	l.global.Unlock()

	nl.Lock()

	call()

	nl.Dec()
	nl.Unlock()

	if nl.Count() <= 0 {
		l.global.Lock()
		if nl.Count() <= 0 {
			delete(l.locks, name)
		}
		l.global.Unlock()
	}
}
