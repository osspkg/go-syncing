/*
 *  Copyright (c) 2024-2025 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import (
	"context"
	"fmt"
	"sync"
)

type (
	Group interface {
		Cancel()
		Wait()
		Background(name string, call func(ctx context.Context))
		Run(name string, call func(ctx context.Context))
		OnPanic(call func(err error))
	}

	_group struct {
		wg        sync.WaitGroup
		mux       sync.RWMutex
		globalCtx context.Context
		cancelCtx context.CancelFunc

		onPanic func(err error)
	}
)

func NewGroup(ctx context.Context) Group {
	ctx, cancel := context.WithCancel(ctx)
	return &_group{
		globalCtx: ctx,
		cancelCtx: cancel,
	}
}

func (v *_group) OnPanic(call func(err error)) {
	v.mux.Lock()
	defer v.mux.Unlock()

	v.onPanic = call
}

func (v *_group) Wait() {
	v.wg.Wait()
}

func (v *_group) Cancel() {
	v.cancelCtx()
}

func (v *_group) Background(name string, call func(ctx context.Context)) {
	v.wg.Add(1)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				v.mux.RLock()
				if v.onPanic != nil {
					v.onPanic(fmt.Errorf("%s: %v", name, err))
				}
				v.mux.RUnlock()
			}
			v.wg.Done()
		}()

		call(v.globalCtx)
	}()
}

func (v *_group) Run(name string, call func(ctx context.Context)) {
	v.wg.Add(1)
	defer func() {
		if err := recover(); err != nil {
			v.mux.RLock()
			if v.onPanic != nil {
				v.onPanic(fmt.Errorf("%s: %v", name, err))
			}
			v.mux.RUnlock()
		}
		v.wg.Done()
	}()

	call(v.globalCtx)
}
