# Parking Puzzle Engine — Specification

Status: Draft v1  
Language: Go  
Scope: level model, move engine, optimal solver, analyzer, procedural level generator, CLI

---

## 1. Goal

Build an offline toolchain for a Rush Hour / parking-lot style puzzle game.

The system must be able to:

1. represent a parking puzzle compactly;
2. enumerate all legal moves;
3. find an optimal solution;
4. analyze why a level is easy or difficult;
5. generate solvable levels for a requested difficulty tier;
6. reject low-quality or duplicate levels;
7. generate large deterministic level sets in parallel.

The generator is intended primarily as a development-time tool. The mobile game can consume pre-generated JSON levels and does not need to solve or generate levels at runtime.

---

## 2. Design principles

### 2.1 Separation of responsibilities

The architecture is built around three independent subsystems:

- **Solver** — answers whether a level is solvable and returns an optimal solution.
- **Analyzer** — measures structural and search-space properties of a level and derives difficulty/quality metrics.
- **Generator** — produces candidate levels and uses Solver + Analyzer to accept, mutate, or reject them.

The generator must not contain hidden difficulty logic that is unavailable to the analyzer.

### 2.2 KISS / YAGNI

Version 1 deliberately avoids:

- A* / IDA* unless BFS proves insufficient;
- genetic algorithms;
- machine-learning difficulty prediction;
- arbitrary board shapes;
- obstacles;
- multiple exits;
- special vehicle behavior;
- runtime generation in the game client;
- persistence/database abstractions;
- interfaces with only one implementation.

### 2.3 Determinism

Every generation command must accept a seed.

For the same:

- executable version;
- configuration;
- seed;

it must produce the same sequence of candidates and accepted levels when run with one worker.

Parallel bulk generation does not need to preserve output ordering, but each generated job must derive its own deterministic seed.

---

## 3. MVP game rules

### 3.1 Board

Version 1 uses a fixed:

```text
6 × 6
```

board.

Coordinates:

```text
x = 0..5, left to right
y = 0..5, top to bottom
```

### 3.2 Vehicles

Each vehicle:

- is horizontal or vertical;
- has length 2 or 3;
- moves only along its orientation axis;
- cannot overlap another vehicle;
- cannot leave the board.

### 3.3 Target vehicle

For MVP:

- the target is horizontal;
- its fixed row is the exit row;
- the exit is on the right side of the board.

A level is solved when the target vehicle can leave through the right exit.

For solver purposes, the solved predicate is:

> there are no occupied cells between the target's right edge and the board's right edge.

The player does not need a separate "move outside the board" action.

### 3.4 Move semantics

One move means moving one vehicle by any positive number of free cells in one direction.

Example:

```text
AA....
```

may become:

```text
...AA.
```

in one move.

This is the primary metric used for optimal solution length.

Cell distance may be tracked as a secondary metric.

---

## 4. Package layout

```text
cmd/
  parking/
    main.go

internal/
  puzzle/
    level.go
    state.go
    move.go
    encode.go
    validate.go

  engine/
    board.go
    masks.go
    moves.go
    goal.go

  solver/
    bfs.go
    result.go
    explore.go

  analyzer/
    analyzer.go
    solution.go
    dependency.go
    branching.go
    difficulty.go
    quality.go

  generator/
    config.go
    layout.go
    scramble.go
    mutate.go
    generate.go
    bulk.go

  levelio/
    json.go

  render/
    ascii.go
```

No package may depend on `cmd/parking`.

Dependency direction:

```text
puzzle
  ↑
engine
  ↑
solver     analyzer
   ↑        ↑
    \      /
     generator
```

`analyzer` may call `solver` when analysis requires graph exploration.

---

# 5. Core model

## 5.1 Orientation

```go
type Orientation uint8

const (
    Horizontal Orientation = iota
    Vertical
)
```

## 5.2 Vehicle

```go
type Vehicle struct {
    ID          uint8
    Orientation Orientation
    Length      uint8
    Fixed       uint8
}
```

Meaning of `Fixed`:

- horizontal vehicle: fixed Y coordinate;
- vertical vehicle: fixed X coordinate.

The changing coordinate is stored in `State`.

Constraints:

```text
Length ∈ {2, 3}
Fixed  ∈ [0, 5]
```

Vehicle IDs must be dense and correspond to their index in `Level.Vehicles`:

```text
Vehicles[i].ID == uint8(i)
```

This makes state encoding and hot-path loops simpler.

## 5.3 State

Public/readable representation:

```go
type State struct {
    Positions []uint8
}
```

`Positions[i]` is:

- X for horizontal vehicle i;
- Y for vertical vehicle i.

The slice length must equal `len(Level.Vehicles)`.

### 5.3.1 Internal encoded state

The solver hot path uses:

```go
type StateKey uint64
```

A 6×6 board requires 3 bits per moving coordinate.

With at most 18 vehicles:

```text
18 × 3 = 54 bits
```

which fits in `uint64`.

The MVP must cap the number of vehicles at 18.

Encoding:

```go
func EncodeState(positions []uint8) StateKey {
    var key uint64
    for i, p := range positions {
        key |= uint64(p) << (i * 3)
    }
    return StateKey(key)
}
```

Decoding should normally be avoided in solver hot paths. Helper accessors may read or modify a single 3-bit coordinate directly.

Suggested helpers:

```go
func Position(key StateKey, vehicle int) uint8
func WithPosition(key StateKey, vehicle int, pos uint8) StateKey
```

## 5.4 Level

```go
type Level struct {
    Width    uint8
    Height   uint8
    Target   uint8
    Vehicles []Vehicle
    Initial  State
}
```

