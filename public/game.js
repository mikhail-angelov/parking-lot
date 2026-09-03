(function (root, factory) {
  const game = factory();
  if (typeof module === "object" && module.exports) module.exports = game;
  root.ParkingGame = game;
})(typeof globalThis !== "undefined" ? globalThis : this, function () {
  function vehicleCells(vehicle, position) {
    const cells = [];
    for (let n = 0; n < vehicle.length; n++) {
      cells.push(
        vehicle.orientation === "horizontal"
          ? [position + n, vehicle.fixed]
          : [vehicle.fixed, position + n],
      );
    }
    return cells;
  }

  function createGame(level) {
    const initial = level.vehicles.map((vehicle) => vehicle.position);
    let positions = [...initial];
    let history = [];
    let moves = 0;

    function occupied(except = -1) {
      const result = new Set();
      level.vehicles.forEach((vehicle, index) => {
        if (index === except) return;
        vehicleCells(vehicle, positions[index]).forEach(([x, y]) =>
          result.add(`${x},${y}`),
        );
      });
      return result;
    }

    function canOccupy(index, position) {
      const vehicle = level.vehicles[index];
      if (!vehicle || !Number.isInteger(position)) return false;
      const limit =
        vehicle.orientation === "horizontal"
          ? level.board.width
          : level.board.height;
      if (position < 0 || position + vehicle.length > limit) return false;
      const used = occupied(index);
      return vehicleCells(vehicle, position).every(
        ([x, y]) => !used.has(`${x},${y}`),
      );
    }

    function canMove(index, position) {
      const from = positions[index];
      if (from === undefined || position === from) return false;
      const step = position > from ? 1 : -1;
      for (let current = from + step; ; current += step) {
        if (!canOccupy(index, current)) return false;
        if (current === position) return true;
      }
    }

    function move(index, position) {
      if (!canMove(index, position)) return false;
      history.push([...positions]);
      positions[index] = position;
      moves++;
      return true;
    }

    function undo() {
      const previous = history.pop();
      if (!previous) return false;
      positions = previous;
      moves--;
      return true;
    }

    function reset() {
      positions = [...initial];
      history = [];
      moves = 0;
    }

    function isSolved() {
      const target = level.vehicles[level.target];
      const used = occupied(level.target);
      for (
        let x = positions[level.target] + target.length;
        x < level.board.width;
        x++
      ) {
        if (used.has(`${x},${target.fixed}`)) return false;
      }
      return true;
    }

    return {
      canMove,
      move,
      undo,
      reset,
      isSolved,
      get positions() {
        return [...positions];
      },
      get moves() {
        return moves;
      },
      get canUndo() {
        return history.length > 0;
      },
    };
  }

  return { createGame, vehicleCells };
});
