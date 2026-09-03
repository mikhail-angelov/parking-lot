package main

import (
	"parking/internal/generator"
	"strings"
	"testing"
	"time"
)

func TestFormatGenerationProgressIncludesETA(t *testing.T) {
	got := formatGenerationProgress("pack-003", generator.Progress{
		Accepted:  4,
		Target:    10,
		Processed: 5,
		MaxJobs:   30,
		Failures:  1,
	}, 20*time.Second)

	for _, want := range []string{"pack-003", "4/10", "5/30", "ошибки 1", "ETA 30s"} {
		if !strings.Contains(got, want) {
			t.Fatalf("progress %q does not contain %q", got, want)
		}
	}
}
