// Copyright 2026 Elementum Ltd. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package discovery

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewParallelExecutor(t *testing.T) {
	t.Run("nil config uses default", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		if executor.GetMaxConcurrency() != DefaultMaxConcurrency {
			t.Errorf("expected max concurrency %d, got %d", DefaultMaxConcurrency, executor.GetMaxConcurrency())
		}
	})

	t.Run("zero concurrency uses default", func(t *testing.T) {
		executor := NewParallelExecutor(&ParallelConfig{MaxConcurrency: 0})
		if executor.GetMaxConcurrency() != DefaultMaxConcurrency {
			t.Errorf("expected max concurrency %d, got %d", DefaultMaxConcurrency, executor.GetMaxConcurrency())
		}
	})

	t.Run("custom concurrency is respected", func(t *testing.T) {
		executor := NewParallelExecutor(&ParallelConfig{MaxConcurrency: 5})
		if executor.GetMaxConcurrency() != 5 {
			t.Errorf("expected max concurrency 5, got %d", executor.GetMaxConcurrency())
		}
	})
}

func TestRunParallel(t *testing.T) {
	t.Run("empty tasks returns nil", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		err := executor.RunParallel(context.Background(), nil)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("single task runs directly", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		called := false
		err := executor.RunParallel(context.Background(), []func() error{
			func() error { called = true; return nil },
		})
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if !called {
			t.Error("expected task to be called")
		}
	})

	t.Run("all tasks run", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		var count int64
		tasks := make([]func() error, 10)
		for i := range tasks {
			tasks[i] = func() error {
				atomic.AddInt64(&count, 1)
				return nil
			}
		}
		err := executor.RunParallel(context.Background(), tasks)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if count != 10 {
			t.Errorf("expected 10 tasks to run, got %d", count)
		}
	})

	t.Run("returns first error", func(t *testing.T) {
		executor := NewParallelExecutor(&ParallelConfig{MaxConcurrency: 2})
		expectedErr := errors.New("test error")
		tasks := []func() error{
			func() error { time.Sleep(10 * time.Millisecond); return nil },
			func() error { return expectedErr },
			func() error { time.Sleep(10 * time.Millisecond); return nil },
		}
		err := executor.RunParallel(context.Background(), tasks)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("respects concurrency limit", func(t *testing.T) {
		executor := NewParallelExecutor(&ParallelConfig{MaxConcurrency: 2})
		var concurrent, maxConcurrent int64

		tasks := make([]func() error, 10)
		for i := range tasks {
			tasks[i] = func() error {
				c := atomic.AddInt64(&concurrent, 1)
				for {
					m := atomic.LoadInt64(&maxConcurrent)
					if c > m {
						if atomic.CompareAndSwapInt64(&maxConcurrent, m, c) {
							break
						}
					} else {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt64(&concurrent, -1)
				return nil
			}
		}

		err := executor.RunParallel(context.Background(), tasks)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if maxConcurrent > 2 {
			t.Errorf("max concurrent tasks exceeded limit: got %d, want <= 2", maxConcurrent)
		}
	})
}

func TestRunParallelCollectErrors(t *testing.T) {
	t.Run("collects all errors", func(t *testing.T) {
		executor := NewParallelExecutor(&ParallelConfig{MaxConcurrency: 3})
		tasks := []func() error{
			func() error { return errors.New("error 1") },
			func() error { return nil },
			func() error { return errors.New("error 2") },
		}
		errs := executor.RunParallelCollectErrors(context.Background(), tasks)
		if len(errs) != 2 {
			t.Errorf("expected 2 errors, got %d", len(errs))
		}
	})

	t.Run("returns empty for all success", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		tasks := []func() error{
			func() error { return nil },
			func() error { return nil },
		}
		errs := executor.RunParallelCollectErrors(context.Background(), tasks)
		if len(errs) != 0 {
			t.Errorf("expected 0 errors, got %d", len(errs))
		}
	})
}

func TestMapParallel(t *testing.T) {
	t.Run("maps all items", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		items := []int{1, 2, 3, 4, 5}
		results, err := MapParallel(context.Background(), executor, items, func(i int) (int, error) {
			return i * 2, nil
		})
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		expected := []int{2, 4, 6, 8, 10}
		for i, r := range results {
			if r != expected[i] {
				t.Errorf("at index %d: expected %d, got %d", i, expected[i], r)
			}
		}
	})

	t.Run("preserves order", func(t *testing.T) {
		executor := NewParallelExecutor(&ParallelConfig{MaxConcurrency: 2})
		items := []int{1, 2, 3, 4, 5}
		results, err := MapParallel(context.Background(), executor, items, func(i int) (int, error) {
			// Random sleep to mix up execution order
			time.Sleep(time.Duration(i%3) * time.Millisecond)
			return i, nil
		})
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		for i, r := range results {
			if r != items[i] {
				t.Errorf("at index %d: expected %d, got %d", i, items[i], r)
			}
		}
	})

	t.Run("returns first error", func(t *testing.T) {
		executor := NewParallelExecutor(nil)
		items := []int{1, 2, 3}
		expectedErr := errors.New("error on 2")
		_, err := MapParallel(context.Background(), executor, items, func(i int) (int, error) {
			if i == 2 {
				return 0, expectedErr
			}
			return i, nil
		})
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
