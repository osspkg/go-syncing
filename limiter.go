/*
 *  Copyright (c) 2024-2025 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import (
	"context"
	"fmt"
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
		Tap(ctx context.Context) error
	}
)

func NewLimiter(ctx context.Context, count int, interval time.Duration) Limiter {
	if count <= 0 {
		count = 1
	}

	rl := &limiter{
		count: count,
		pause: interval / time.Duration(count),
		ch:    make(chan struct{}, count),
		ctx:   ctx,
	}

	go rl.refill()
	go func() {
		tik := time.NewTicker(interval)
		defer tik.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-tik.C:
				rl.refill()
			}
		}
	}()

	return rl
}

func (l *limiter) Tap(ctx context.Context) error {
	select {
	case <-l.ch:
		return nil
	case <-l.ctx.Done():
		return fmt.Errorf("context canceled")
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
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
