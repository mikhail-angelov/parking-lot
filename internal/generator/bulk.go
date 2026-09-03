package generator

import (
	"fmt"
	"parking/internal/analyzer"
	"sync"
)

type BulkConfig struct {
	Generator      Config
	Count          int
	Workers        int
	ExcludedHashes map[string]bool
	Progress       func(Progress)
}

type Progress struct {
	Accepted   int
	Target     int
	Processed  int
	MaxJobs    int
	Failures   int
	Duplicates int
}

type bulkResult struct {
	level *GeneratedLevel
	err   error
}

func Bulk(c BulkConfig) ([]GeneratedLevel, error) {
	if c.Count < 0 || c.Workers < 1 {
		return nil, fmt.Errorf("count must be non-negative and workers must be positive")
	}
	if c.Count == 0 {
		return []GeneratedLevel{}, nil
	}
	out := make([]GeneratedLevel, 0, c.Count)
	seen := map[string]bool{}
	failures, duplicates := 0, 0
	var firstErr error
	maxJobs := c.Count * retryMultiplier(c.Generator.Tier)
	processed := 0
	for nextJob := 0; len(out) < c.Count && nextJob < maxJobs; {
		batchSize := c.Count - len(out)
		if batchSize > maxJobs-nextJob {
			batchSize = maxJobs - nextJob
		}
		results := generateBatch(c, nextJob, batchSize)
		nextJob += batchSize
		for r := range results {
			processed++
			if r.err != nil {
				failures++
				if firstErr == nil {
					firstErr = r.err
				}
			} else if key := CanonicalHash(r.level.Level); seen[key] || c.ExcludedHashes[key] {
				duplicates++
			} else {
				seen[key] = true
				out = append(out, *r.level)
			}
			if c.Progress != nil {
				c.Progress(Progress{
					Accepted:   len(out),
					Target:     c.Count,
					Processed:  processed,
					MaxJobs:    maxJobs,
					Failures:   failures,
					Duplicates: duplicates,
				})
			}
		}
	}
	if len(out) != c.Count {
		if firstErr != nil {
			return nil, fmt.Errorf("generated %d of %d levels (%d failed, %d duplicates): %w", len(out), c.Count, failures, duplicates, firstErr)
		}
		return nil, fmt.Errorf("generated %d of %d levels (%d duplicates)", len(out), c.Count, duplicates)
	}
	return out, nil
}

func retryMultiplier(tier analyzer.DifficultyTier) int {
	switch tier {
	case analyzer.Hard, analyzer.Expert:
		return 80
	default:
		return 3
	}
}

func generateBatch(c BulkConfig, start, count int) <-chan bulkResult {
	workers := c.Workers
	if workers > count {
		workers = count
	}
	jobs := make(chan int)
	results := make(chan bulkResult, count)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				config := c.Generator
				config.Seed = c.Generator.Seed + int64(job)
				level, err := Generate(config)
				results <- bulkResult{level: level, err: err}
			}
		}()
	}
	go func() {
		for job := start; job < start+count; job++ {
			jobs <- job
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	return results
}
