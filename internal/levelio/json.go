package levelio

import (
	"encoding/json"
	"fmt"
	"os"
	"parking/internal/analyzer"
	"parking/internal/puzzle"
	"path/filepath"
)

type fileVehicle struct {
	ID          uint8  `json:"id"`
	Orientation string `json:"orientation"`
	Length      uint8  `json:"length"`
	Fixed       uint8  `json:"fixed"`
	Position    uint8  `json:"position"`
}

// File is the versioned JSON representation of a level.
type File struct {
	Version int    `json:"version"`
	ID      string `json:"id,omitempty"`
	Board   struct {
		Width  uint8  `json:"width"`
		Height uint8  `json:"height"`
		Exit   string `json:"exit"`
	} `json:"board"`
	Target     uint8              `json:"target"`
	Vehicles   []fileVehicle      `json:"vehicles"`
	Analysis   *analyzer.Analysis `json:"analysis,omitempty"`
	Solution   []puzzle.Move      `json:"solution,omitempty"`
	Generation *Generation        `json:"generation,omitempty"`
}

// Generation records the seed used to create a level.
type Generation struct {
	Seed int64 `json:"seed"`
}

// FromFile validates and converts a JSON level representation.
func FromFile(f File) (puzzle.Level, error) {
	if f.Version != 1 {
		return puzzle.Level{}, fmt.Errorf("unsupported level version %d", f.Version)
	}
	if f.Board.Exit != "" && f.Board.Exit != "right" {
		return puzzle.Level{}, fmt.Errorf("unsupported exit %q", f.Board.Exit)
	}
	l := puzzle.Level{Width: f.Board.Width, Height: f.Board.Height, Target: f.Target, Initial: puzzle.State{Positions: make([]uint8, len(f.Vehicles))}, Vehicles: make([]puzzle.Vehicle, len(f.Vehicles))}
	for i, v := range f.Vehicles {
		o, e := puzzle.ParseOrientation(v.Orientation)
		if e != nil {
			return l, fmt.Errorf("parse vehicle %d orientation: %w", i, e)
		}
		l.Vehicles[i] = puzzle.Vehicle{ID: v.ID, Orientation: o, Length: v.Length, Fixed: v.Fixed}
		l.Initial.Positions[i] = v.Position
	}
	if err := puzzle.ValidateLevelAllowSolved(l); err != nil {
		return l, fmt.Errorf("validate level: %w", err)
	}
	return l, nil
}

// ToFile converts a level to its versioned JSON representation.
func ToFile(l puzzle.Level) File {
	f := File{Version: 1, Target: l.Target}
	f.Board.Width = l.Width
	f.Board.Height = l.Height
	f.Board.Exit = "right"
	f.Vehicles = make([]fileVehicle, len(l.Vehicles))
	for i, v := range l.Vehicles {
		f.Vehicles[i] = fileVehicle{ID: v.ID, Orientation: v.Orientation.String(), Length: v.Length, Fixed: v.Fixed, Position: l.Initial.Positions[i]}
	}
	return f
}

// Load reads and validates a level from a caller-provided path.
func Load(path string) (puzzle.Level, error) {
	b, readErr := os.ReadFile(path) //nolint:gosec // Loading an explicit CLI path is the intended behavior.
	if readErr != nil {
		return puzzle.Level{}, fmt.Errorf("read level %q: %w", path, readErr)
	}
	var f File
	if decodeErr := json.Unmarshal(b, &f); decodeErr != nil {
		return puzzle.Level{}, fmt.Errorf("decode level %q: %w", path, decodeErr)
	}
	l, convertErr := FromFile(f)
	if convertErr != nil {
		return l, fmt.Errorf("invalid level: %w", convertErr)
	}
	return l, nil
}

// Write serializes a level atomically.
func Write(path string, l puzzle.Level) error {
	b, err := json.MarshalIndent(ToFile(l), "", "  ")
	if err != nil {
		return fmt.Errorf("encode level: %w", err)
	}
	return writeAtomic(path, b)
}

// NewGeneratedFile adds analysis and generation metadata to a level.
func NewGeneratedFile(id string, l puzzle.Level, a analyzer.Analysis, solution []puzzle.Move, seed int64) File {
	f := ToFile(l)
	f.ID = id
	f.Analysis = &a
	f.Solution = solution
	f.Generation = &Generation{Seed: seed}
	return f
}

// WriteDataset serializes one level or a level collection atomically.
func WriteDataset(path string, docs []File) error {
	var jsonValue any = docs
	if len(docs) == 1 {
		jsonValue = docs[0]
	}
	jsonData, err := json.MarshalIndent(jsonValue, "", "  ")
	if err != nil {
		return fmt.Errorf("encode level dataset: %w", err)
	}
	return writeAtomic(path, jsonData)
}

// Pack is one manifest entry describing a generated level pack.
type Pack struct {
	ID    string `json:"id"`
	File  string `json:"file"`
	Tier  string `json:"tier"`
	Count int    `json:"count"`
	Index int    `json:"index"`
}

// WritePack serializes one generated level pack atomically.
func WritePack(dir, id string, docs []File) error {
	b, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return fmt.Errorf("encode level pack %q: %w", id, err)
	}
	return writeAtomic(filepath.Join(dir, id+".json"), b)
}

// WriteManifest writes the JSON pack index consumed by the web app.
func WriteManifest(dir string, packs []Pack) error {
	b, err := json.MarshalIndent(packs, "", "  ")
	if err != nil {
		return fmt.Errorf("encode level manifest: %w", err)
	}
	return writeAtomic(filepath.Join(dir, "manifest.json"), b)
}

func writeAtomic(path string, data []byte) (err error) {
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %q: %w", path, err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		if err != nil {
			_ = os.Remove(temporaryPath)
		}
	}()
	if _, writeErr := temporary.Write(data); writeErr != nil {
		err = fmt.Errorf("write temporary file for %q: %w", path, writeErr)
		return err
	}
	if chmodErr := temporary.Chmod(0o644); chmodErr != nil {
		err = fmt.Errorf("set permissions on temporary file for %q: %w", path, chmodErr)
		return err
	}
	if closeErr := temporary.Close(); closeErr != nil {
		err = fmt.Errorf("close temporary file for %q: %w", path, closeErr)
		return err
	}
	if renameErr := os.Rename(temporaryPath, path); renameErr != nil {
		return fmt.Errorf("replace %q: %w", path, renameErr)
	}
	return nil
}
