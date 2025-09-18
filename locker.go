/*
 *  Copyright (c) 2024-2025 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import s "sync"

type (
	Lock interface {
		RLock(call func())
		Lock(call func())
	}
	_lock struct {
		mux s.RWMutex
	}
)

func NewLock() Lock {
	return &_lock{}
}

func (v *_lock) Lock(call func()) {
	v.mux.Lock()
	defer v.mux.Unlock()

	call()
}
func (v *_lock) RLock(call func()) {
	v.mux.RLock()
	defer v.mux.RUnlock()

	call()
}

//------------------------------------------------------------------

type (
	XLock[V any] interface {
		RLock(call func() (V, error)) (V, error)
		Lock(call func() (V, error)) (V, error)
	}
	_xlock[V any] struct {
		mux s.RWMutex
	}
)

func NewXLock[V any]() XLock[V] {
	return &_xlock[V]{}
}

func (v *_xlock[V]) Lock(call func() (V, error)) (V, error) {
	v.mux.Lock()
	defer v.mux.Unlock()

	return call()
}
func (v *_xlock[V]) RLock(call func() (V, error)) (V, error) {
	v.mux.RLock()
	defer v.mux.RUnlock()

	return call()
}
