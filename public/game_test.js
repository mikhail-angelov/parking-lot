const test = require("node:test");
const assert = require("node:assert/strict");

const { createGame } = require("./game.js");

function level() {
  return {
    board: { width: 6, height: 6 },
    target: 0,
    vehicles: [
      { orientation: "horizontal", length: 2, fixed: 0, position: 0 },
      { orientation: "vertical", length: 2, fixed: 2, position: 0 },
    ],
  };
}

test("a vehicle cannot jump over another vehicle", () => {
  const game = createGame(level());

  assert.equal(game.canMove(0, 3), false);
  assert.equal(game.move(0, 3), false);
  assert.deepEqual(game.positions, [0, 0]);
  assert.equal(game.moves, 0);
});

test("move, solved state, undo, and reset share one transition model", () => {
  const puzzle = {
    board: { width: 6, height: 6 },
    target: 0,
    vehicles: [
      { orientation: "horizontal", length: 2, fixed: 2, position: 0 },
      { orientation: "vertical", length: 2, fixed: 2, position: 2 },
    ],
  };
  const game = createGame(puzzle);

  assert.equal(game.move(1, 0), true);
  assert.equal(game.moves, 1);
  assert.equal(game.isSolved(), true);
  assert.equal(game.undo(), true);
  assert.equal(game.isSolved(), false);
  assert.deepEqual(game.positions, [0, 2]);
  game.move(1, 3);
  game.reset();
  assert.equal(game.moves, 0);
  assert.equal(game.canUndo, false);
  assert.deepEqual(game.positions, [0, 2]);
});
