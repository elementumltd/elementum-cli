// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"context"
	"sort"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// DefaultMaxConcurrency is the default number of concurrent API calls
const DefaultMaxConcurrency int64 = 20

// parallelResult captures the result of a parallel task with its name for error reporting
type parallelResult struct {
	name string
	err  error
}

// ParallelResults collects results from parallel tasks in a thread-safe manner
type ParallelResults struct {
	mu      sync.Mutex
	results []parallelResult
}

// Add records a task result
func (pr *ParallelResults) Add(name string, err error) {
	if err == nil {
		return
	}
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.results = append(pr.results, parallelResult{name: name, err: err})
}

// Errors returns all collected errors
func (pr *ParallelResults) Errors() []parallelResult {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return pr.results
}

// HasErrors returns true if any errors were collected
func (pr *ParallelResults) HasErrors() bool {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return len(pr.results) > 0
}

// ParallelConfig configures parallel execution behavior
type ParallelConfig struct {
	// MaxConcurrency limits the number of concurrent operations.
	// Default: 10 if not specified or <= 0
	MaxConcurrency int64
}

// ParallelExecutor manages parallel execution of tasks with bounded concurrency
type ParallelExecutor struct {
	sem    *semaphore.Weighted
	config *ParallelConfig
}

// NewParallelExecutor creates a new ParallelExecutor with the given config.
// If config is nil or MaxConcurrency <= 0, DefaultMaxConcurrency is used.
func NewParallelExecutor(config *ParallelConfig) *ParallelExecutor {
	maxConcurrency := DefaultMaxConcurrency
	if config != nil && config.MaxConcurrency > 0 {
		maxConcurrency = config.MaxConcurrency
	}

	return &ParallelExecutor{
		sem: semaphore.NewWeighted(maxConcurrency),
		config: &ParallelConfig{
			MaxConcurrency: maxConcurrency,
		},
	}
}

// RunParallel executes all tasks in parallel with bounded concurrency.
// It returns the first error encountered, or nil if all tasks succeed.
// All tasks are started even if one fails, but RunParallel waits for
// all to complete before returning.
func (p *ParallelExecutor) RunParallel(ctx context.Context, tasks []func() error) error {
	if len(tasks) == 0 {
		return nil
	}

	// For single task, just run it directly
	if len(tasks) == 1 {
		return tasks[0]()
	}

	g, ctx := errgroup.WithContext(ctx)

	for _, task := range tasks {
		task := task // capture loop variable
		g.Go(func() error {
			// Acquire semaphore slot
			if err := p.sem.Acquire(ctx, 1); err != nil {
				return err
			}
			defer p.sem.Release(1)

			return task()
		})
	}

	return g.Wait()
}

// RunParallelCollectErrors executes all tasks in parallel with bounded concurrency
// and collects all errors instead of stopping at the first error.
// Returns a slice of errors (may be empty if all succeeded).
func (p *ParallelExecutor) RunParallelCollectErrors(ctx context.Context, tasks []func() error) []error {
	if len(tasks) == 0 {
		return nil
	}

	var mu sync.Mutex
	var errors []error

	var wg sync.WaitGroup
	for _, task := range tasks {
		task := task // capture loop variable
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Acquire semaphore slot
			if err := p.sem.Acquire(ctx, 1); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}
			defer p.sem.Release(1)

			if err := task(); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return errors
}

// GetMaxConcurrency returns the configured maximum concurrency
func (p *ParallelExecutor) GetMaxConcurrency() int64 {
	return p.config.MaxConcurrency
}

// MapParallel applies a function to each item in a slice in parallel with bounded concurrency.
// Results are stored in the order of the input slice (not execution order).
// Returns the first error encountered, or nil if all operations succeed.
func MapParallel[T any, R any](ctx context.Context, p *ParallelExecutor, items []T, fn func(T) (R, error)) ([]R, error) {
	if len(items) == 0 {
		return nil, nil
	}

	results := make([]R, len(items))
	g, ctx := errgroup.WithContext(ctx)

	for i, item := range items {
		i, item := i, item // capture loop variables
		g.Go(func() error {
			if err := p.sem.Acquire(ctx, 1); err != nil {
				return err
			}
			defer p.sem.Release(1)

			result, err := fn(item)
			if err != nil {
				return err
			}
			results[i] = result
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// MapParallelCollectErrors applies a function to each item in parallel,
// collecting all errors instead of stopping at the first.
// Results are stored in order; failed items will have zero values.
func MapParallelCollectErrors[T any, R any](ctx context.Context, p *ParallelExecutor, items []T, fn func(T) (R, error)) ([]R, []error) {
	if len(items) == 0 {
		return nil, nil
	}

	results := make([]R, len(items))
	var mu sync.Mutex
	var errors []error

	var wg sync.WaitGroup
	for i, item := range items {
		i, item := i, item
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := p.sem.Acquire(ctx, 1); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}
			defer p.sem.Release(1)

			result, err := fn(item)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}
			results[i] = result
		}()
	}

	wg.Wait()
	return results, errors
}

// SortByID is a helper interface for types that can be sorted by ID
type SortByID interface {
	GetID() string
}

// SortSliceByID sorts a slice of items by their ID for deterministic output
func SortSliceByID[T SortByID](items []T) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetID() < items[j].GetID()
	})
}

// SortMapValues extracts map values and sorts them by ID
func SortMapValues[K comparable, V SortByID](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	SortSliceByID(values)
	return values
}
