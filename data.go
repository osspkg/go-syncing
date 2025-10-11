/*
 *  Copyright (c) 2024-2025 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import (
	"iter"
	"sync"
)

type Map[K comparable, V any] struct {
	data map[K]V
	mux  sync.RWMutex
}

func NewMap[K comparable, V any](size uint) *Map[K, V] {
	return &Map[K, V]{
		data: make(map[K]V, size),
		mux:  sync.RWMutex{},
	}
}

func (v *Map[K, V]) Size() int {
	v.mux.RLock()
	defer v.mux.RUnlock()

	return len(v.data)
}

func (v *Map[K, V]) Set(key K, val V) {
	v.mux.Lock()
	defer v.mux.Unlock()

	v.data[key] = val
}

func (v *Map[K, V]) Del(key K) {
	v.mux.Lock()
	defer v.mux.Unlock()

	delete(v.data, key)
}

func (v *Map[K, V]) Get(key K) (val V, ok bool) {
	v.mux.RLock()
	defer v.mux.RUnlock()

	val, ok = v.data[key]
	return
}

func (v *Map[K, V]) Extract(key K) (val V, ok bool) {
	v.mux.Lock()
	defer v.mux.Unlock()

	val, ok = v.data[key]
	if ok {
		delete(v.data, key)
	}
	return
}

func (v *Map[K, V]) Keys() []K {
	v.mux.RLock()
	defer v.mux.RUnlock()

	tmp := make([]K, 0, len(v.data))
	for key := range v.data {
		tmp = append(tmp, key)
	}
	return tmp
}

func (v *Map[K, V]) Reset() {
	v.mux.Lock()
	defer v.mux.Unlock()

	for key := range v.data {
		delete(v.data, key)
	}
}

func (v *Map[K, V]) Yield() iter.Seq2[K, V] {
	keys := v.Keys()
	return func(yield func(K, V) bool) {
		for _, key := range keys {
			if val, ok := v.Get(key); ok {
				if !yield(key, val) {
					return
				}
			}
		}
	}
}

//-----------------------------------------------------------------------------------------------

type Slice[V any] struct {
	data []V
	mux  sync.RWMutex
}

func NewSlice[V any](size uint) *Slice[V] {
	return &Slice[V]{
		data: make([]V, 0, size),
		mux:  sync.RWMutex{},
	}
}

func (v *Slice[V]) Size() int {
	v.mux.RLock()
	defer v.mux.RUnlock()

	return len(v.data)
}

func (v *Slice[V]) Reset() {
	v.mux.Lock()
	defer v.mux.Unlock()

	v.data = v.data[:0]
}

func (v *Slice[V]) Append(val ...V) {
	v.mux.Lock()
	defer v.mux.Unlock()

	v.data = append(v.data, val...)
}

func (v *Slice[V]) Extract() []V {
	v.mux.Lock()
	defer v.mux.Unlock()

	tmp := make([]V, len(v.data))
	copy(tmp[:], v.data[:])
	v.data = v.data[:0]
	return tmp
}

func (v *Slice[V]) Index(i int) (V, bool) {
	v.mux.RLock()
	defer v.mux.RUnlock()

	if i >= len(v.data) {
		var empty V
		return empty, false
	}

	return v.data[i], true
}

func (v *Slice[V]) Yield() iter.Seq[V] {
	length := len(v.data)
	return func(yield func(V) bool) {
		for i := 0; i < length; i++ {
			if val, ok := v.Index(i); ok {
				if !yield(val) {
					return
				}
			}
		}
	}
}
