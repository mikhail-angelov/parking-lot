package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"parking/internal/analyzer"
	"parking/internal/engine"
	"parking/internal/puzzle"
	"sort"
)

type Config struct {
	Tier         analyzer.DifficultyTier
	MinVehicles  int
	MaxVehicles  int
	ScrambleMin  int
	ScrambleMax  int
	MaxAttempts  int
	MaxMutations int
	Seed         int64
}
type GeneratedLevel struct {
	Level    puzzle.Level
	Analysis analyzer.Analysis
	Solution []puzzle.Move
	Seed     int64
}

type evaluatedCandidate struct {
	generated *GeneratedLevel
	prepared  *engine.PreparedLevel
}

func Defaults(t analyzer.DifficultyTier) Config {
	c := Config{Tier: t, MaxAttempts: 100, MaxMutations: 100}
	switch t {
	case analyzer.Easy:
		c.MinVehicles, c.MaxVehicles, c.ScrambleMin, c.ScrambleMax = 9, 12, 20, 45
		c.MaxMutations = 20
	case analyzer.Medium:
		c.MinVehicles, c.MaxVehicles, c.ScrambleMin, c.ScrambleMax = 11, 15, 35, 70
		c.MaxAttempts = 1000
		c.MaxMutations = 20
	case analyzer.Hard, analyzer.Expert:
		c.MinVehicles, c.MaxVehicles, c.ScrambleMin, c.ScrambleMax = 13, 18, 50, 100
	}
	return c
}
func Generate(c Config) (*GeneratedLevel, error) {
	if c.Tier != analyzer.Easy && c.Tier != analyzer.Medium && c.Tier != analyzer.Hard && c.Tier != analyzer.Expert {
		return nil, fmt.Errorf("invalid difficulty %q", c.Tier)
	}
	if c.MinVehicles < 1 || c.MaxVehicles < c.MinVehicles || c.MaxVehicles > 18 {
		return nil, fmt.Errorf("invalid vehicle range")
	}
	if c.MaxAttempts <= 0 {
		return nil, fmt.Errorf("max attempts must be positive")
	}
	if c.ScrambleMin < 0 || c.ScrambleMax < c.ScrambleMin {
		return nil, fmt.Errorf("invalid scramble range")
	}
	if c.MaxMutations < 0 {
		return nil, fmt.Errorf("max mutations must be non-negative")
	}
	r := rand.New(rand.NewSource(c.Seed))
	bestScore := -1.0
	bestTier := analyzer.DifficultyTier("")
	var bestMetrics analyzer.Metrics
	for attempt := 0; attempt < c.MaxAttempts; attempt++ {
		l, e := layout(r, c)
		if e != nil {
			continue
		}
		p, err := engine.Prepare(l)
		if err != nil {
			continue
		}
		steps := c.ScrambleMin
		if c.ScrambleMax > c.ScrambleMin {
			steps += r.Intn(c.ScrambleMax - c.ScrambleMin + 1)
		}
		k := l.InitialKey()
		prev := puzzle.Move{Vehicle: 255}
		seenStates := map[puzzle.StateKey]bool{k: true}
		for i := 0; i < steps; i++ {
			ms := engine.GenerateMoves(p, k, nil)
			if len(ms) == 0 {
				break
			}
			choices := make([]puzzle.Move, 0, len(ms))
			for _, m := range ms {
				if m.Vehicle == prev.Vehicle && m.From == prev.To && m.To == prev.From && len(ms) > 1 {
					continue
				}
				choices = append(choices, m)
			}
			if len(choices) == 0 {
				choices = ms
			}
			m := choices[r.Intn(len(choices))]
			bestScore := -1
			for _, candidate := range choices {
				next := engine.ApplyMove(p, k, candidate)
				score := blockerCount(p, next) * 5
				if candidate.Vehicle == l.Target && candidate.To < candidate.From {
					score += 4
				}
				if !seenStates[next] {
					score += 3
				}
				score += r.Intn(4)
				if score > bestScore {
					bestScore, m = score, candidate
				}
			}
			k = engine.ApplyMove(p, k, m)
			seenStates[k] = true
			prev = m
		}
		if c.Tier == analyzer.Expert {
			k = deepestState(p, l.InitialKey(), minInt(steps, 50), 2000000)
		} else if c.Tier == analyzer.Hard {
			k = deepestState(p, l.InitialKey(), minInt(steps, 35), 500000)
		}
		l.Initial.Positions = decode(k, len(l.Vehicles))
		evaluated, err := evaluateCandidate(l, c.Seed, c.Tier)
		if err != nil || !evaluated.generated.Analysis.Solvable {
			continue
		}
		p = evaluated.prepared
		a := evaluated.generated.Analysis
		if a.DifficultyScore > bestScore {
			bestScore, bestTier, bestMetrics = a.DifficultyScore, a.DifficultyTier, a.Metrics
		}
		best := evaluated.generated
		if analyzer.MatchesTier(a.DifficultyScore, c.Tier) && a.Accepted {
			return best, nil
		}
		for mutation := 0; mutation < c.MaxMutations; mutation++ {
			if (c.Tier == analyzer.Hard || c.Tier == analyzer.Expert) && mutation%3 == 0 {
				if mutated, ok := randomStructuralMutation(l, r); ok {
					candidate, err := evaluateCandidate(mutated, c.Seed, c.Tier)
					if err == nil && candidate.generated.Analysis.Solvable {
						ca := candidate.generated.Analysis
						if ca.DifficultyScore > bestScore {
							bestScore, bestTier, bestMetrics = ca.DifficultyScore, ca.DifficultyTier, ca.Metrics
						}
						if analyzer.MatchesTier(ca.DifficultyScore, c.Tier) && ca.Accepted {
							return candidate.generated, nil
						}
						if closer(ca, c.Tier, best.Analysis) {
							best = candidate.generated
							l, p, k = mutated, candidate.prepared, mutated.InitialKey()
						}
					}
				}
				continue
			}
			if mutation%2 == 0 {
				if next, ok := randomPositionMutation(l, r); ok {
					k = next
				}
			} else {
				ms := engine.GenerateMoves(p, k, nil)
				if len(ms) == 0 {
					break
				}
				k = engine.ApplyMove(p, k, ms[r.Intn(len(ms))])
			}
			candidate := l
			candidate.Initial.Positions = decode(k, len(l.Vehicles))
			evaluated, err := evaluateCandidate(candidate, c.Seed, c.Tier)
			if err != nil || !evaluated.generated.Analysis.Solvable {
				continue
			}
			ca := evaluated.generated.Analysis
			if ca.DifficultyScore > bestScore {
				bestScore, bestTier, bestMetrics = ca.DifficultyScore, ca.DifficultyTier, ca.Metrics
			}
			if analyzer.MatchesTier(ca.DifficultyScore, c.Tier) && ca.Accepted {
				return evaluated.generated, nil
			}
			if closer(ca, c.Tier, best.Analysis) {
				best = evaluated.generated
				l, p, k = candidate, evaluated.prepared, candidate.InitialKey()
			}
		}
	}
	return nil, fmt.Errorf("attempt limit reached (best score %.2f, tier %s, moves %d, depth %d, regressions %d, direction changes %d)", bestScore, bestTier, bestMetrics.OptimalMoves, bestMetrics.DependencyDepth, bestMetrics.TargetRegressions, bestMetrics.DirectionChanges)
}