For MVP:

```text
Width  = 6
Height = 6
Target vehicle orientation = Horizontal
Exit = right side
Exit row = Vehicles[Target].Fixed
```

The exit is therefore implicit and does not need to be stored in v1 JSON.

---

# 6. Move model

```go
type Move struct {
    Vehicle uint8
    From    uint8
    To      uint8
}
```

Constraints:

```text
From != To
```

Direction and distance are derived:

```go
func (m Move) Distance() uint8
func (m Move) Direction() int8 // -1 or +1
```

`From` is retained because it makes:

- debugging;
- solution rendering;
- reverse moves;
- corruption detection

simpler.

---

# 7. Board representation

The board has 36 cells and fits in `uint64`.

Bit index:

```text
bit = y*6 + x
```

Only bits 0..35 are used.

```go
type BoardMask uint64
```

## 7.1 Precomputed vehicle masks

For a given level, precompute every legal geometric placement of every vehicle:

```go
type PreparedLevel struct {
    Level Level

    // Masks[v][position] gives occupied cells for vehicle v.
    Masks [][]BoardMask
}
```

For a horizontal length-2 vehicle, valid positions are 0..4.
For a horizontal length-3 vehicle, valid positions are 0..3.
The same applies vertically.

This avoids rebuilding masks during BFS.

## 7.2 Occupancy

```go
func Occupancy(p *PreparedLevel, state StateKey) BoardMask
```

Implementation:

```text
occupied = 0
for each vehicle v:
    pos = Position(state, v)
    occupied |= Masks[v][pos]
```

No heap allocation is allowed in this hot path.

---

# 8. Validation

`puzzle.ValidateLevel` must reject malformed levels before solver execution.

Checks:

1. board is exactly 6×6 in v1;
2. vehicle count is 1..18;
3. IDs are dense and ordered;
4. target index exists;
5. target is horizontal;
6. every length is 2 or 3;
7. every fixed coordinate is in range;
8. every initial moving coordinate is legal for its vehicle length;
9. no vehicles overlap in the initial state;
10. target lies in a valid exit row;
11. initial state is not already solved unless explicitly allowed by generator tests.

Return structured errors where practical.

---

# 9. Move generation

Public API:

```go
func GenerateMoves(p *PreparedLevel, state StateKey, dst []Move) []Move
```

The caller may pass a reusable destination slice to reduce allocations.

Algorithm for each vehicle:

1. build total occupancy;
2. remove the current vehicle's own mask;
3. scan one cell at a time in the negative direction until blocked/boundary;
4. each reachable position produces one move;
5. scan the positive direction similarly.

Because one move may travel any distance, every reachable destination along the free segment is emitted.

Example:

```text
..AA..
```

with all other cells free yields destinations:

```text
x=0
x=1
x=3
x=4
```

subject to vehicle length.

Move ordering must be deterministic.

Recommended order:

```text
vehicle ID ascending
negative-direction destinations nearest → farthest
positive-direction destinations nearest → farthest
```

Deterministic ordering is useful for reproducible solutions and tests.

---

# 10. Apply move

```go
func ApplyMove(p *PreparedLevel, state StateKey, move Move) StateKey
```

The hot-path version assumes the move was produced by `GenerateMoves`.

A separate checked helper may exist for tests / CLI input:

```go
func ApplyMoveChecked(...) (StateKey, error)
```

Applying a move is only a 3-bit update in `StateKey`.

---

# 11. Goal detection

```go
func IsSolved(p *PreparedLevel, state StateKey) bool
```

For the target vehicle:

1. calculate its rightmost occupied X;
2. inspect cells between that X and board edge on the target row;
3. if none are occupied by other vehicles, return true.

The target does not need to be positioned at x=4/3 itself; it only needs a clear path to the exit.

This matches common Rush Hour semantics where the final exit slide is not counted as an additional puzzle move.

---

# 12. Solver

## 12.1 V1 algorithm

Use Breadth-First Search.

Reasons:

- guarantees minimum move count;
- simple and easy to verify;
- board state is compact;
- search statistics are useful for the analyzer;
- no heuristic tuning required.

Do not implement A* until benchmarks demonstrate a need.

## 12.2 API

```go
type Options struct {
    MaxDepth  int
    MaxStates int
}

type Result struct {
    Solved bool

    MinMoves int
    Solution []puzzle.Move

    VisitedStates  int
    ExpandedStates int
    MaxFrontier     int

    LimitReached bool
}

func Solve(p *engine.PreparedLevel, opts Options) Result
```

Defaults:

```text
MaxDepth  = 100
MaxStates = 2,000,000
```

The actual limits must be configurable from CLI.

## 12.3 BFS node metadata

Use:

```go
type parentInfo struct {
    Parent puzzle.StateKey
    Move   puzzle.Move
    Depth  uint16
}
```

Visited map:

```go
map[puzzle.StateKey]parentInfo
```

The initial state can use itself as parent or a separate sentinel.

## 12.4 BFS pseudocode

```text
initial = Encode(level.Initial)

if IsSolved(initial):
    return solved(depth=0)

queue.push(initial)
visited[initial] = root

while queue not empty:
    current = queue.pop()

    if depth(current) >= MaxDepth:
        continue

    moves = GenerateMoves(current)

    for move in moves:
        next = ApplyMove(current, move)

        if next in visited:
            continue

        visited[next] = {
            parent: current,
            move: move,
            depth: depth(current)+1,
        }

        if IsSolved(next):
            return reconstruct(next)

        queue.push(next)

        if len(visited) >= MaxStates:
            return limit-reached

return unsolved
```

