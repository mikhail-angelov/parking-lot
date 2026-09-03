package analyzer

import (
	"math"
	"parking/internal/engine"
	"parking/internal/puzzle"
	"parking/internal/solver"
)

type DifficultyTier string

const (
	Easy   DifficultyTier = "easy"
	Medium DifficultyTier = "medium"
	Hard   DifficultyTier = "hard"
	Expert DifficultyTier = "expert"
)

type Metrics struct {
	OptimalMoves       int     `json:"optimalMoves"`
	TotalCellDistance  int     `json:"totalCellDistance"`
	VehiclesMoved      int     `json:"vehiclesMoved"`
	VehicleRevisits    int     `json:"vehicleRevisits"`
	DirectionChanges   int     `json:"directionChanges"`
	TargetRegressions  int     `json:"targetRegressions"`
	InitialBlockers    int     `json:"initialBlockers"`
	DependencyDepth    int     `json:"dependencyDepth"`
	DependencyNodes    int     `json:"dependencyNodes"`
	AverageBranching   float64 `json:"averageBranching"`
	MaxBranching       int     `json:"maxBranching"`
	DistractingRatio   float64 `json:"distractingRatio"`
	ForcedMoveRatio    float64 `json:"forcedMoveRatio"`
	ReachableStates    int     `json:"reachableStates"`
	OptimalSolutions   int     `json:"optimalSolutions"`
	ParticipationRatio float64 `json:"participationRatio"`
}
type Config struct {
	MaxDepth         int
	MaxStates        int
	QualityThreshold float64
	MinParticipation float64
	MinOptimalMoves  int
}
type Analysis struct {
	Solvable        bool           `json:"solvable"`
	LimitReached    bool           `json:"limitReached,omitempty"`
	Solution        []puzzle.Move  `json:"-"`
	DifficultyScore float64        `json:"difficultyScore"`
	DifficultyTier  DifficultyTier `json:"difficultyTier"`
	QualityScore    float64        `json:"qualityScore"`
	Accepted        bool           `json:"accepted"`
	RejectReasons   []string       `json:"rejectReasons,omitempty"`
	Metrics         Metrics        `json:"metrics"`
}

type tierPolicy struct {
	tier             DifficultyTier
	minScore         float64
	maxScore         float64
	center           float64
	minParticipation float64
	minMoves         int
}

var tierPolicies = [...]tierPolicy{
	{tier: Easy, minScore: 25, maxScore: 50, center: 37.5, minParticipation: 0.5, minMoves: 4},
	{tier: Medium, minScore: 50, maxScore: 75, center: 62.5, minParticipation: 0.6, minMoves: 10},
	{tier: Hard, minScore: 75, maxScore: 90, center: 82.5, minParticipation: 0.6, minMoves: 15},
	{tier: Expert, minScore: 90, center: 95, minParticipation: 0.6, minMoves: 20},
}

func tier(s float64) DifficultyTier {
	for _, policy := range tierPolicies {
		if policy.maxScore == 0 || s < policy.maxScore {
			return policy.tier
		}
	}
	return Expert
}

// MatchesTier reports whether a score belongs to the requested generation
// band. Scores below 25 are deliberately outside the playable scale: the
// former Medium band is now the minimum Easy difficulty.
func MatchesTier(score float64, requested DifficultyTier) bool {
	policy, ok := policyFor(requested)
	return ok && score >= policy.minScore && (policy.maxScore == 0 || score < policy.maxScore)
}

func TierCenter(tier DifficultyTier) float64 {
	policy, _ := policyFor(tier)
	return policy.center
}

