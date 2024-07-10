/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import s "sync"

type (
	Group interface {
		Wait()
		Background(call func())
		Run(call func())
	}

	_group struct {
		wg s.WaitGroup
	}
)

func NewGroup() Group {
	return &_group{}
}

func (v *_group) Wait() {
	v.wg.Wait()
}

func (v *_group) Background(call func()) {
	v.wg.Add(1)
	go func() {
		call()
		v.wg.Done()
	}()
}

func (v *_group) Run(call func()) {
	v.wg.Add(1)
	call()
	v.wg.Done()
}
