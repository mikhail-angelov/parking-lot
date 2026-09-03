package solver

import (
	"parking/internal/engine"
	"parking/internal/puzzle"
)

// Options contains breadth-first search limits.
type Options struct {
	MaxDepth  int
	MaxStates int
}

// DefaultMaxDepth is the search depth used when none is specified.
const DefaultMaxDepth = 100

// Result describes an optimal-solution search.
type Result struct {
	Solved         bool
	MinMoves       int
	Solution       []puzzle.Move
	VisitedStates  int
	ExpandedStates int
	MaxFrontier    int
	LimitReached   bool
}

func normalize(o Options) Options {
	if o.MaxDepth <= 0 {
		o.MaxDepth = DefaultMaxDepth
	}
	if o.MaxStates <= 0 {
		o.MaxStates = 2_000_000
	}
	return o
}

type parentInfo struct {
	Parent puzzle.StateKey
	Move   puzzle.Move
	Depth  int
}

// Solve finds an optimal solution with breadth-first search.
func Solve(p *engine.PreparedLevel, o Options) Result {
	o = normalize(o)
	start := p.Level.InitialKey()
	r := Result{MinMoves: -1}
	if engine.IsSolved(p, start) {
		return Result{Solved: true, MinMoves: 0, VisitedStates: 1, ExpandedStates: 0, MaxFrontier: 1}
	}
	q := []puzzle.StateKey{start}
	seen := map[puzzle.StateKey]parentInfo{start: {Parent: start, Depth: 0}}
	r.VisitedStates, r.MaxFrontier = 1, 1
	moves := make([]puzzle.Move, 0, 32)
	depthLimited := false
	for head := 0; head < len(q); head++ {
		cur := q[head]
		info := seen[cur]
		r.ExpandedStates++
		if info.Depth >= o.MaxDepth {
			depthLimited = true
			continue
		}
		moves = engine.GenerateMoves(p, cur, moves)
		for _, m := range moves {
			next := engine.ApplyMove(p, cur, m)
			if _, ok := seen[next]; ok {
				continue
			}
			if len(seen) >= o.MaxStates {
				r.LimitReached = true
				return r
			}
			seen[next] = parentInfo{Parent: cur, Move: m, Depth: info.Depth + 1}
			r.VisitedStates++
			if engine.IsSolved(p, next) {
				r.Solved = true
				r.MinMoves = info.Depth + 1
				r.Solution = reconstruct(seen, next)
				return r
			}
			q = append(q, next)
			if len(q)-head-1 > r.MaxFrontier {
				r.MaxFrontier = len(q) - head - 1
			}
		}
	}
	r.MinMoves = -1
	r.LimitReached = depthLimited
	return r
}
func reconstruct(seen map[puzzle.StateKey]parentInfo, k puzzle.StateKey) []puzzle.Move {
	out := []puzzle.Move{}
	for seen[k].Parent != k {
		n := seen[k]
		out = append(out, n.Move)
		k = n.Parent
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// ExplorationOptions contains full state-space exploration limits.
type ExplorationOptions struct {
	MaxDepth  int
	MaxStates int
}

// Exploration summarizes the reachable state space of a level.
type Exploration struct {
	Solved            bool
	MinMoves          int
	ReachableStates   int
	StatesByDepth     []int
	Transitions       int
	OptimalSolutions  int
	LimitReached      bool
	StateLimitReached bool
	ToGoal            map[puzzle.StateKey]int
}

// Explore measures every state reachable within the configured limits.
func Explore(p *engine.PreparedLevel, o ExplorationOptions) Exploration {
	if o.MaxDepth <= 0 {
		o.MaxDepth = DefaultMaxDepth
	}
	if o.MaxStates <= 0 {
		o.MaxStates = 2_000_000
	}
	start := p.Level.InitialKey()
	if engine.IsSolved(p, start) {
		return Exploration{Solved: true, MinMoves: 0, ReachableStates: 1, StatesByDepth: []int{1}, OptimalSolutions: 1, ToGoal: map[puzzle.StateKey]int{start: 0}}
	}
	dist := map[puzzle.StateKey]int{start: 0}
	ways := map[puzzle.StateKey]int{start: 1}
	q := []puzzle.StateKey{start}
	result := Exploration{MinMoves: -1}
	moves := make([]puzzle.Move, 0, 32)
	for head := 0; head < len(q); head++ {
		cur := q[head]
		d := dist[cur]
		if engine.IsSolved(p, cur) {
			continue
		}
		if d >= o.MaxDepth {
			result.LimitReached = true
			continue
		}
		moves = engine.GenerateMoves(p, cur, moves)
		for _, m := range moves {
			result.Transitions++
			next := engine.ApplyMove(p, cur, m)
			nd := d + 1
			if old, ok := dist[next]; ok {
				if old == nd {
					ways[next] = minCap(ways[next] + ways[cur])
				}
				continue
			}
			if len(dist) >= o.MaxStates {
				result.LimitReached = true
				result.StateLimitReached = true
				return finishExploration(p, result, dist, ways)
			}
			dist[next] = nd
			ways[next] = ways[cur]
			q = append(q, next)
			if engine.IsSolved(p, next) {
				result.Solved = true
				if result.MinMoves < 0 {
					result.MinMoves = nd
				}
			}
		}
	}
	return finishExploration(p, result, dist, ways)
}

func finishExploration(p *engine.PreparedLevel, result Exploration, dist, ways map[puzzle.StateKey]int) Exploration {
	result.ReachableStates = len(dist)
	goals := make([]puzzle.StateKey, 0)
	for state, d := range dist {
		for len(result.StatesByDepth) <= d {
			result.StatesByDepth = append(result.StatesByDepth, 0)
		}
		result.StatesByDepth[d]++
		if d == result.MinMoves && engine.IsSolved(p, state) {
			result.OptimalSolutions = minCap(result.OptimalSolutions + ways[state])
		}
		if engine.IsSolved(p, state) {
			goals = append(goals, state)
		}
	}

	result.ToGoal = make(map[puzzle.StateKey]int, len(dist))
	queue := append([]puzzle.StateKey(nil), goals...)
	for _, goal := range goals {
		result.ToGoal[goal] = 0
	}
	moves := make([]puzzle.Move, 0, 32)
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		moves = engine.GenerateMoves(p, current, moves)
		for _, move := range moves {
			next := engine.ApplyMove(p, current, move)
			if _, explored := dist[next]; !explored {
				continue
			}
			if _, seen := result.ToGoal[next]; seen {
				continue
			}
			result.ToGoal[next] = result.ToGoal[current] + 1
			queue = append(queue, next)
		}
	}
	return result
}
func minCap(n int) int {
	if n > 1000 {
		return 1000
	}
	return n
}