func policyFor(tier DifficultyTier) (tierPolicy, bool) {
	for _, policy := range tierPolicies {
		if policy.tier == tier {
			return policy, true
		}
	}
	return tierPolicy{}, false
}
func clamp(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
func Analyze(p *engine.PreparedLevel, c Config) Analysis {
	return analyze(p, c, 0)
}

// AnalyzeForTier applies the requested tier's optimal-move floor before
// calculating expensive exploration metrics. It is intended for candidate
// generation; Analyze always returns the complete analysis of solvable input.
func AnalyzeForTier(p *engine.PreparedLevel, c Config, requested DifficultyTier) Analysis {
	return analyze(p, c, minimumOptimalMoves(requested))
}

func analyze(p *engine.PreparedLevel, c Config, requiredOptimalMoves int) Analysis {
	if c.QualityThreshold == 0 {
		c.QualityThreshold = 70
	}
	sr := solver.Solve(p, solver.Options{MaxDepth: c.MaxDepth, MaxStates: c.MaxStates})
	a := Analysis{Solvable: sr.Solved, LimitReached: sr.LimitReached, Solution: sr.Solution}
	if engine.IsSolved(p, p.Level.InitialKey()) {
		a.QualityScore = 0
		a.RejectReasons = []string{"initially solved"}
		return a
	}
	if sr.LimitReached {
		a.QualityScore = 0
		a.RejectReasons = []string{"search limit reached"}
		return a
	}
	if !sr.Solved {
		a.QualityScore = 0
		a.RejectReasons = []string{"unsolvable"}
		return a
	}
	m := Metrics{OptimalMoves: sr.MinMoves}
	if requiredOptimalMoves > 0 && m.OptimalMoves < requiredOptimalMoves {
		a.DifficultyTier = tier(0)
		a.QualityScore = 0
		a.RejectReasons = []string{"too few optimal moves"}
		a.Metrics = m
		return a
	}
	seen := map[uint8]bool{}
	counts := map[uint8]int{}
	last := map[uint8]int8{}
	for _, mv := range sr.Solution {
		m.TotalCellDistance += int(mv.Distance())
		seen[mv.Vehicle] = true
		counts[mv.Vehicle]++
		d := mv.Direction()
		if last[mv.Vehicle] != 0 && last[mv.Vehicle] != d {
			m.DirectionChanges++
		}
		last[mv.Vehicle] = d
		if mv.Vehicle == p.Level.Target && mv.To < mv.From {
			m.TargetRegressions++
		}
	}
	m.VehiclesMoved = len(seen)
	for _, n := range counts {
		if n > 1 {
			m.VehicleRevisits += n - 1
		}
	}
	m.ParticipationRatio = float64(m.VehiclesMoved) / float64(len(p.Level.Vehicles))
	m.InitialBlockers = len(engine.TargetBlockingVehicles(p, p.Level.InitialKey()))
	m.DependencyDepth, m.DependencyNodes = dependencyMetrics(p)
	explorationDepth := sr.MinMoves + 4
	configuredDepth := c.MaxDepth
	if configuredDepth <= 0 {
		configuredDepth = solver.DefaultMaxDepth
	}
	if configuredDepth < explorationDepth {
		explorationDepth = configuredDepth
	}
	ex := solver.Explore(p, solver.ExplorationOptions{MaxDepth: explorationDepth, MaxStates: c.MaxStates})
	m.ReachableStates = ex.ReachableStates
	m.OptimalSolutions = ex.OptimalSolutions
	if ex.StateLimitReached {
		a.LimitReached = true
		a.QualityScore = 0
		a.RejectReasons = []string{"exploration limit reached"}
		a.Metrics = m
		return a
	}
	m.AverageBranching, m.MaxBranching, m.DistractingRatio, m.ForcedMoveRatio = branching(p, sr.Solution, ex.ToGoal)
	score := 100 * (.30*clamp(float64(m.OptimalMoves-4)/26) + .20*clamp(float64(m.DependencyDepth)/5) + .12*clamp(float64(m.TargetRegressions)/3) + .10*clamp(float64(m.DirectionChanges)/8) + .12*m.DistractingRatio + .10*clamp(math.Log2(float64(m.ReachableStates+1))/16) + .06*clamp(float64(m.VehicleRevisits)/10))
	a.DifficultyScore = score
	a.DifficultyTier = tier(score)
	a.Metrics = m
	a.QualityScore = 100
	if m.InitialBlockers == 0 {
		a.QualityScore -= 40
		a.RejectReasons = append(a.RejectReasons, "no initial blockers")
	}
	minParticipation := c.MinParticipation
	if minParticipation == 0 {
		minParticipation = defaultMinParticipation(a.DifficultyTier)
	}
	if minParticipation > 0 && m.ParticipationRatio < minParticipation {
		a.QualityScore -= 25
		a.RejectReasons = append(a.RejectReasons, "low participation")
	}
	if m.OptimalSolutions >= 1000 {
		a.QualityScore -= 15
		a.RejectReasons = append(a.RejectReasons, "too many optimal solutions")
	}
	if m.DependencyDepth == 0 {
		a.QualityScore -= 10
		a.RejectReasons = append(a.RejectReasons, "no dependency depth")
	}
	if m.VehiclesMoved < 4 {
		a.QualityScore -= 10
		a.RejectReasons = append(a.RejectReasons, "too few vehicles moved")
	}
	minMoves := c.MinOptimalMoves
	if minMoves == 0 {
		minMoves = minimumOptimalMoves(a.DifficultyTier)
	}
	if m.OptimalMoves < minMoves {
		a.QualityScore -= 15
		a.RejectReasons = append(a.RejectReasons, "too few optimal moves")
	}
	if a.QualityScore < 0 {
		a.QualityScore = 0
	}
	a.Accepted = a.QualityScore >= c.QualityThreshold
	return a
}

func defaultMinParticipation(tier DifficultyTier) float64 {
	policy, _ := policyFor(tier)
	return policy.minParticipation
}

func minimumOptimalMoves(tier DifficultyTier) int {
	policy, ok := policyFor(tier)
	if !ok {
		return 1
	}
	return policy.minMoves
}
func dependencyMetrics(p *engine.PreparedLevel) (int, int) {
	k := p.Level.InitialKey()
	direct := engine.TargetBlockingVehicles(p, k)
	allNodes := make(map[int]bool)
	depth := 0
	for _, blocker := range direct {
		allNodes[blocker.Vehicle] = true
		result := resolveDependency(p, k, blocker.Vehicle, blocker.Cells, map[int]bool{int(p.Level.Target): true})
		branchDepth := 1
		if result.ok {
			branchDepth += result.depth
			for node := range result.nodes {
				allNodes[node] = true
			}
		}
		if branchDepth > depth {
			depth = branchDepth
		}
	}
	return depth, len(allNodes)
}

type dependencyResult struct {
	depth int
	nodes map[int]bool
	ok    bool
}

func resolveDependency(p *engine.PreparedLevel, k puzzle.StateKey, vehicle int, conflict engine.BoardMask, stack map[int]bool) dependencyResult {
	if stack[vehicle] {
		return dependencyResult{}
	}
	stack[vehicle] = true
	defer delete(stack, vehicle)

	from := puzzle.Position(k, vehicle)
	best := dependencyResult{}
	for position := 0; position < engine.PositionCount(p, vehicle); position++ {
		to := uint8(position)
		if to == from || !engine.PlacementClears(p, vehicle, to, conflict) {
			continue
		}
		candidate := dependencyResult{nodes: make(map[int]bool), ok: true}
		for _, blocker := range engine.BlockingVehiclesForMove(p, k, vehicle, to) {
			if stack[blocker.Vehicle] {
				candidate.ok = false
				break
			}
			child := resolveDependency(p, k, blocker.Vehicle, blocker.Cells, stack)
			if !child.ok {
				candidate.ok = false
				break
			}
			candidate.nodes[blocker.Vehicle] = true
			for node := range child.nodes {
				candidate.nodes[node] = true
			}
			if child.depth+1 > candidate.depth {
				candidate.depth = child.depth + 1
			}
		}
		if candidate.ok && (!best.ok || candidate.depth < best.depth || candidate.depth == best.depth && len(candidate.nodes) < len(best.nodes)) {
			best = candidate
		}
	}
	return best
}

func branching(p *engine.PreparedLevel, solution []puzzle.Move, toGoal map[puzzle.StateKey]int) (float64, int, float64, float64) {
	if len(solution) == 0 {
		return 0, 0, 0, 0
	}
	k := p.Level.InitialKey()
	branchingSum, maxBranch, total, distracting, forcedStates, pathStates := 0, 0, 0, 0, 0, 0
	for step, m := range solution {
		moves := engine.GenerateMoves(p, k, nil)
		if len(moves) == 0 {
			k = engine.ApplyMove(p, k, m)
			continue
		}
		branchingSum += len(moves)
		if len(moves) > maxBranch {
			maxBranch = len(moves)
		}
		good := 0
		remaining := len(solution) - step - 1
		for _, candidate := range moves {
			next := engine.ApplyMove(p, k, candidate)
			total++
			if distance, ok := toGoal[next]; ok && distance == remaining {
				good++
			} else {
				distracting++
			}
		}
		if good == 1 {
			forcedStates++
		}
		pathStates++
		k = engine.ApplyMove(p, k, m)
	}
	n := float64(len(solution))
	distractingRatio, forcedRatio := 0.0, 0.0
	if total > 0 {
		distractingRatio = float64(distracting) / float64(total)
	}
	if pathStates > 0 {
		forcedRatio = float64(forcedStates) / float64(pathStates)
	}
	return float64(branchingSum) / n, maxBranch, distractingRatio, forcedRatio
}