Because BFS discovers states in nondecreasing depth, the first solved state is optimal.

## 12.5 Path reconstruction

Walk parent links from goal to root, append moves, then reverse the slice.

No full state path needs to be stored in `Result` by default.

---

# 13. Exploration API

The analyzer needs more than the first solution.

Provide:

```go
type ExplorationOptions struct {
    MaxDepth  int
    MaxStates int
}

type Exploration struct {
    Solved   bool
    MinMoves int

    ReachableStates int
    StatesByDepth   []int

    // Number of directed generated transitions seen during exploration.
    Transitions int

    // Capped count.
    OptimalSolutions int

    LimitReached bool
}

func Explore(p *engine.PreparedLevel, opts ExplorationOptions) Exploration
```

V1 exploration should continue at least through `MinMoves` depth after the first goal is found so it can count shortest-path alternatives.

For expensive graph metrics, analysis may use configurable caps.

---

# 14. Counting optimal solutions

We want a capped count of distinct shortest move sequences.

During BFS maintain:

```text
distance[state]
ways[state]
```

Rules:

```text
first discovery of next:
    distance[next] = distance[current] + 1
    ways[next] = ways[current]

same-depth rediscovery:
    ways[next] += ways[current]
```

Cap counts at:

```text
1000
```

A solved state can be any state where the target path is clear, so the final optimal-solution count is the capped sum of ways for all solved states at minimum depth.

The count is diagnostic, not part of correctness.

---

# 15. Analyzer

## 15.1 API

```go
type Analysis struct {
    Solvable bool

    DifficultyScore float64
    DifficultyTier  DifficultyTier

    QualityScore float64
    Accepted     bool
    RejectReasons []string

    Metrics Metrics
}

func Analyze(p *engine.PreparedLevel, cfg Config) Analysis
```

Raw metrics must always be persisted separately from derived scores.

## 15.2 Metrics

```go
type Metrics struct {
    OptimalMoves       int
    TotalCellDistance  int
    VehiclesMoved      int
    VehicleRevisits    int
    DirectionChanges   int
    TargetRegressions  int

    InitialBlockers    int
    DependencyDepth    int
    DependencyNodes    int

    AverageBranching   float64
    MaxBranching       int
    DistractingRatio   float64
    ForcedMoveRatio    float64

    ReachableStates    int
    OptimalSolutions   int

    ParticipationRatio float64
}
```

Some metrics may initially be approximate. Approximation must be documented and deterministic.

---

# 16. Solution metrics

Given an optimal solution:

```text
m0, m1, ..., mn
```

calculate:

## 16.1 OptimalMoves

```text
len(solution)
```

## 16.2 TotalCellDistance

Sum:

```text
abs(move.To - move.From)
```

## 16.3 VehiclesMoved

Number of unique vehicle IDs appearing in the solution.

## 16.4 VehicleRevisits

For each vehicle:

```text
max(0, numberOfMovesByVehicle - 1)
```

Sum across all vehicles.

This captures puzzles where the same vehicle must be manipulated repeatedly.

## 16.5 DirectionChanges

For each vehicle, inspect its sequence of moves.

A direction change occurs when consecutive moves of that vehicle have opposite signs.

Example:

```text
A +2
...
A -1
```

counts as one direction change for A.

## 16.6 TargetRegressions

The target's desired direction is positive X toward the right exit.

Any target move with:

```text
To < From
```

counts as a regression.

This is a strong difficulty signal because the player must temporarily move the goal vehicle away from the exit.

---

# 17. Participation ratio

```text
ParticipationRatio = VehiclesMoved / TotalVehicles
```

The target counts as a vehicle.

This metric is mainly a quality filter.

A level with 15 vehicles but only 4 participating in the optimal solution contains significant decorative noise.

Initial recommended quality threshold for Easy:

```text
ParticipationRatio >= 0.50
```

For Medium/Hard/Expert:

```text
ParticipationRatio >= 0.60
```

These thresholds are configuration, not hard-coded game rules.

---

# 18. Initial blockers

`InitialBlockers` is the number of vehicles directly occupying cells between the target's current right edge and the exit.

Count distinct vehicles, not cells.

A valid generated level should normally have:

```text
InitialBlockers >= 1
```

---

# 19. Dependency graph

Difficulty often comes from blockers that themselves cannot move until other blockers are moved.

We build an approximate dependency graph from the initial state.

## 19.1 Nodes

Each vehicle is a node.

The root is the target vehicle.

## 19.2 Direct target dependencies

For every vehicle blocking the target's exit corridor, add:

```text
Target -> Blocker
```

## 19.3 Blocker dependencies

For a blocking vehicle B, determine what displacement would remove B from the corridor or from the position preventing its parent from moving.

For each feasible direction of B:

1. determine the minimum displacement needed to clear the conflict;
2. identify vehicles occupying the cells B would need to traverse/occupy;
3. add edges:

```text
B -> blocking vehicle
```

If B can clear the conflict immediately in at least one direction, it has no required child dependency for that direction.

If multiple directions exist, choose the direction with the smallest dependency depth.

This turns the graph metric into:

> minimum blocker-chain depth required to make progress.

## 19.4 DependencyDepth

Definition:

```text
Target -> A -> B -> C
```

has:

```text
DependencyDepth = 3
```

The target itself is depth 0.

Cycles are possible. The DFS must keep a recursion stack and stop cyclic paths.

## 19.5 DependencyNodes

Count unique non-target vehicles reachable from the dependency graph root.

This distinguishes:

```text
one deep narrow chain
```

from:

```text
several interacting blocker chains
```

