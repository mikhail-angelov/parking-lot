package generator

import (
	"encoding/json"
	"os"
	"parking/internal/levelio"
	"path/filepath"
	"testing"
)

func TestGeneratedPacksAreCanonicallyUnique(t *testing.T) {
	paths, err := filepath.Glob("../../public/levels/pack-*.json")
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]string)
	count := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var files []levelio.File
		if err := json.Unmarshal(data, &files); err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			level, err := levelio.FromFile(file)
			if err != nil {
				t.Fatalf("%s: %v", file.ID, err)
			}
			hash := CanonicalHash(level)
			if previous, exists := seen[hash]; exists {
				t.Fatalf("canonical duplicate: %s and %s", previous, file.ID)
			}
			seen[hash] = file.ID
			count++
		}
	}
	if count != 600 {
		t.Fatalf("generated levels=%d, want 600", count)
	}
}
