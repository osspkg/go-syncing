/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import (
	"testing"
)

func TestMap(t *testing.T) {
	t.Run("NewMap", func(t *testing.T) {
		m := NewMap[string, int](10)
		if m == nil {
			t.Fatal("NewMap returned nil")
		}
		if m.Size() != 0 {
			t.Errorf("Expected size 0, got %d", m.Size())
		}
	})

	t.Run("Set and Get", func(t *testing.T) {
		m := NewMap[string, int](0)
		m.Set("a", 1)
		m.Set("b", 2)

		if val, ok := m.Get("a"); !ok || val != 1 {
			t.Errorf("Get a = %d, %v; want 1, true", val, ok)
		}
		if val, ok := m.Get("b"); !ok || val != 2 {
			t.Errorf("Get b = %d, %v; want 2, true", val, ok)
		}
		if _, ok := m.Get("c"); ok {
			t.Error("Get c should be false")
		}
	})

	t.Run("Size", func(t *testing.T) {
		m := NewMap[string, int](0)
		if s := m.Size(); s != 0 {
			t.Errorf("Size = %d, want 0", s)
		}
		m.Set("x", 1)
		m.Set("y", 2)
		if s := m.Size(); s != 2 {
			t.Errorf("Size = %d, want 2", s)
		}
		m.Del("x")
		if s := m.Size(); s != 1 {
			t.Errorf("Size = %d, want 1", s)
		}
	})

	t.Run("Del", func(t *testing.T) {
		m := NewMap[string, int](0)
		m.Set("k", 10)
		m.Del("k")
		if _, ok := m.Get("k"); ok {
			t.Error("Key still exists after Del")
		}
		m.Del("nonexistent")
	})

	t.Run("Extract", func(t *testing.T) {
		m := NewMap[string, int](0)
		m.Set("one", 1)
		m.Set("two", 2)

		val, ok := m.Extract("one")
		if !ok || val != 1 {
			t.Errorf("Extract one = %d, %v; want 1, true", val, ok)
		}
		if _, ok := m.Get("one"); ok {
			t.Error("Key 'one' still present after Extract")
		}
		if m.Size() != 1 {
			t.Errorf("Size after Extract = %d, want 1", m.Size())
		}

		val, ok = m.Extract("three")
		if ok {
			t.Errorf("Extract non-existent got %d, %v; want false", val, ok)
		}
	})

	t.Run("Keys", func(t *testing.T) {
		m := NewMap[string, int](0)
		m.Set("a", 1)
		m.Set("b", 2)
		m.Set("c", 3)

		keys := m.Keys()
		if len(keys) != 3 {
			t.Fatalf("Keys length = %d, want 3", len(keys))
		}
		got := make(map[string]bool)
		for _, k := range keys {
			got[k] = true
		}
		for _, exp := range []string{"a", "b", "c"} {
			if !got[exp] {
				t.Errorf("Key %q missing from Keys", exp)
			}
		}
	})

	t.Run("Reset", func(t *testing.T) {
		m := NewMap[string, int](0)
		m.Set("a", 1)
		m.Set("b", 2)
		m.Reset()
		if m.Size() != 0 {
			t.Errorf("After Reset size = %d, want 0", m.Size())
		}
		if _, ok := m.Get("a"); ok {
			t.Error("Key 'a' still exists after Reset")
		}
	})

	t.Run("Yield", func(t *testing.T) {
		m := NewMap[int, string](0)
		m.Set(1, "one")
		m.Set(2, "two")
		m.Set(3, "three")

		seen := make(map[int]string)
		for k, v := range m.Yield() {
			seen[k] = v
		}
		if len(seen) != 3 {
			t.Errorf("Yield produced %d pairs, want 3", len(seen))
		}
		if seen[1] != "one" || seen[2] != "two" || seen[3] != "three" {
			t.Errorf("Yield produced unexpected pairs: %v", seen)
		}

		count := 0
		for range m.Yield() {
			count++
			if count == 1 {
				break
			}
		}
		if count != 1 {
			t.Errorf("Yield break: got %d iterations, want 1", count)
		}
	})
}