## 19.6 V1 limitation

The dependency graph is a structural heuristic and does not replace the solver.

It does not need to perfectly prove necessity of every dependency in v1.

Its usefulness will be validated empirically against generated levels.

---

# 20. Branching metrics

We want metrics that describe player choice, not implementation-specific BFS behavior.

For every state along one canonical optimal solution path:

1. generate all legal moves;
2. record move count.

## 20.1 AverageBranching

```text
sum(legalMovesAtState) / numberOfSolutionStates
```

Do not include the terminal solved state.

## 20.2 MaxBranching

Maximum number of legal moves at any state on the canonical optimal path.

---

# 21. Progressing and distracting moves

A better metric than raw branching is how many legal moves actually preserve optimal progress.

## 21.1 Required distance information

For a state `S` at depth `d` along an optimal solution of total length `N`, a move to `T` is an optimal-progress move if:

```text
distanceToGoal(T) == N - d - 1
```

Exact reverse distances for arbitrary states are expensive because the goal is a set of states, not a single state.

For v1 use bounded memoized solves from neighboring states.

Because analysis runs offline and only for accepted/promising candidates, this is acceptable.

Optimization:

- compute these metrics only for the canonical optimal path;
- cache `StateKey -> minDistanceToGoal` inside one analysis;
- cap each sub-solve at the known remaining optimal distance + a small margin.

## 21.2 DistractingRatio

For each state on the canonical optimal path:

```text
distractingMoves = legalMoves - progressingMoves
```

Aggregate:

```text
DistractingRatio = totalDistractingMoves / totalLegalMoves
```

If no legal moves exist, use zero.

A high value means the player sees many plausible but non-progressing choices.

## 21.3 ForcedMoveRatio

For each canonical optimal-path state, determine how many legal moves preserve an optimal solution.

A state is "forced" if exactly one legal move is optimal-progressing.

```text
ForcedMoveRatio = forcedStates / solutionStates
```

High forcedness is not inherently easy or hard; it is kept as an independent feature.

---

# 22. Reachable state count

`ReachableStates` is the number of unique states discovered by bounded exploration from the initial state.

Preferred configuration:

```text
explore through min(OptimalMoves + 4, configuredMaxDepth)
```

The metric is a proxy for local state-space complexity.

Do not use `VisitedStates` from `Solve` directly as a difficulty metric because it depends on move ordering and solver termination behavior.

---

# 23. Difficulty score v1

The first formula is intentionally simple and inspectable.

Normalize each metric into a roughly 0..1 range:

```text
movesN       = clamp((OptimalMoves      - 4) / 26, 0, 1)
depthN       = clamp(DependencyDepth       / 5, 0, 1)
regressionN  = clamp(TargetRegressions     / 3, 0, 1)
dirChangeN   = clamp(DirectionChanges      / 8, 0, 1)
distractN    = clamp(DistractingRatio,          0, 1)
spaceN       = clamp(log2(ReachableStates+1) / 16, 0, 1)
revisitN     = clamp(VehicleRevisits       / 10, 0, 1)
```

Then:

```text
DifficultyScore = 100 * (
    0.30 * movesN      +
    0.20 * depthN      +
    0.12 * regressionN +
    0.10 * dirChangeN  +
    0.12 * distractN   +
    0.10 * spaceN      +
    0.06 * revisitN
)
```

The weights are provisional and must remain configuration values.

The raw metrics are authoritative; the score can be recalculated later without regenerating levels.

## 23.1 Playable generation bands

```text
0  <= score < 25  below the generated difficulty scale
25 <= score < 50  Easy   (former Medium)
50 <= score < 75  Medium (former Hard)
75 <= score < 90  Hard   (former Expert)
90 <= score <=100 Expert
```

The analyzer may still classify a sub-25 score as Easy for display, but the
generator never accepts it. This keeps the user-facing Easy → Hard ramp in
the former Medium → Expert range without adding another playable tier.

Do not add an "Insane" tier in v1; first collect real distributions.

---

# 24. Quality analysis

Difficulty and quality are separate.

A difficult level may still be tedious, noisy, repetitive, or nearly duplicated.

## 24.1 QualityScore

Start at 100 and subtract penalties.

Suggested v1 penalties:

```text
-100 if unsolvable
-100 if initially solved
-40  if InitialBlockers == 0
-25  if ParticipationRatio < tier minimum
-15  if OptimalSolutions >= 1000 (capped overflow)
-15  if OptimalMoves < tier minimum
-10  if DependencyDepth == 0
-10  if VehiclesMoved < 4
-10  if too similar to an existing accepted level
```

Default tier minimums used by the analyzer:

```text
             ParticipationRatio   OptimalMoves
Easy         0.50                 4
Medium       0.60                 10
Hard         0.60                 15
Expert       0.60                 20
```

`analyzer.Config` may override the participation and move minimums for
calibration runs.

Clamp to 0..100.

Initial acceptance threshold:

```text
QualityScore >= 70
```

Hard rejection rules such as unsolvable/duplicate are also listed in `RejectReasons`.

---

# 25. Duplicate detection

## 25.1 Exact canonical representation

Vehicle IDs are not semantically important.

Canonicalize by sorting non-target vehicles using:

```text
orientation
fixed
length
initial position
```

The target remains vehicle 0 in canonical output.

Reassign IDs after sorting.

Serialize the compact canonical representation and hash it, for example with SHA-256.

Exact hash equality means duplicate.

## 25.2 Mirror symmetry

For MVP with right-side exit, horizontal mirroring changes the exit side and is not equivalent under current rules.

