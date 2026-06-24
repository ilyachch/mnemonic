package parallel

import "sync"

// MapIndexed applies fn to each item using a bounded worker pool.
//
// Results and errors are returned in the same order as the input slice.
func MapIndexed[T any, R any](items []T, limit int, fn func(int, T) (R, error)) ([]R, []error) {
	if len(items) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 1
	}
	if limit > len(items) {
		limit = len(items)
	}

	results := make([]R, len(items))
	errs := make([]error, len(items))

	type job struct {
		index int
		item  T
	}

	jobs := make(chan job)
	var wg sync.WaitGroup
	wg.Add(limit)

	for i := 0; i < limit; i++ {
		go func() {
			defer wg.Done()
			for job := range jobs {
				result, err := fn(job.index, job.item)
				if err != nil {
					errs[job.index] = err
					continue
				}
				results[job.index] = result
			}
		}()
	}

	for i, item := range items {
		jobs <- job{index: i, item: item}
	}
	close(jobs)
	wg.Wait()

	return results, errs
}

// FirstError returns the first non-nil error in errs.
func FirstError(errs []error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
