/*
 *  Copyright (c) 2024-2025 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import (
	"context"
	"time"
)

type (
	limiter struct {
		count int
		pause time.Duration
		ch    chan struct{}
		ctx   context.Context
	}

	Limiter interface {
		Tap(ctx context.Context) bool
	}
)

func NewLimiter(ctx context.Context, count int, interval time.Duration) Limiter {
	if count <= 0 {
		count = 1
	}

	lim := &limiter{
		count: count,
		pause: interval / time.Duration(count),
		ch:    make(chan struct{}, count),
		ctx:   ctx,
	}

	go lim.refill()
	go func() {
		tik := time.NewTicker(interval)
		defer tik.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-tik.C:
				lim.refill()
			}
		}
	}()

	return lim
}

func (l *limiter) Tap(ctx context.Context) bool {
	select {
	case <-l.ch:
		return true
	case <-l.ctx.Done():
		return false
	case <-ctx.Done():
		return false
	}
}

func (l *limiter) refill() {
	for i := 0; i < l.count; i++ {
		select {
		case <-l.ctx.Done():
			return
		case l.ch <- struct{}{}:
			time.Sleep(l.pause)
		default:
		}
	}
}