Vertical mirroring preserves a right-side exit but changes the target lane.

V1 exact deduplication does not merge mirrored boards.

A later similarity layer may do so if desired.

## 25.3 Near-duplicate detection

Not required for first implementation.

Later options:

- normalized vehicle-feature edit distance;
- board occupancy Hamming distance;
- solution-signature comparison;
- canonical structural fingerprints.

---

# 26. Generator architecture

Generation has two layers:

1. create a solvable candidate;
2. search toward requested difficulty/quality.

```text
requested tier
    ↓
random solved layout
    ↓
reverse scramble
    ↓
solve
    ↓
analyze
    ↓
accept / mutate / reject
```

---

# 27. Generator configuration

```go
type DifficultyTier string

const (
    Easy   DifficultyTier = "easy"
    Medium DifficultyTier = "medium"
    Hard   DifficultyTier = "hard"
    Expert DifficultyTier = "expert"
)

type Config struct {
    Tier DifficultyTier

    MinVehicles int
    MaxVehicles int

    ScrambleMin int
    ScrambleMax int

    MaxAttempts  int
    MaxMutations int

    Seed int64
}
```

Initial ranges:

```text
Easy:
  vehicles 9..12
  scramble 20..45

Medium:
  vehicles 11..15
  scramble 35..70

Hard:
  vehicles 13..18
  scramble 50..100

Expert:
  vehicles 13..18
  scramble 50..100
```

These are search heuristics only, not difficulty definitions.

Easy, Hard, and Expert cap a single seed at 100 layout attempts. Medium keeps
1000 attempts because valid Medium levels may appear later in the mutation
search; its tier-specific move floor makes rejected attempts cheap. Bulk
generation tries independent seeds until the requested count is reached,
bounded at 3× the requested count for Easy/Medium and 80× for Hard/Expert.
The larger Hard budget reflects its intentionally narrow, empirically rarer
score band; unused retry capacity adds no work after the requested count is
reached.

---

# 28. Layout generation

The generator first produces a geometric vehicle layout.

## 28.1 Target

Create target first:

```text
orientation = horizontal
length = 2
fixed row = random reasonable row
```

Recommended initial target rows:

```text
1..4
```

Avoid extreme top/bottom lanes in v1 unless diversity proves insufficient.

## 28.2 Other vehicles

For each additional vehicle:

1. choose orientation;
2. choose length 2 or 3;
3. choose fixed lane;
4. choose a placement that does not overlap the current solved-state layout.

Suggested probabilities:

```text
length 2: 75%
length 3: 25%
```

Orientation can begin at 50/50.

## 28.3 Solved-state placement

The target must have a clear exit corridor.

Other vehicles may be anywhere that does not overlap.

Generation may retry vehicle placement a bounded number of times. If a layout cannot be completed, discard the whole layout and start again.

---

# 29. Reverse scramble

Starting from the solved state, perform legal moves to reach a candidate state.

Because every vehicle move is reversible, the candidate is guaranteed to have a path back to a solved state.

## 29.1 Rules

Maintain:

```text
seenStates
previousMove
```

At each step:

1. generate legal moves;
2. remove moves that exactly undo `previousMove` when alternatives exist;
3. prefer moves leading to states not in `seenStates`;
4. select one using seeded randomness;
5. apply it;
6. mark state seen.

If every move returns to seen states, allow a seen state rather than aborting immediately.

## 29.2 Biased scramble

Pure random walk often returns shallow puzzles.

Use a configurable bias:

```text
70% random move
30% heuristic move
```

A heuristic move receives bonuses when the resulting state:

- increases the number of target blockers;
- increases approximate dependency depth;
- moves the target away from the exit;
- introduces a new vehicle into the target dependency graph.

Do not run full analysis during every scramble step. Use cheap local heuristics only.

## 29.3 Final candidate requirements

After scramble:

- candidate must not be solved;
- target must be blocked;
- candidate must differ from solved state;
- candidate must pass structural validation.

Then run the real solver.

---

# 30. Candidate evaluation

For every candidate:

1. run `Solve`;
2. reject if unsolved or limits reached;
3. reject trivial solutions below configured floor;
4. run `Analyze`;
5. compute distance from target tier;
6. accept, mutate, or reject.

---

# 31. Target-score distance

Each tier has a preferred score center:

```text
Easy   37.5
Medium 62.5
Hard   82.5
Expert 95
```

Define:

```text
fitness = abs(DifficultyScore - tierCenter)
```

Lower is better.

Quality penalties are added heavily:

```text
fitness += max(0, 70 - QualityScore) * 3
```

A hard-rejected level has infinite fitness.

---

# 32. Mutation search

If a candidate is valid but outside the target score range, attempt hill-climbing mutations before discarding it.

## 32.1 Safe v1 mutations

Start with mutations that preserve the vehicle set/layout definition and only change positions:

- move one non-target vehicle to another legal position;
- move the target to another legal position on its row;
- perform a short random legal walk of 1..5 moves.

Every mutation is followed by:

```text
validate → solve → analyze
```

## 32.2 Structural mutations

After position-only mutation works reliably, add:

- change vehicle length 2 ↔ 3 if geometry remains valid;
- relocate fixed lane;
- rotate a non-target vehicle;
- add a vehicle;
- remove a low-participation vehicle.

These are Phase 5, not initial MVP.

## 32.3 Hill-climbing acceptance

Track the best candidate.

Accept a mutation if:

```text
newFitness < currentFitness
```

To avoid local minima, optionally allow a small probability such as 5% of accepting a worse but valid mutation.

This is enough for v1; do not implement full simulated annealing yet.

---

