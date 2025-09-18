/*
 *  Copyright (c) 2024-2025 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.osspkg.com/casecheck"
)

func TestUnit_Limiter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	lim := NewLimiter(ctx, 5, 250*time.Millisecond)
	var counter int64

	go func() {
		for i := 0; i < 1000; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				if !lim.Tap(context.Background()) {
					return
				}
				atomic.AddInt64(&counter, 1)
			}
		}
	}()

	<-ctx.Done()

	count := atomic.LoadInt64(&counter)
	casecheck.True(t, count >= 19 && count <= 21, "want 19-21, got %d", count)
}
