(() => {
  const packKey = "parking-puzzle-pack-v1";
  const tierLabels = {
    easy: "лёгкая",
    medium: "средняя",
    hard: "сложная",
    expert: "эксперт",
  };
  const state = {
    packs: [],
    packId: null,
    levels: [],
    index: 0,
    game: null,
    selected: -1,
    drag: null,
  };
  let packRequest = 0;
  const $ = (id) => document.getElementById(id);
  const canvas = $("board");
  const ctx = canvas.getContext("2d");

  async function loadManifest() {
    const response = await fetch("levels/manifest.json", { cache: "no-store" });
    if (!response.ok)
      throw new Error(`Не удалось загрузить список пачек (${response.status})`);
    return response.json();
  }

  async function loadPackLevels(pack) {
    const response = await fetch(`levels/${pack.file}`, { cache: "no-store" });
    if (!response.ok)
      throw new Error(`Не удалось загрузить уровни (${response.status})`);
    return response.json();
  }

  const progressKey = () => `parking-puzzle-progress-v2:${state.packId}`;
  const level = () => state.levels[state.index];

  function resize() {
    const rect = canvas.getBoundingClientRect();
    const border = parseFloat(getComputedStyle(canvas).borderLeftWidth) || 0;
    const width = rect.width - border * 2;
    const dpr = window.devicePixelRatio || 1;
    canvas.width = Math.round(width * dpr);
    canvas.height = Math.round(width * dpr);
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    draw();
  }

  function roundedRect(x, y, width, height, radius) {
    ctx.beginPath();
    ctx.roundRect(x, y, width, height, radius);
  }

  function draw() {
    const rect = canvas.getBoundingClientRect();
    const border = parseFloat(getComputedStyle(canvas).borderLeftWidth) || 0;
    const width = rect.width - border * 2;
    const height = rect.height - border * 2;
    const cell = width / 6;
    ctx.clearRect(0, 0, width, height);
    ctx.fillStyle = "#dce6e2";
    ctx.fillRect(0, 0, width, height);
    ctx.strokeStyle = "rgba(255,255,255,.35)";
    ctx.lineWidth = 1;
    for (let index = 1; index < 6; index++) {
      ctx.beginPath();
      ctx.moveTo(index * cell, 0);
      ctx.lineTo(index * cell, height);
      ctx.moveTo(0, index * cell);
      ctx.lineTo(width, index * cell);
      ctx.stroke();
    }
    if (!level()) return;

    const positions = state.game.positions;
    level().vehicles.forEach((vehicle, index) => {
      let position = positions[index];
      if (state.drag?.index === index) position = state.drag.position;
      const x =
        (vehicle.orientation === "horizontal" ? position : vehicle.fixed) *
          cell +
        4;
      const y =
        (vehicle.orientation === "horizontal" ? vehicle.fixed : position) *
          cell +
        4;
      const vehicleWidth =
        (vehicle.orientation === "horizontal" ? vehicle.length : 1) * cell - 8;
      const vehicleHeight =
        (vehicle.orientation === "vertical" ? vehicle.length : 1) * cell - 8;
      ctx.save();
      roundedRect(x, y, vehicleWidth, vehicleHeight, 10);
      const gradient = ctx.createLinearGradient(
        x,
        y,
        x + vehicleWidth,
        y + vehicleHeight,
      );
      if (index === level().target) {
        gradient.addColorStop(0, "#ff8877");
        gradient.addColorStop(1, "#d8483e");
      } else {
        gradient.addColorStop(0, "#557277");
        gradient.addColorStop(1, "#293f43");
      }
      ctx.fillStyle = gradient;
      ctx.shadowColor =
        index === level().target ? "rgba(201,68,59,.35)" : "rgba(20,40,42,.28)";
      ctx.shadowBlur = 8;
      ctx.shadowOffsetY = 4;
      ctx.fill();
      ctx.shadowColor = "transparent";
      ctx.strokeStyle =
        index === state.selected ? "#f6c85f" : "rgba(255,255,255,.16)";
      ctx.lineWidth = index === state.selected ? 3 : 1;
      ctx.stroke();
      ctx.fillStyle = "rgba(210,240,236,.62)";
      const windowWidth =
        vehicle.orientation === "horizontal"
          ? vehicleWidth * 0.42
          : vehicleWidth * 0.28;
      const windowHeight =
        vehicle.orientation === "horizontal"
          ? vehicleHeight * 0.28
          : vehicleHeight * 0.42;
      roundedRect(
        x + (vehicleWidth - windowWidth) / 2,
        y + (vehicleHeight - windowHeight) / 2,
        windowWidth,
        windowHeight,
        3,
      );
      ctx.fill();
      ctx.fillStyle = "#ffffffaa";
      ctx.font = "700 12px system-ui";
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.fillText(
        index === level().target ? "T" : String.fromCharCode(65 + index),
        x + vehicleWidth / 2,
        y + vehicleHeight / 2,
      );
      ctx.restore();
    });

    const target = level().vehicles[level().target];
    const exitY = (target.fixed + 0.5) * cell;
    ctx.fillStyle = "#c9443b";
    ctx.font = "900 14px system-ui";
    ctx.textAlign = "right";
    ctx.textBaseline = "middle";
    ctx.fillText("→", width - 3, exitY);
  }

  function completedLevelIDs() {
    try {
      const value = JSON.parse(localStorage.getItem(progressKey()) || "[]");
      return Array.isArray(value) ? value : [];
    } catch {
      return [];
    }
  }

  function isComplete(index) {
    return completedLevelIDs().includes(state.levels[index].id);
  }

  function markComplete() {
    const completed = completedLevelIDs();
    const id = level().id;
    if (!completed.includes(id)) completed.push(id);
    localStorage.setItem(progressKey(), JSON.stringify(completed));
  }

  function renderLevels() {
    const list = $("level-list");
    list.replaceChildren();
    state.levels.forEach((_item, index) => {
      const done = isComplete(index);
      const button = document.createElement("button");
      const number = document.createElement("span");
      const check = document.createElement("span");
      button.className = `level-button ${done ? "done" : ""}`;
      number.textContent = index + 1;
      check.className = "check";
      check.textContent = done ? "✓" : "";
      button.append(number, check);
      button.addEventListener("click", () => loadLevel(index));
      list.append(button);
    });
  }

  function render() {
    draw();
    $("moves").textContent = state.game.moves;
    $("level-count").textContent = `Уровень ${state.index + 1}`;
    $("level-status").textContent =
      `${state.index + 1} / ${state.levels.length}`;
    $("level-title").textContent = level().id || `Парковка ${state.index + 1}`;
    $("tier-label").textContent = level().analysis?.difficultyTier || "уровень";
    document
      .querySelectorAll(".level-button")
      .forEach((button, index) =>
        button.classList.toggle("active", index === state.index),
      );
    $("undo").disabled = !state.game.canUndo;
    $("next").disabled =
      state.index >= state.levels.length - 1 || !isComplete(state.index);
  }

  function pointerPosition(event) {
    const rect = canvas.getBoundingClientRect();
    const border = parseFloat(getComputedStyle(canvas).borderLeftWidth) || 0;
    const cell = (rect.width - border * 2) / 6;
    return [
      (event.clientX - rect.left - border) / cell,
      (event.clientY - rect.top - border) / cell,
    ];
  }

  function hitTest(x, y) {
    const positions = state.game.positions;
    for (let index = level().vehicles.length - 1; index >= 0; index--) {
      const vehicle = level().vehicles[index];
      const position = positions[index];
      const left =
        vehicle.orientation === "horizontal" ? position : vehicle.fixed;
      const top =
        vehicle.orientation === "horizontal" ? vehicle.fixed : position;
      const width = vehicle.orientation === "horizontal" ? vehicle.length : 1;
      const height = vehicle.orientation === "vertical" ? vehicle.length : 1;
      if (x >= left && x <= left + width && y >= top && y <= top + height)
        return index;
    }
    return -1;
  }

  function startDrag(event) {
    const [x, y] = pointerPosition(event);
    const index = hitTest(x, y);
    if (index < 0) return;
    const position = state.game.positions[index];
    state.selected = index;
    canvas.focus();
    state.drag = {
      index,
      startX: x,
      startY: y,
      startPosition: position,
      position,
    };
    canvas.setPointerCapture(event.pointerId);
    draw();
  }

  function moveDrag(event) {
    if (!state.drag) return;
    const [x, y] = pointerPosition(event);
    const vehicle = level().vehicles[state.drag.index];
    const delta = Math.round(
      vehicle.orientation === "horizontal"
        ? x - state.drag.startX
        : y - state.drag.startY,
    );
    const candidate = state.drag.startPosition + delta;
    if (
      candidate === state.drag.startPosition ||
      state.game.canMove(state.drag.index, candidate)
    ) {
      state.drag.position = candidate;
    }
    draw();
  }

  function endDrag(event) {
    if (!state.drag) return;
    const drag = state.drag;
    state.drag = null;
    if (canvas.hasPointerCapture(event.pointerId))
      canvas.releasePointerCapture(event.pointerId);
    if (drag.position !== drag.startPosition) commit(drag.index, drag.position);
    else draw();
  }

  function commit(index, position) {
    if (!state.game.move(index, position)) return;
    render();
    if (state.game.isSolved()) win();
  }

  function keyMove(event) {
    if (state.selected < 0) return;
    const vehicle = level().vehicles[state.selected];
    const direction =
      vehicle.orientation === "horizontal"
        ? event.key === "ArrowLeft"
          ? -1
          : event.key === "ArrowRight"
            ? 1
            : 0
        : event.key === "ArrowUp"
          ? -1
          : event.key === "ArrowDown"
            ? 1
            : 0;
    if (!direction) return;
    event.preventDefault();
    commit(state.selected, state.game.positions[state.selected] + direction);
  }

  function undo() {
    if (state.game.undo()) render();
  }

  function reset() {
    state.game.reset();
    state.drag = null;
    $("win-dialog").hidden = true;
    render();
  }

  function win() {
    markComplete();
    renderLevels();
    render();
    $("win-summary").textContent =
      `Уровень пройден за ${state.game.moves} ход${state.game.moves === 1 ? "" : "ов"}.`;
    $("dialog-next").disabled = state.index >= state.levels.length - 1;
    $("win-dialog").hidden = false;
  }

  function loadLevel(index) {
    state.index = index;
    state.game = window.ParkingGame.createGame(level());
    state.selected = -1;
    state.drag = null;
    $("win-dialog").hidden = true;
    render();
  }

  function nextLevel() {
    if (state.index < state.levels.length - 1) loadLevel(state.index + 1);
  }

  function renderPackOptions() {
    const select = $("pack-select");
    select.replaceChildren();
    state.packs.forEach((pack) => {
      const option = document.createElement("option");
      option.value = pack.id;
      option.textContent = `Пачка ${pack.index} · ${tierLabels[pack.tier] || pack.tier}`;
      select.append(option);
    });
    select.value = state.packId;
  }

  async function switchPack(packId, { persist = true } = {}) {
    const request = ++packRequest;
    const pack = state.packs.find((p) => p.id === packId) || state.packs[0];
    try {
      const levels = await loadPackLevels(pack);
      if (request !== packRequest) return;
      if (!levels.length) throw new Error("Пачка уровней пуста");
      state.packId = pack.id;
      state.levels = levels;
      if (persist) localStorage.setItem(packKey, state.packId);
      clearError();
      renderPackOptions();
      renderLevels();
      loadLevel(0);
    } catch (error) {
      if (request !== packRequest) return;
      throw error;
    }
  }

  function initTelegram() {
    const tg = window.Telegram?.WebApp;
    if (!tg) return;
    tg.ready();
    tg.expand();
    tg.disableVerticalSwipes?.();
    const applyTheme = () => {
      const p = tg.themeParams || {};
      const root = document.documentElement.style;
      if (p.bg_color) root.setProperty("--paper", p.bg_color);
      if (p.secondary_bg_color)
        root.setProperty("--panel", p.secondary_bg_color);
      if (p.text_color) root.setProperty("--ink", p.text_color);
      if (p.hint_color) root.setProperty("--muted", p.hint_color);
      if (p.button_color) root.setProperty("--accent", p.button_color);
      tg.setHeaderColor?.(p.bg_color ? "bg_color" : "secondary_bg_color");
      tg.setBackgroundColor?.(p.bg_color || "#f6f8f7");
    };
    const applyViewport = () => {
      document.documentElement.style.setProperty(
        "--tg-viewport-height",
        `${tg.viewportStableHeight || tg.viewportHeight || window.innerHeight}px`,
      );
    };
    applyTheme();
    applyViewport();
    tg.onEvent("themeChanged", applyTheme);
    tg.onEvent("viewportChanged", applyViewport);
  }

  async function init() {
    initTelegram();
    try {
      state.packs = await loadManifest();
      if (!state.packs.length) throw new Error("Список пачек пуст");
      const saved = localStorage.getItem(packKey);
      const initialPack =
        state.packs.find((p) => p.id === saved) || state.packs[0];
      $("pack-select").addEventListener("change", (event) => {
        switchPack(event.target.value).catch((error) => {
          renderPackOptions();
          reportError(error);
        });
      });
      await switchPack(initialPack.id, { persist: false });
      canvas.addEventListener("pointerdown", startDrag);
      canvas.addEventListener("pointermove", moveDrag);
      canvas.addEventListener("pointerup", endDrag);
      canvas.addEventListener("pointercancel", endDrag);
      canvas.addEventListener("keydown", keyMove);
      canvas.addEventListener("click", (event) => {
        const [x, y] = pointerPosition(event);
        const hit = hitTest(x, y);
        if (hit >= 0) {
          state.selected = hit;
          draw();
        }
      });
      window.addEventListener("resize", resize);
      $("reset").addEventListener("click", reset);
      $("undo").addEventListener("click", undo);
      $("next").addEventListener("click", nextLevel);
      $("dialog-next").addEventListener("click", nextLevel);
      resize();
    } catch (error) {
      reportError(error);
    }
  }

  function reportError(error) {
    clearError();
    $("level-title").textContent = "Не удалось загрузить уровни";
    const message = document.createElement("p");
    message.id = "load-error";
    message.style.color = "#c9443b";
    message.textContent = error.message;
    canvas.after(message);
  }

  function clearError() {
    $("load-error")?.remove();
  }

  init();
})();