# 33. Generator API

```go
type GeneratedLevel struct {
    Level    puzzle.Level
    Analysis analyzer.Analysis
    Solution []puzzle.Move
    Seed     int64
}

func Generate(cfg Config) (*GeneratedLevel, error)
```

Generation errors should distinguish:

```text
invalid configuration
attempt limit reached
solver limit repeatedly reached
no acceptable candidate found
```

---

# 34. Bulk generation

```go
type BulkConfig struct {
    Generator Config
    Count     int
    Workers   int
}
```

Use a bounded worker pool.

For job `i`, derive a deterministic sub-seed from root seed and job number.

Do not share `rand.Rand` between goroutines.

Each worker owns its RNG.

Collector responsibilities:

- exact duplicate filtering;
- assigning final level IDs;
- writing output;
- aggregate statistics.

---

# 35. JSON format

Keep game data and generated metadata separate enough that the mobile client may ignore analysis.

Example:

```json
{
  "version": 1,
  "id": "hard-000123",
  "board": {
    "width": 6,
    "height": 6,
    "exit": "right"
  },
  "target": 0,
  "vehicles": [
    {
      "id": 0,
      "orientation": "horizontal",
      "length": 2,
      "fixed": 2,
      "position": 1
    },
    {
      "id": 1,
      "orientation": "vertical",
      "length": 3,
      "fixed": 4,
      "position": 0
    }
  ],
  "analysis": {
    "difficultyScore": 63.4,
    "difficultyTier": "hard",
    "qualityScore": 88,
    "metrics": {
      "optimalMoves": 18,
      "totalCellDistance": 27,
      "vehiclesMoved": 8,
      "vehicleRevisits": 4,
      "directionChanges": 3,
      "targetRegressions": 1,
      "initialBlockers": 2,
      "dependencyDepth": 3,
      "dependencyNodes": 6,
      "averageBranching": 8.2,
      "maxBranching": 13,
      "distractingRatio": 0.71,
      "forcedMoveRatio": 0.33,
      "reachableStates": 4831,
      "optimalSolutions": 2,
      "participationRatio": 0.67
    }
  },
  "solution": [
    {"vehicle": 4, "from": 1, "to": 3},
    {"vehicle": 2, "from": 2, "to": 0}
  ],
  "generation": {
    "seed": 874218841
  }
}
```

The CLI should support omitting `analysis`, `solution`, or `generation` when producing production/mobile assets.

---

# 36. ASCII renderer

A simple renderer is required before any graphical UI.

Example:

```text
+------+
|AA.BCC|
|...B..|
|TT.BDD> EXIT
|EE....|
|..FFF.|
|GG....|
+------+
```

Requirements:

- target displayed distinctly, e.g. `T`;
- other vehicles assigned deterministic characters;
- exit marker visible;
- optional move highlighting later.

CLI commands should be able to print every solution state for debugging.

---

# 37. CLI

Binary name:

```text
parking
```

## 37.1 Generate one level

```bash
parking generate \
  --difficulty hard \
  --seed 12345
```

## 37.2 Generate a dataset

```bash
parking generate \
  --difficulty hard \
  --count 1000 \
  --workers 8 \
  --seed 12345 \
  --output levels-hard.json
```

## 37.3 Solve

```bash
parking solve level.json
```

Output:

```text
Solved: yes
Optimal moves: 18
Visited states: 4217
Expanded states: 3892

1. vehicle 4: 1 -> 3
2. vehicle 2: 2 -> 0
...
```

Optional:

```bash
parking solve level.json --show-boards
```

## 37.4 Analyze

```bash
parking analyze level.json
```

Output all raw metrics plus score/tier/rejection reasons.

## 37.5 Render

```bash
parking render level.json
```

---

# 38. Performance goals

These are engineering targets, not hard requirements.

On a modern desktop CPU:

- ordinary level solve: preferably < 100 ms;
- hard level solve: preferably < 1 s;
- no single accepted level should normally require > 2,000,000 states;
- bulk generator should scale reasonably with CPU cores.

Do not optimize prematurely.

First benchmark real generated state-space sizes.

---

# 39. Memory considerations

A Go `map[uint64]parentInfo` has significant overhead.

Start with it for correctness and simplicity.

Only if profiling shows memory pressure, consider:

- custom open-addressing hash table;
- separate compact arrays;
- bidirectional search;
- integer node IDs plus parent arrays.

These are explicitly outside initial implementation.

---

# 40. Testing strategy

## 40.1 `puzzle` tests

Test:

- state encode/decode round-trip;
- single coordinate updates;
- invalid vehicle length;
- invalid fixed coordinate;
- overlap detection;
- target validation;
- vehicle-count limit.

## 40.2 `engine` tests

Use tiny hand-authored boards with known moves.

Test:

- occupancy masks;
- horizontal movement boundaries;
- vertical movement boundaries;
- blocking by another vehicle;
- multiple-distance moves;
- deterministic move order;
- applying moves;
- solved predicate.

## 40.3 `solver` tests

Create fixtures with known optimal depths:

```text
0 moves
1 move
2 moves
several alternative optimal solutions
unsolvable fixture
```

Assert:

- solver finds optimal length;
- returned moves are legal;
- applying solution reaches solved state;
- limits work;
- repeated runs return the same canonical solution.

## 40.4 `analyzer` tests

Use deliberately constructed levels where one metric is obvious.

Examples:

- target directly blocked by one freely movable vehicle → dependency depth 1;
- blocker blocked by another → depth 2;
- target must move left first → target regression >= 1;
- same vehicle moved in both directions → direction change >= 1;
- unused decorative vehicle → participation ratio decreases.

