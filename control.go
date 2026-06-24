/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import "context"

type (
	Control interface {
		Acquire(ctx context.Context) bool
		Release()
	}
	sem struct {
		c chan struct{}
	}
)

func NewControl(count uint64) Control {
	count = max(count, 1)
	return &sem{
		c: make(chan struct{}, count),
	}
}

func (s *sem) Acquire(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}

	select {
	case s.c <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *sem) Release() {
	select {
	case <-s.c:
	default:
	}
}
