/*
 *  Copyright (c) 2024-2025 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import (
	"sync"
	"testing"
	"time"

	"go.osspkg.com/casecheck"
)

func TestUnit_Funnel(t *testing.T) {
	var wg sync.WaitGroup

	result := NewSlice[string](5)

	startCh := make(chan struct{})

	ml := NewFunnel[string]()

	wg.Add(3)

	go func() {
		defer wg.Done()

		close(startCh)

		ml.Valve("a", func() {
			result.Append("a1")
		})

		time.Sleep(100 * time.Millisecond)
	}()

	<-startCh

	go func() {
		defer wg.Done()

		ml.Valve("a", func() {
			result.Append("a2")
		})

		time.Sleep(100 * time.Millisecond)
	}()

	go func() {
		defer wg.Done()

		ml.Valve("b", func() {
			result.Append("b1")
		})

		time.Sleep(100 * time.Millisecond)
	}()

	wg.Wait()

	casecheck.Equal(t, []string{"a1", "b1", "a2"}, result.Extract())
}

func BenchmarkFunnel(b *testing.B) {
	f := NewFunnel[string]()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			f.Valve("", func() {

			})
		}
	})
}
