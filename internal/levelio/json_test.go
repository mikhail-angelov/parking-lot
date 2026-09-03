package levelio

import (
	"os"
	"path/filepath"
	"testing"

	"parking/internal/analyzer"
	"parking/internal/puzzle"
)

func TestWriteDatasetWritesOnlyJSON(t *testing.T) {
	level := puzzle.Level{
		Width: 6, Height: 6, Target: 0,
		Vehicles: []puzzle.Vehicle{
			{ID: 0, Orientation: puzzle.Horizontal, Length: 2, Fixed: 2},
			{ID: 1, Orientation: puzzle.Vertical, Length: 2, Fixed: 2},
		},
		Initial: puzzle.State{Positions: []uint8{0, 2}},
	}
	doc := NewGeneratedFile("easy-1", level, analyzer.Analysis{Solvable: true}, nil, 7)
	path := filepath.Join(t.TempDir(), "levels.json")

	if err := WriteDataset(path, []File{doc}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "levels.js")); !os.IsNotExist(err) {
		t.Fatalf("unexpected JavaScript artifact: %v", err)
	}
}

func TestPackFilesAreJSONOnly(t *testing.T) {
	dir := t.TempDir()
	if err := WritePack(dir, "pack-001", nil); err != nil {
		t.Fatal(err)
	}
	if err := WriteManifest(dir, []Pack{{ID: "pack-001", File: "pack-001.json"}}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"pack-001.json", "manifest.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	for _, name := range []string{"pack-001.js", "manifest.js"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("unexpected JavaScript artifact %s: %v", name, err)
		}
	}
}

func TestFromFileRejectsMissingVersion(t *testing.T) {
	file := File{Target: 0}
	file.Board.Width = 6
	file.Board.Height = 6
	file.Board.Exit = "right"
	file.Vehicles = []fileVehicle{
		{ID: 0, Orientation: "horizontal", Length: 2, Fixed: 2, Position: 0},
		{ID: 1, Orientation: "vertical", Length: 2, Fixed: 2, Position: 2},
	}

	if _, err := FromFile(file); err == nil {
		t.Fatal("expected version 0 to be rejected")
	}
}