Difficulty score tests should verify formula stability, not subjective truth.

## 40.5 `generator` tests

For fixed seeds:

- generation is reproducible;
- generated level validates;
- solver always solves accepted levels;
- accepted level is not initially solved;
- accepted tier score lies in requested range;
- generated exact duplicates are rejected.

Avoid asserting one exact generated board unless the test specifically protects determinism; otherwise minor algorithm improvements would make tests brittle.

## 40.6 Property tests / fuzzing

Go fuzz tests are especially useful for:

- state encoding;
- `ApplyMove` after generated legal moves;
- occupancy invariants;
- generated-move reversibility;
- validation never panics on malformed input.

Important invariant:

> For every legal move A → B generated from a valid state, a reverse move B → A for the same vehicle must be legal in the resulting state.

---

# 41. Benchmarks

Add Go benchmarks for:

```text
EncodeState
Occupancy
GenerateMoves
ApplyMove
Solve/easy
Solve/hard
Analyze/hard
```

Do not optimize based on microbenchmarks alone; solver-level throughput is primary.

---

# 42. Logging

Generation logs should include enough information to reproduce failures:

```text
root seed
job seed
attempt number
mutation number
solver limits
candidate score
reject reason
```

Example:

```text
WARN candidate rejected seed=874218841 attempt=42 reason=solver_limit states=2000000
```

Bulk generation should not print every rejected candidate by default.

Suggested levels:

```text
quiet
normal
verbose
trace
```

---

# 43. Generation statistics

Bulk generation should print or optionally export:

```text
candidates attempted
solver failures
solver limit hits
quality rejections
duplicate rejections
accepted levels
average attempts per accepted level
average solve time
score histogram
optimal-move histogram
dependency-depth histogram
```

This will be important for tuning difficulty weights.

---

# 44. Calibration workflow

The initial difficulty score is heuristic.

Expected development loop:

```text
generate 1000+ levels
        ↓
sample levels across score buckets
        ↓
human play/review
        ↓
record perceived difficulty
        ↓
change weights / thresholds
        ↓
recompute scores from stored raw metrics
```

Because raw analysis metrics are persisted, changing score weights must not require generating levels again.

Later, if real player telemetry becomes available, a statistical or ML model may replace the manual formula without changing generator/solver architecture.

---

# 45. Implementation phases

## Phase 1 — Model + engine

Deliverables:

- `Level`, `Vehicle`, `StateKey`, `Move`;
- validation;
- mask preparation;
- occupancy;
- move generation;
- move application;
- solved predicate;
- ASCII renderer;
- unit/fuzz tests.

Exit criterion:

> hand-authored levels can be loaded, rendered, and legally manipulated.

## Phase 2 — Optimal solver

Deliverables:

- BFS;
- limits;
- optimal path reconstruction;
- basic search statistics;
- `parking solve`;
- solver tests and benchmarks.

Exit criterion:

> known fixtures return verified minimum-move solutions.

## Phase 3 — Basic solvable generator

Deliverables:

- seeded RNG;
- solved layout generation;
- reverse scramble;
- basic candidate filters;
- solver verification;
- `parking generate` for one or many levels.

Exit criterion:

> generator reliably creates diverse solvable nontrivial levels.

## Phase 4 — Analyzer

Deliverables:

- solution metrics;
- dependency graph heuristic;
- branching metrics;
- progressing/distracting analysis;
- reachable-state exploration;
- difficulty score;
- quality score;
- `parking analyze`.

Exit criterion:

> every generated level receives inspectable raw metrics and a stable score.

## Phase 5 — Difficulty-targeted generation

Deliverables:

- tier configurations;
- position mutations;
- hill climbing;
- exact canonical deduplication;
- quality gates.

Exit criterion:

> requesting Easy/Medium/Hard/Expert produces materially different metric distributions.

## Phase 6 — Bulk generation and tuning

Deliverables:

- worker pool;
- deterministic job seeds;
- dataset export;
- histograms/statistics;
- performance profiling;
- difficulty calibration workflow.

Exit criterion:

> tens of thousands of candidate levels can be processed offline with reproducible metadata.

---

# 46. Acceptance criteria for v1

The project is considered v1-complete when:

1. a valid JSON level can be loaded and rendered;
2. all legal moves can be enumerated deterministically;
3. BFS returns a verified optimal solution;
4. generator creates guaranteed-solvable candidates from solved states;
5. analyzer reports all required raw metrics;
6. score places levels into four difficulty tiers;
7. generator can target each tier through accept/reject + mutation;
8. exact duplicate levels are rejected;
9. generation is reproducible from seeds;
10. bulk generation can use multiple CPU cores;
11. accepted generated levels pass solver verification before export;
12. test coverage includes engine invariants, solver optimality, analyzer metrics, and generator determinism.

---

# 47. Open questions to validate experimentally

These should not block implementation.

1. Is 6×6 sufficient for the intended mobile game, or will 7×7 be useful later?
2. Does "one vehicle slide of any distance = one move" match the intended product UX?
3. How strongly does dependency depth correlate with perceived difficulty?
4. Is exact shortest-solution count useful enough to justify its exploration cost?
5. Does `DistractingRatio` correlate with difficulty or merely visual noise?
6. Should target regressions receive a larger weight?
7. Are decorative vehicles actually undesirable, or useful for visual complexity?
8. What score ranges produce good Easy/Medium/Hard/Expert distributions after real playtesting?

The architecture intentionally stores raw metrics so these answers can change without redesigning the core system.

---

# 48. Recommended first implementation order

Implement in this exact order:

