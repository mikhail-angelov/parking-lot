package generator

import (
	"parking/internal/analyzer"
	"testing"
)

func TestBulkReportsProgress(t *testing.T) {
	config := Defaults(analyzer.Easy)
	config.Seed = 7
	var updates []Progress

	levels, err := Bulk(BulkConfig{
		Generator: config,
		Count:     1,
		Workers:   1,
		Progress: func(progress Progress) {
			updates = append(updates, progress)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 1 || len(updates) == 0 {
		t.Fatalf("levels=%d updates=%d", len(levels), len(updates))
	}
	last := updates[len(updates)-1]
	if last.Accepted != 1 || last.Target != 1 || last.Processed != 1 {
		t.Fatalf("last progress=%+v", last)
	}
}

func TestBulkReportsHardRetryExhaustion(t *testing.T) {
	config := Defaults(analyzer.Hard)
	config.MinVehicles = 0
	var last Progress

	_, err := Bulk(BulkConfig{
		Generator: config,
		Count:     1,
		Workers:   1,
		Progress: func(progress Progress) {
			last = progress
		},
	})
	if err == nil {
		t.Fatal("expected invalid generator config to exhaust retries")
	}
	if last.MaxJobs != 80 || last.Processed != 80 || last.Failures != 80 || last.Accepted != 0 {
		t.Fatalf("last progress=%+v", last)
	}
}

func TestBulkExcludesCanonicalHashesFromEarlierRuns(t *testing.T) {
	config := Defaults(analyzer.Easy)
	config.Seed = 7
	first, err := Bulk(BulkConfig{Generator: config, Count: 1, Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	excluded := map[string]bool{CanonicalHash(first[0].Level): true}
	duplicates := 0

	second, err := Bulk(BulkConfig{
		Generator:      config,
		Count:          1,
		Workers:        1,
		ExcludedHashes: excluded,
		Progress: func(progress Progress) {
			duplicates = progress.Duplicates
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if CanonicalHash(second[0].Level) == CanonicalHash(first[0].Level) {
		t.Fatal("excluded level was generated again")
	}
	if duplicates == 0 {
		t.Fatal("excluded level must be reported as a duplicate")
	}
}
