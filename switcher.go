/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import "sync/atomic"

const (
	on  uint64 = 1
	off uint64 = 0
)

type (
	Switch interface {
		On() bool
		Off() bool
		IsOn() bool
		IsOff() bool
	}

	_switch struct {
		i uint64
	}
)

func NewSwitch() Switch {
	return &_switch{i: 0}
}

func (v *_switch) On() bool {
	return atomic.CompareAndSwapUint64(&v.i, off, on)
}

func (v *_switch) Off() bool {
	return atomic.CompareAndSwapUint64(&v.i, on, off)
}

func (v *_switch) IsOn() bool {
	return atomic.LoadUint64(&v.i) == on
}

func (v *_switch) IsOff() bool {
	return atomic.LoadUint64(&v.i) == off
}
