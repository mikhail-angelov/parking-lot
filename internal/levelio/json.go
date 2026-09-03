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

type Generation struct {
	Seed int64 `json:"seed"`
}

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
			return l, e
		}
		l.Vehicles[i] = puzzle.Vehicle{ID: v.ID, Orientation: o, Length: v.Length, Fixed: v.Fixed}
		l.Initial.Positions[i] = v.Position
	}
	return l, puzzle.ValidateLevelAllowSolved(l)
}
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
func Load(path string) (puzzle.Level, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return puzzle.Level{}, e
	}
	var f File
	if e = json.Unmarshal(b, &f); e != nil {
		return puzzle.Level{}, e
	}
	l, e := FromFile(f)
	if e != nil {
		return l, fmt.Errorf("invalid level: %w", e)
	}
	return l, nil
}
func Write(path string, l puzzle.Level) error {
	b, e := json.MarshalIndent(ToFile(l), "", "  ")
	if e == nil {
		e = writeAtomic(path, b)
	}
	return e
}

func NewGeneratedFile(id string, l puzzle.Level, a analyzer.Analysis, solution []puzzle.Move, seed int64) File {
	f := ToFile(l)
	f.ID = id
	f.Analysis = &a
	f.Solution = solution
	f.Generation = &Generation{Seed: seed}
	return f
}

func WriteDataset(path string, docs []File) error {
	var jsonValue any = docs
	if len(docs) == 1 {
		jsonValue = docs[0]
	}
	jsonData, err := json.MarshalIndent(jsonValue, "", "  ")
	if err != nil {
		return err
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

func WritePack(dir, id string, docs []File) error {
	b, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, id+".json"), b)
}

// WriteManifest writes the JSON pack index consumed by the web app.
func WriteManifest(dir string, packs []Pack) error {
	b, err := json.MarshalIndent(packs, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, "manifest.json"), b)
}

func writeAtomic(path string, data []byte) (err error) {
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		if err != nil {
			_ = os.Remove(temporaryPath)
		}
	}()
	if _, err = temporary.Write(data); err != nil {
		return err
	}
	if err = temporary.Chmod(0644); err != nil {
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