```text
1. Level + StateKey
2. validation
3. board masks
4. GenerateMoves
5. ApplyMove
6. IsSolved
7. ASCII renderer
8. BFS solver
9. JSON I/O
10. CLI solve/render
11. reverse-scramble generator
12. basic generate CLI
13. solution metrics
14. dependency metric
15. branching metrics
16. difficulty / quality scores
17. mutation search
18. deduplication
19. bulk worker pool
20. calibration statistics
```

The first meaningful milestone is step 8: once a correct optimal solver exists, almost every later feature can be built and verified against it.

---

# 49. Web deployment: level packs, mobile, Telegram Mini App

Status: Draft v1. Covers `public/` (the player-facing web app) and the
`generate-packs` CLI command, layered on top of the engine described above.
The web app has no backend: it is static files served over HTTPS, consumed
directly by a browser or by Telegram's in-app WebView.

## 49.1 Level packs

Levels ship as packs of up to 100, written under `public/levels/`:

```text
public/levels/
  manifest.json       # [{id, file, tier, count, index}, ...]
  pack-001.json        # []levelio.File, one pack's levels
  pack-002.json
  ...
```

JSON is the only generated data format. The web app and CLI both consume the
same files, avoiding duplicate artifacts that can drift apart.

### 49.1.1 `generate-packs` CLI

```text
parking generate-packs [--packs N] [--pack-size N] [--seed N] [--workers N] [--out dir]
```

Defaults: `--packs 6 --pack-size 100 --seed 1 --workers 4 --out public/levels`.

Difficulty ramps smoothly across the requested pack count: packs are split
into thirds and assigned Easy → Medium → Hard in order (e.g. 6 packs ⇒ 2
Easy, 2 Medium, 2 Hard). These tiers now cover the former Medium → Expert
range. Expert is reserved for scores of 90 and above and is not included in
the default ramp.

If `generator.Bulk` returns an error for a given pack (e.g. a tier that
cannot produce any accepted level), that pack is skipped with a warning
printed to stderr — the run continues rather than aborting the whole batch,
and the manifest only lists packs that were actually written.

While bulk generation is running, the CLI prints a progress snapshot at most
once every five seconds. It includes accepted and processed counts, the
maximum job budget, failures, duplicates, elapsed time, and an ETA estimated
from the observed acceptance rate. The estimate is expected to fluctuate at
the start of a pack.

### 49.1.2 Regenerating content

`scripts/generate-levels.sh` wraps the CLI call and leaves generated files
uncommitted for review. Override defaults via env vars (`PACKS`, `PACK_SIZE`,
`SEED`, `WORKERS`) or CLI flags passed through.

## 49.2 Web app pack switching

`public/app.js` fetches `levels/manifest.json` on load, populates the
`#pack-select` dropdown in the level panel, and restores the last-selected
pack from `localStorage` (`parking-puzzle-pack-v1`). Selecting a pack fetches
`levels/<pack.file>` and resets to level 1 of that pack. Per-level progress
(`parking-puzzle-progress-v2:<packId>`) is scoped per pack, so completion
state doesn't bleed between packs.

Because packs are fetched as JSON, the app must be served over HTTP(S); direct
`file://` opening is not supported.

## 49.3 Mobile layout

The existing canvas board and CSS grid layout already had one mobile
breakpoint (`@media (max-width:680px)`, stacking panels). This phase fixed
two pre-existing CSS bugs while extending it for small screens and
Telegram's WebView:

- `env(safe-area-inset-*)` padding on `.app-shell`, so content clears the
  notch/home-indicator area and Telegram's own chrome.
- `body { min-height: var(--tg-viewport-height, 100dvh) }`, driven by the
  Telegram viewport handling below; falls back to `100dvh` outside Telegram.
- the canvas is sized in CSS via `--cell`, which already scales with
  viewport width — no further canvas-specific change was needed.

## 49.4 Telegram Mini App integration

`index.html` loads `https://telegram.org/js/telegram-web-app.js`. On boot,
`app.js`'s `initTelegram()`:

- calls `WebApp.ready()` and `WebApp.expand()`;
- calls `WebApp.disableVerticalSwipes()` when available, so dragging a
  vertical vehicle near the top/bottom of the board doesn't trigger
  Telegram's swipe-to-minimize gesture;
- maps `WebApp.themeParams` (`bg_color`, `secondary_bg_color`, `text_color`,
  `hint_color`, `button_color`) onto the existing `--paper`/`--panel`/
  `--ink`/`--muted`/`--accent` CSS custom properties, re-applied on the
  `themeChanged` event;
- mirrors `viewportStableHeight`/`viewportHeight` into the
  `--tg-viewport-height` CSS var, re-applied on `viewportChanged`.

Outside Telegram (`window.Telegram` absent, e.g. plain GitHub Pages or
local dev), `initTelegram()` is a no-op and the app runs as a normal site.

### 49.4.1 Registering the Mini App (manual, one-time, outside this repo)

1. Deploy `public/` to GitHub Pages (Settings → Pages → serve from the
   branch/folder holding `public/`, or a workflow that copies it there).
   No build step is required — the JS/CSS/HTML are plain static files.
2. In Telegram, talk to **@BotFather**: create a bot (`/newbot`) if one
   doesn't exist yet, then `/newapp` (or `/setmenubutton` for an existing
   bot) and supply the GitHub Pages HTTPS URL as the Web App URL.
3. Telegram Mini Apps require HTTPS; GitHub Pages provides this by default,
   so no additional TLS setup is needed.

This step cannot be automated from the repo — it requires interactive
BotFather conversation and a Telegram account.