func evaluateCandidate(level puzzle.Level, seed int64, tier analyzer.DifficultyTier) (*evaluatedCandidate, error) {
	if err := puzzle.ValidateLevel(level); err != nil {
		return nil, err
	}
	prepared, err := engine.Prepare(level)
	if err != nil {
		return nil, err
	}
	analysis := analyzer.AnalyzeForTier(prepared, analyzer.Config{}, tier)
	return &evaluatedCandidate{
		generated: &GeneratedLevel{Level: level, Analysis: analysis, Solution: analysis.Solution, Seed: seed},
		prepared:  prepared,
	}, nil
}
func randomStructuralMutation(l puzzle.Level, r *rand.Rand) (puzzle.Level, bool) {
	if len(l.Vehicles) < 2 {
		return l, false
	}
	candidate := l
	candidate.Vehicles = append([]puzzle.Vehicle(nil), l.Vehicles...)
	candidate.Initial.Positions = append([]uint8(nil), l.Initial.Positions...)
	i := r.Intn(len(l.Vehicles))
	v := candidate.Vehicles[i]
	if i == int(l.Target) {
		v.Fixed = uint8(1 + r.Intn(4))
		candidate.Vehicles[i] = v
		ps := append([]uint8(nil), candidate.Initial.Positions...)
		ps[i] = uint8(r.Intn(5))
		candidate.Initial.Positions = ps
		if validPositions(candidate, ps) {
			return candidate, true
		}
		return l, false
	}
	if r.Intn(3) == 0 {
		v.Orientation = puzzle.Orientation(r.Intn(2))
	}
	if r.Intn(3) == 0 {
		if v.Length == 2 {
			v.Length = 3
		} else {
			v.Length = 2
		}
	}
	v.Fixed = uint8(r.Intn(6))
	candidate.Vehicles[i] = v
	ps := append([]uint8(nil), candidate.Initial.Positions...)
	ps[i] = uint8(r.Intn(int(7 - v.Length)))
	candidate.Initial.Positions = ps
	if validPositions(candidate, ps) {
		return candidate, true
	}
	return l, false
}
func randomPositionMutation(l puzzle.Level, r *rand.Rand) (puzzle.StateKey, bool) {
	ps := append([]uint8(nil), l.Initial.Positions...)
	for tries := 0; tries < 30; tries++ {
		i := r.Intn(len(l.Vehicles))
		if i == int(l.Target) {
			ps[i] = uint8(r.Intn(5))
			if validPositions(l, ps) {
				return puzzle.EncodeState(ps), true
			}
			continue
		}
		limit := int(6 - l.Vehicles[i].Length)
		ps[i] = uint8(r.Intn(limit + 1))
		if validPositions(l, ps) {
			return puzzle.EncodeState(ps), true
		}
	}
	return l.InitialKey(), false
}
func validPositions(l puzzle.Level, ps []uint8) bool {
	occupied := map[[2]uint8]bool{}
	for i, v := range l.Vehicles {
		for n := uint8(0); n < v.Length; n++ {
			x, y := ps[i]+n, v.Fixed
			if v.Orientation == puzzle.Vertical {
				x, y = v.Fixed, ps[i]+n
			}
			if occupied[[2]uint8{x, y}] {
				return false
			}
			occupied[[2]uint8{x, y}] = true
		}
	}
	return true
}
func blockerCount(p *engine.PreparedLevel, k puzzle.StateKey) int {
	return len(engine.TargetBlockingVehicles(p, k))
}
func deepestState(p *engine.PreparedLevel, start puzzle.StateKey, maxDepth, maxStates int) puzzle.StateKey {
	type node struct {
		k     puzzle.StateKey
		depth int
	}
	q := []node{{start, 0}}
	seen := map[puzzle.StateKey]bool{start: true}
	deep := start
	for head := 0; head < len(q) && len(seen) < maxStates; head++ {
		cur := q[head]
		if cur.depth > q[0].depth {
			deep = cur.k
		}
		if cur.depth >= maxDepth {
			deep = cur.k
			continue
		}
		for _, m := range engine.GenerateMoves(p, cur.k, nil) {
			next := engine.ApplyMove(p, cur.k, m)
			if !seen[next] {
				seen[next] = true
				q = append(q, node{next, cur.depth + 1})
				if cur.depth+1 >= q[len(q)-1].depth {
					deep = next
				}
			}
		}
	}
	return deep
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func closer(a analyzer.Analysis, target analyzer.DifficultyTier, b analyzer.Analysis) bool {
	return fitness(a, target) < fitness(b, target)
}

func fitness(a analyzer.Analysis, target analyzer.DifficultyTier) float64 {
	if !a.Solvable || a.LimitReached {
		return math.Inf(1)
	}
	score := abs(a.DifficultyScore - analyzer.TierCenter(target))
	if a.QualityScore < 70 {
		score += (70 - a.QualityScore) * 3
	}
	return score
}
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
func layout(r *rand.Rand, c Config) (puzzle.Level, error) {
	n := c.MinVehicles
	if c.MaxVehicles > c.MinVehicles {
		n += r.Intn(c.MaxVehicles - c.MinVehicles + 1)
	}
	l := puzzle.Level{Width: 6, Height: 6, Target: 0, Initial: puzzle.State{Positions: make([]uint8, n)}, Vehicles: make([]puzzle.Vehicle, n)}
	l.Vehicles[0] = puzzle.Vehicle{ID: 0, Orientation: puzzle.Horizontal, Length: 2, Fixed: uint8(1 + r.Intn(4))}
	l.Initial.Positions[0] = uint8(1 + r.Intn(3))
	occupied := map[[2]uint8]bool{}
	for x := int(l.Initial.Positions[0]); x < int(l.Initial.Positions[0])+2; x++ {
		occupied[[2]uint8{uint8(x), l.Vehicles[0].Fixed}] = true
	}
	for i := 1; i < n; i++ {
		ok := false
		for tries := 0; tries < 100 && !ok; tries++ {
			o := puzzle.Orientation(r.Intn(2))
			length := uint8(2)
			if r.Intn(4) == 0 {
				length = 3
			}
			fixed := uint8(r.Intn(6))
			pos := uint8(r.Intn(int(7 - length)))
			good := true
			for z := uint8(0); z < length; z++ {
				x, y := pos+z, fixed
				if o == puzzle.Vertical {
					x, y = fixed, pos+z
				}
				if y == l.Vehicles[0].Fixed && x >= 2 {
					good = false
					break
				}
				if occupied[[2]uint8{x, y}] {
					good = false
					break
				}
			}
			if good {
				l.Vehicles[i] = puzzle.Vehicle{ID: uint8(i), Orientation: o, Length: length, Fixed: fixed}
				l.Initial.Positions[i] = pos
				for z := uint8(0); z < length; z++ {
					x, y := pos+z, fixed
					if o == puzzle.Vertical {
						x, y = fixed, pos+z
					}
					occupied[[2]uint8{x, y}] = true
				}
				ok = true
			}
		}
		if !ok {
			return l, fmt.Errorf("layout exhausted")
		}
	}
	return l, nil
}
func decode(k puzzle.StateKey, n int) []uint8 {
	p := make([]uint8, n)
	for i := range p {
		p[i] = puzzle.Position(k, i)
	}
	return p
}
func CanonicalHash(l puzzle.Level) string {
	type f struct {
		o                  puzzle.Orientation
		fixed, length, pos uint8
	}
	a := make([]f, 0, len(l.Vehicles))
	for i, v := range l.Vehicles {
		if i == int(l.Target) {
			continue
		}
		a = append(a, f{v.Orientation, v.Fixed, v.Length, l.Initial.Positions[i]})
	}
	sort.Slice(a, func(i, j int) bool {
		if a[i].o != a[j].o {
			return a[i].o < a[j].o
		}
		if a[i].fixed != a[j].fixed {
			return a[i].fixed < a[j].fixed
		}
		if a[i].length != a[j].length {
			return a[i].length < a[j].length
		}
		return a[i].pos < a[j].pos
	})
	h := sha256.New()
	target := l.Vehicles[l.Target]
	fmt.Fprintf(h, "%d:%d:%d:%d:%d:%d", l.Width, l.Height, target.Orientation, target.Fixed, target.Length, l.Initial.Positions[l.Target])
	for _, x := range a {
		fmt.Fprintf(h, ";%d,%d,%d,%d", x.o, x.fixed, x.length, x.pos)
	}
	return hex.EncodeToString(h.Sum(nil))
}