func TestSlice(t *testing.T) {
	t.Run("NewSlice", func(t *testing.T) {
		s := NewSlice[int](5)
		if s == nil {
			t.Fatal("NewSlice returned nil")
		}
		if s.Size() != 0 {
			t.Errorf("Expected size 0, got %d", s.Size())
		}
	})

	t.Run("Append and Size", func(t *testing.T) {
		s := NewSlice[int](0)
		s.Append(1, 2, 3)
		if s.Size() != 3 {
			t.Errorf("After Append size = %d, want 3", s.Size())
		}
		s.Append(4)
		if s.Size() != 4 {
			t.Errorf("After second Append size = %d, want 4", s.Size())
		}
	})

	t.Run("All", func(t *testing.T) {
		s := NewSlice[int](0)
		s.Append(10, 20, 30)
		all := s.All()
		expected := []int{10, 20, 30}
		if len(all) != len(expected) {
			t.Fatalf("All length = %d, want %d", len(all), len(expected))
		}
		for i, v := range all {
			if v != expected[i] {
				t.Errorf("All[%d] = %d, want %d", i, v, expected[i])
			}
		}
		all[0] = 999
		if val, _ := s.Index(0); val != 10 {
			t.Errorf("Original slice changed via All returned slice: got %d, want 10", val)
		}
	})

	t.Run("Reset", func(t *testing.T) {
		s := NewSlice[int](0)
		s.Append(1, 2, 3)
		s.Reset()
		if s.Size() != 0 {
			t.Errorf("After Reset size = %d, want 0", s.Size())
		}
		s.Append(5)
		if s.Size() != 1 || s.All()[0] != 5 {
			t.Error("Slice not usable after Reset")
		}
	})

	t.Run("Extract", func(t *testing.T) {
		s := NewSlice[int](0)
		s.Append(1, 2, 3)
		extracted := s.Extract()
		expected := []int{1, 2, 3}
		if len(extracted) != len(expected) {
			t.Fatalf("Extract length = %d, want %d", len(extracted), len(expected))
		}
		for i, v := range extracted {
			if v != expected[i] {
				t.Errorf("Extract[%d] = %d, want %d", i, v, expected[i])
			}
		}
		if s.Size() != 0 {
			t.Errorf("After Extract size = %d, want 0", s.Size())
		}
		extracted2 := s.Extract()
		if len(extracted2) != 0 {
			t.Errorf("Second Extract returned non-empty slice: %v", extracted2)
		}
	})

	t.Run("Index", func(t *testing.T) {
		s := NewSlice[int](0)
		s.Append(100, 200)

		val, ok := s.Index(0)
		if !ok || val != 100 {
			t.Errorf("Index 0: got %d, %v; want 100, true", val, ok)
		}
		val, ok = s.Index(1)
		if !ok || val != 200 {
			t.Errorf("Index 1: got %d, %v; want 200, true", val, ok)
		}
		val, ok = s.Index(2)
		if ok {
			t.Errorf("Index 2: got %d, %v; want false", val, ok)
		}
	})

	t.Run("Set", func(t *testing.T) {
		s := NewSlice[int](0)
		s.Append(1, 2, 3)

		s.Set(1, 99)
		if val, _ := s.Index(1); val != 99 {
			t.Errorf("After Set(1,99): got %d, want 99", val)
		}

		s.Set(5, 42)
		if s.Size() != 6 {
			t.Errorf("After Set(5,42) size = %d, want 6", s.Size())
		}
		if val, _ := s.Index(5); val != 42 {
			t.Errorf("Index 5 = %d, want 42", val)
		}
		if val, _ := s.Index(3); val != 0 {
			t.Errorf("Index 3 = %d, want 0", val)
		}
		if val, _ := s.Index(4); val != 0 {
			t.Errorf("Index 4 = %d, want 0", val)
		}

		s.Set(-1, 999)
		if s.Size() != 6 {
			t.Errorf("Set with negative index changed size: %d", s.Size())
		}
	})

	t.Run("Splice", func(t *testing.T) {
		eq := func(a, b []int) bool {
			if len(a) != len(b) {
				return false
			}
			for i := range a {
				if a[i] != b[i] {
					return false
				}
			}
			return true
		}

		tests := []struct {
			name        string
			initial     []int
			start       int
			deleteCount int
			elements    []int
			expected    []int
		}{
			{"delete middle", []int{1, 2, 3, 4, 5}, 1, 2, nil, []int{1, 4, 5}},
			{"insert at start", []int{2, 3, 4}, 0, 0, []int{0, 1}, []int{0, 1, 2, 3, 4}},
			{"insert at end", []int{1, 2}, 2, 0, []int{3, 4}, []int{1, 2, 3, 4}},
			{"replace", []int{1, 2, 3}, 1, 1, []int{99, 100}, []int{1, 99, 100, 3}},
			{"delete more than length", []int{1, 2, 3}, 1, 10, nil, []int{1}},
			{"negative start", []int{1, 2, 3}, -1, 1, []int{99}, []int{99, 2, 3}},
			{"start beyond length", []int{1, 2, 3}, 5, 1, []int{99}, []int{1, 2, 3, 99}},
			{"negative deleteCount", []int{1, 2, 3}, 1, -2, nil, []int{1, 2, 3}},
			{"zero deleteCount and insert", []int{1, 2}, 1, 0, []int{99}, []int{1, 99, 2}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := NewSlice[int](0)
				s.Append(tt.initial...)
				s.Splice(tt.start, tt.deleteCount, tt.elements...)
				got := s.All()
				if !eq(got, tt.expected) {
					t.Errorf("Splice result = %v, want %v", got, tt.expected)
				}
			})
		}
	})

	t.Run("Yield", func(t *testing.T) {
		s := NewSlice[string](0)
		s.Append("a", "b", "c")

		seen := []string{}
		for v := range s.Yield() {
			seen = append(seen, v)
		}
		expected := []string{"a", "b", "c"}
		if len(seen) != len(expected) {
			t.Fatalf("Yield produced %d items, want %d", len(seen), len(expected))
		}
		for i := range seen {
			if seen[i] != expected[i] {
				t.Errorf("Yield[%d] = %q, want %q", i, seen[i], expected[i])
			}
		}

		count := 0
		for range s.Yield() {
			count++
			if count == 1 {
				break
			}
		}
		if count != 1 {
			t.Errorf("Yield break: got %d iterations, want 1", count)
		}
	})
}
