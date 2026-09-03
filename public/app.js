(() => {
  const packKey = "parking-puzzle-pack-v1";
  const tierLabels = {
    easy: "лёгкая",
    medium: "средняя",
    hard: "сложная",
    expert: "эксперт",
  };
  const palette = [
    "#4f7cff",
    "#7c5cff",
    "#2bb673",
    "#f2a93b",
    "#3fb8c9",
    "#c97b3b",
    "#5aa6ff",
    "#e35fa0",
  ];
  const state = {
    packs: [],
    pack: null,
    packId: null,
    count: 0,
    levels: [],
    legacy: null,
    index: 0,
    game: null,
    selected: -1,
    drag: null,
    anim: null,
  };
  let packRequest = 0;
  const $ = (id) => document.getElementById(id);
  const canvas = $("board");
  const ctx = canvas.getContext("2d");

  function shade(hex, amount) {
    const n = parseInt(hex.slice(1), 16);
    const clamp = (v) => Math.max(0, Math.min(255, v));
    const r = clamp((n >> 16) + amount);
    const g = clamp(((n >> 8) & 0xff) + amount);
    const b = clamp((n & 0xff) + amount);
    return `#${((1 << 24) | (r << 16) | (g << 8) | b).toString(16).slice(1)}`;
  }

  function wheelPair(cx, h) {
    return `
      <g>
        <rect x="${cx - 10}" y="3" width="20" height="18" rx="7" fill="#151a1d"/>
        <rect x="${cx - 10}" y="${h - 21}" width="20" height="18" rx="7" fill="#151a1d"/>
        <rect x="${cx - 5}" y="5" width="10" height="14" rx="4" fill="#66757c"/>
        <rect x="${cx - 5}" y="${h - 19}" width="10" height="14" rx="4" fill="#66757c"/>
        <path d="M${cx - 3} 7h6 M${cx - 3} ${h - 7}h6" stroke="#aeb9bd" stroke-width="2" stroke-linecap="round"/>
      </g>`;
  }

  function buildCarSprite(length, color, isTarget) {
    const w = length * 100;
    const h = 100;
    const bodyColor = isTarget ? "#ff2f1f" : color;
    const bodyDark = isTarget ? "#a9170d" : shade(color, -38);
    const bodyMid = isTarget ? "#e92416" : shade(color, -10);
    const bodyLight = isTarget ? "#ff7669" : shade(color, 36);
    const id = `car${length}${isTarget ? "t" : color.slice(1)}`;
    const cabinW = Math.min(94, w * 0.48);
    const cabinX = w * 0.53 - cabinW / 2;
    const wheelX = [w * 0.22, w * 0.78];
    const wheels = wheelX.map((cx) => wheelPair(cx, h)).join("");
    const rearGlassX = cabinX + 8;
    const frontGlassX = cabinX + cabinW - 29;
    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${w} ${h}">
      <defs>
        <linearGradient id="${id}-body" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0" stop-color="${bodyLight}"/>
          <stop offset="0.18" stop-color="${bodyColor}"/>
          <stop offset="0.74" stop-color="${bodyMid}"/>
          <stop offset="1" stop-color="${bodyDark}"/>
        </linearGradient>
        <linearGradient id="${id}-glass" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0" stop-color="#e7fbff"/>
          <stop offset="0.42" stop-color="#8ec8d5"/>
          <stop offset="1" stop-color="#315a67"/>
        </linearGradient>
        <linearGradient id="${id}-lamp" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0" stop-color="#ffc83d"/>
          <stop offset="1" stop-color="#fff9c9"/>
        </linearGradient>
      </defs>
      ${wheels}
      <rect x="7" y="12" width="${w - 14}" height="${h - 24}" rx="28" fill="url(#${id}-body)" stroke="${bodyDark}" stroke-width="3"/>
      <path d="M19 17 Q${w / 2} 5 ${w - 25} 17" fill="none" stroke="#fff" stroke-width="4" stroke-linecap="round" opacity=".45"/>
      <path d="M18 82 Q${w / 2} 92 ${w - 24} 82" fill="none" stroke="#101a1e" stroke-width="3" stroke-linecap="round" opacity=".22"/>

      <rect x="${cabinX}" y="14" width="${cabinW}" height="72" rx="20" fill="${bodyDark}" opacity=".88"/>
      <path d="M${rearGlassX + 18} 18 H${frontGlassX - 3} V43 H${rearGlassX} Z" fill="url(#${id}-glass)" stroke="#203a43" stroke-width="2"/>
      <path d="M${rearGlassX} 57 H${frontGlassX - 3} V82 H${rearGlassX + 18} Z" fill="url(#${id}-glass)" stroke="#203a43" stroke-width="2"/>
      <path d="M${frontGlassX} 18 L${cabinX + cabinW - 8} 27 V73 L${frontGlassX} 82 Z" fill="url(#${id}-glass)" stroke="#203a43" stroke-width="2"/>
      <path d="M${rearGlassX} 18 L${rearGlassX + 16} 25 V75 L${rearGlassX} 82 Z" fill="#6ca7b5" stroke="#203a43" stroke-width="2"/>
      <rect x="${rearGlassX + 19}" y="44" width="${cabinW - 55}" height="12" rx="6" fill="${bodyColor}"/>
      <path d="M${rearGlassX + 22} 22 H${frontGlassX - 7}" stroke="#fff" stroke-width="4" stroke-linecap="round" opacity=".42"/>

      <path d="M${w - 16} 30v13 M${w - 16} 57v13" stroke="url(#${id}-lamp)" stroke-width="7" stroke-linecap="round"/>
      <path d="M16 31v12 M16 57v12" stroke="#ff756c" stroke-width="6" stroke-linecap="round"/>
      <path d="M10 50h12 M${w - 22} 50h12" stroke="#dce4e4" stroke-width="3" stroke-linecap="round" opacity=".75"/>
      <path d="M${cabinX + cabinW - 10} 20l9-5v12z M${cabinX + cabinW - 10} 80l9 5V73z" fill="${bodyDark}" stroke="#172126" stroke-width="1.5"/>
    </svg>`;
  }

  function buildTruckSprite(length, color, isTarget) {
    const w = length * 100;
    const h = 100;
    const cabColor = isTarget ? "#ff2f1f" : color;
    const cabDark = isTarget ? "#a9170d" : shade(color, -38);
    const cabLight = isTarget ? "#ff7669" : shade(color, 36);
    const cargoColor = shade(color, 26);
    const cargoDark = shade(color, -22);
    const id = `truck${length}${isTarget ? "t" : color.slice(1)}`;
    const cabW = 82;
    const cabX = w - cabW - 7;
    const cargoX = 8;
    const cargoW = cabX - cargoX - 7;
    const wheels = [cargoX + 30, cargoX + cargoW - 28, cabX + 53]
      .map((cx) => wheelPair(cx, h))
      .join("");
    const ridges = [0.2, 0.4, 0.6, 0.8]
      .map(
        (t) =>
          `<line x1="${cargoX + cargoW * t}" y1="22" x2="${cargoX + cargoW * t}" y2="${h - 22}" stroke="${cargoDark}" stroke-width="2" opacity=".35"/>`,
      )
      .join("");
    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${w} ${h}">
      <defs>
        <linearGradient id="${id}-cab" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0" stop-color="${cabLight}"/>
          <stop offset="0.48" stop-color="${cabColor}"/>
          <stop offset="1" stop-color="${cabDark}"/>
        </linearGradient>
        <linearGradient id="${id}-cargo" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0" stop-color="${shade(color, 46)}"/>
          <stop offset="0.2" stop-color="${cargoColor}"/>
          <stop offset="1" stop-color="${cargoDark}"/>
        </linearGradient>
        <linearGradient id="${id}-glass" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0" stop-color="#e7fbff"/>
          <stop offset=".45" stop-color="#8ec8d5"/>
          <stop offset="1" stop-color="#315a67"/>
        </linearGradient>
      </defs>
      ${wheels}
      <rect x="${cargoX}" y="17" width="${cargoW}" height="66" rx="9" fill="url(#${id}-cargo)" stroke="${cargoDark}" stroke-width="3"/>
      ${ridges}
      <path d="M${cargoX + 7} 23H${cargoX + cargoW - 7}" stroke="#fff" stroke-width="4" stroke-linecap="round" opacity=".4"/>
      <path d="M${cargoX + 7} 77H${cargoX + cargoW - 7}" stroke="#101a1e" stroke-width="3" stroke-linecap="round" opacity=".2"/>
      <path d="M${cargoX + 5} 31v12 M${cargoX + 5} 57v12" stroke="#ff756c" stroke-width="6" stroke-linecap="round"/>

      <path d="M${cabX} 16 Q${cabX} 10 ${cabX + 9} 10 H${w - 25} Q${w - 7} 20 ${w - 7} 37 V63 Q${w - 7} 80 ${w - 25} 90 H${cabX + 9} Q${cabX} 90 ${cabX} 84Z" fill="url(#${id}-cab)" stroke="${cabDark}" stroke-width="3"/>
      <path d="M${cabX + 30} 17H${w - 27}Q${w - 15} 25 ${w - 14} 39H${cabX + 30}Z" fill="url(#${id}-glass)" stroke="#203a43" stroke-width="2"/>
      <path d="M${cabX + 30} 61H${w - 14}Q${w - 15} 75 ${w - 27} 83H${cabX + 30}Z" fill="url(#${id}-glass)" stroke="#203a43" stroke-width="2"/>
      <rect x="${cabX + 15}" y="43" width="${cabW - 27}" height="14" rx="6" fill="${cabColor}"/>
      <path d="M${cabX + 36} 21H${w - 31}" stroke="#fff" stroke-width="4" stroke-linecap="round" opacity=".42"/>
      <path d="M${w - 12} 29v13 M${w - 12} 58v13" stroke="#fff0a6" stroke-width="7" stroke-linecap="round"/>
      <path d="M${w - 31} 20l9-5v13z M${w - 31} 80l9 5V72z" fill="${cabDark}" stroke="#172126" stroke-width="1.5"/>
      <path d="M${cabX - 3} 28v44" stroke="#cad5d5" stroke-width="3" opacity=".7"/>
    </svg>`;
  }

  const spriteCache = new Map();
  function getSprite(length, color, isTarget) {
    const key = `${length}-${color}-${isTarget}`;
    let entry = spriteCache.get(key);
    if (!entry) {
      const img = new Image();
      entry = { img, ready: false };
      img.onload = () => {
        entry.ready = true;
        draw();
      };
      const svg = length >= 3 ? buildTruckSprite(length, color, isTarget) : buildCarSprite(length, color, isTarget);
      img.src = `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
      spriteCache.set(key, entry);
    }
    return entry.ready ? entry.img : null;
  }

  async function loadManifest() {
    const response = await fetch("levels/manifest.json");
    if (!response.ok)
      throw new Error(`Не удалось загрузить список пачек (${response.status})`);
    return response.json();
  }

  async function loadPackLevels(pack) {
    const response = await fetch(`levels/${pack.file}`);
    if (!response.ok)
      throw new Error(`Не удалось загрузить уровни (${response.status})`);
    return response.json();
  }

  const progressKey = () => `parking-puzzle-progress-v3:${state.packId}`;
  const lastIndexKey = () => `parking-puzzle-last-index-v1:${state.packId}`;
  const level = () => state.levels[state.index];

  // level ids are deterministic: <tier>-<packIndex:03>-<levelIndex:04>
  function levelIdAt(index) {
    const pack = state.pack;
    if (!pack) return null;
    return `${pack.tier}-${String(pack.index).padStart(3, "0")}-${String(index).padStart(4, "0")}`;
  }

  function resize() {
    const wrap = canvas.parentElement;
    const wrapRect = wrap.getBoundingClientRect();
    const size = Math.max(120, Math.floor(Math.min(wrapRect.width, wrapRect.height)));
    canvas.style.width = `${size}px`;
    canvas.style.height = `${size}px`;
    const dpr = window.devicePixelRatio || 1;
    canvas.width = Math.round(size * dpr);
    canvas.height = Math.round(size * dpr);
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
    ctx.strokeStyle = "rgba(255,255,255,.4)";
    ctx.lineWidth = 1;
    ctx.setLineDash([5, 5]);
    for (let index = 1; index < 6; index++) {
      ctx.beginPath();
      ctx.moveTo(index * cell, 0);
      ctx.lineTo(index * cell, height);
      ctx.moveTo(0, index * cell);
      ctx.lineTo(width, index * cell);
      ctx.stroke();
    }
    ctx.setLineDash([]);
    if (!level()) return;

    const positions = state.game.positions;
    level().vehicles.forEach((vehicle, index) => {
      let position = positions[index];
      if (state.drag?.index === index) position = state.drag.position;
      else if (state.anim?.index === index) position = state.anim.position;
      const isTarget = index === level().target;
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
      const color = isTarget ? "#ff2f1f" : palette[index % palette.length];
      const sprite = getSprite(vehicle.length, color, isTarget);
      ctx.save();
      ctx.shadowColor = isTarget ? "rgba(255,47,31,.5)" : "rgba(20,40,42,.3)";
      ctx.shadowBlur = isTarget ? 13 : 9;
      ctx.shadowOffsetY = 5;
      if (sprite) {
        const drawW = vehicle.orientation === "horizontal" ? vehicleWidth : vehicleHeight;
        const drawH = vehicle.orientation === "horizontal" ? vehicleHeight : vehicleWidth;
        ctx.translate(x + vehicleWidth / 2, y + vehicleHeight / 2);
        if (vehicle.orientation === "vertical") ctx.rotate(Math.PI / 2);
        ctx.drawImage(sprite, -drawW / 2, -drawH / 2, drawW, drawH);
      } else {
        roundedRect(x, y, vehicleWidth, vehicleHeight, 12);
        ctx.fillStyle = color;
        ctx.fill();
      }
      ctx.restore();
      if (index === state.selected) {
        ctx.save();
        roundedRect(x, y, vehicleWidth, vehicleHeight, 12);
        ctx.strokeStyle = "#f6c85f";
        ctx.lineWidth = 3;
        ctx.stroke();
        ctx.restore();
      }
    });

    const target = level().vehicles[level().target];
    const exitY = (target.fixed + 0.5) * cell;
    ctx.strokeStyle = "#2b9d91";
    ctx.lineWidth = 4;
    ctx.lineCap = "round";
    [0, 11].forEach((offset) => {
      ctx.beginPath();
      ctx.moveTo(width - 24 + offset, exitY - 9);
      ctx.lineTo(width - 12 + offset, exitY);
      ctx.lineTo(width - 24 + offset, exitY + 9);
      ctx.stroke();
    });
  }

  function progressMap() {
    try {
      const value = JSON.parse(localStorage.getItem(progressKey()) || "{}");
      return value && typeof value === "object" && !Array.isArray(value) ? value : {};
    } catch {
      return {};
    }
  }

  function isComplete(index) {
    const id = state.levels[index]?.id ?? levelIdAt(index);
    return Boolean(id && progressMap()[id]);
  }

  function levelResult(index) {
    const id = state.levels[index]?.id ?? levelIdAt(index);
    return (id && progressMap()[id]) || null;
  }

  function computeStars(moves, optimalMoves) {
    if (!optimalMoves) return 3;
    if (moves <= optimalMoves) return 3;
    if (moves <= optimalMoves + Math.max(2, Math.ceil(optimalMoves * 0.5))) return 2;
    return 1;
  }

  function markComplete() {
    const data = progressMap();
    const id = level().id;
    const optimal = level().analysis?.metrics?.optimalMoves;
    const moves = state.game.moves;
    const stars = computeStars(moves, optimal);
    const previous = data[id];
    if (!previous || moves < previous.moves) data[id] = { moves, stars };
    else if (stars > previous.stars) previous.stars = stars;
    localStorage.setItem(progressKey(), JSON.stringify(data));
    return data[id].stars;
  }

  function starGlyphs(count) {
    return "★".repeat(count) + `<span class="dim">${"★".repeat(3 - count)}</span>`;
  }

  function renderLevels() {
    const list = $("level-list");
    list.replaceChildren();
    for (let index = 0; index < state.count; index++) {
      const done = isComplete(index);
      const button = document.createElement("button");
      const number = document.createElement("span");
      const check = document.createElement("span");
      button.className = `level-button ${done ? "done" : ""}`;
      number.textContent = index + 1;
      check.className = "check stars";
      if (done) check.innerHTML = starGlyphs(levelResult(index).stars);
      button.append(number, check);
      button.addEventListener("click", () => {
        loadLevel(index);
        closeDrawer();
      });
      list.append(button);
    }
  }

  function render() {
    draw();
    $("moves").textContent = state.game.moves;
    $("level-count").textContent = `Уровень ${state.index + 1}`;
    $("level-status").textContent = `${state.index + 1} / ${state.count}`;
    $("level-title").textContent =
      level()?.id || `Уровень ${state.index + 1}`;
    $("tier-label").textContent =
      level()?.analysis?.difficultyTier ||
      tierLabels[state.pack?.tier] ||
      "уровень";
    document
      .querySelectorAll(".level-button")
      .forEach((button, index) =>
        button.classList.toggle("active", index === state.index),
      );
    $("undo").disabled = !state.game.canUndo;
    $("next").disabled =
      state.index >= state.count - 1 || !isComplete(state.index);
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

  function dragBounds(index) {
    const start = state.game.positions[index];
    let min = start;
    while (state.game.canMove(index, min - 1)) min--;
    let max = start;
    while (state.game.canMove(index, max + 1)) max++;
    return [min, max];
  }

  function startDrag(event) {
    const [x, y] = pointerPosition(event);
    const index = hitTest(x, y);
    if (index < 0) return;
    const position = state.game.positions[index];
    const [min, max] = dragBounds(index);
    state.selected = index;
    canvas.focus();
    state.drag = {
      index,
      startX: x,
      startY: y,
      startPosition: position,
      position,
      min,
      max,
    };
    canvas.setPointerCapture(event.pointerId);
    draw();
  }

  function moveDrag(event) {
    if (!state.drag) return;
    const [x, y] = pointerPosition(event);
    const vehicle = level().vehicles[state.drag.index];
    const delta =
      vehicle.orientation === "horizontal"
        ? x - state.drag.startX
        : y - state.drag.startY;
    const candidate = state.drag.startPosition + delta;
    state.drag.position = Math.min(state.drag.max, Math.max(state.drag.min, candidate));
    draw();
  }

  function endDrag(event) {
    if (!state.drag) return;
    const drag = state.drag;
    state.drag = null;
    if (canvas.hasPointerCapture(event.pointerId))
      canvas.releasePointerCapture(event.pointerId);
    const target = Math.round(drag.position);
    if (target !== drag.startPosition) commit(drag.index, target, { from: drag.position });
    else draw();
  }

  function animateMove(index, from, to, onDone) {
    const duration = 140;
    const start = performance.now();
    function step(now) {
      const t = Math.min(1, (now - start) / duration);
      const eased = 1 - Math.pow(1 - t, 3);
      state.anim = { index, position: from + (to - from) * eased };
      draw();
      if (t < 1) requestAnimationFrame(step);
      else {
        state.anim = null;
        draw();
        onDone?.();
      }
    }
    requestAnimationFrame(step);
  }

  function commit(index, position, { from } = {}) {
    const start = from ?? state.game.positions[index];
    if (!state.game.move(index, position)) return;
    render();
    const finish = () => {
      if (state.game.isSolved()) win();
    };
    if (start !== position) animateMove(index, start, position, finish);
    else finish();
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

  function confettiBurst() {
    const layer = $("confetti");
    const ctx2d = layer.getContext("2d");
    const dpr = window.devicePixelRatio || 1;
    layer.width = window.innerWidth * dpr;
    layer.height = window.innerHeight * dpr;
    ctx2d.setTransform(dpr, 0, 0, dpr, 0, 0);
    const colors = ["#ed5b4f", "#2b9d91", "#f6c85f", "#4f7cff", "#7c5cff", "#2bb673"];
    const particles = Array.from({ length: 80 }, () => ({
      x: Math.random() * window.innerWidth,
      y: -20 - Math.random() * window.innerHeight * 0.4,
      w: 5 + Math.random() * 5,
      color: colors[Math.floor(Math.random() * colors.length)],
      vy: 2.5 + Math.random() * 3,
      vx: (Math.random() - 0.5) * 2.4,
      rot: Math.random() * 360,
      vr: (Math.random() - 0.5) * 12,
    }));
    const start = performance.now();
    function frame(now) {
      const elapsed = now - start;
      ctx2d.clearRect(0, 0, window.innerWidth, window.innerHeight);
      particles.forEach((p) => {
        p.x += p.vx;
        p.y += p.vy;
        p.rot += p.vr;
        ctx2d.save();
        ctx2d.translate(p.x, p.y);
        ctx2d.rotate((p.rot * Math.PI) / 180);
        ctx2d.fillStyle = p.color;
        ctx2d.fillRect(-p.w / 2, -p.w / 4, p.w, p.w / 2);
        ctx2d.restore();
      });
      if (elapsed < 1500) requestAnimationFrame(frame);
      else ctx2d.clearRect(0, 0, window.innerWidth, window.innerHeight);
    }
    requestAnimationFrame(frame);
  }

  function win() {
    const stars = markComplete();
    renderLevels();
    render();
    const optimal = level().analysis?.metrics?.optimalMoves;
    $("win-stars").innerHTML = starGlyphs(stars);
    $("win-summary").textContent = optimal
      ? `Уровень пройден за ${state.game.moves} ход${state.game.moves === 1 ? "" : "ов"} (оптимально: ${optimal}).`
      : `Уровень пройден за ${state.game.moves} ход${state.game.moves === 1 ? "" : "ов"}.`;
    $("dialog-next").disabled = state.index >= state.count - 1;
    $("win-dialog").hidden = false;
    confettiBurst();
  }

  let levelSeq = 0;
  async function getLevel(index) {
    if (state.levels[index]) return state.levels[index];
    const base = state.pack.file.replace(/\.json$/, "");
    const url = `levels/${base}/${String(index + 1).padStart(4, "0")}.json`;
    const response = await fetch(url);
    if (response.ok) {
      const levelData = await response.json();
      state.levels[index] = levelData;
      return levelData;
    }
    // fallback: whole-pack file (legacy layout) — fetch once per pack
    if (!state.legacy) state.legacy = await loadPackLevels(state.pack);
    const levelData = state.legacy[index];
    if (!levelData) throw new Error(`Уровень ${index + 1} не найден`);
    state.levels[index] = levelData;
    return levelData;
  }

  async function loadLevel(index) {
    const seq = ++levelSeq;
    const packId = state.packId;
    state.index = index;
    state.selected = -1;
    state.drag = null;
    state.anim = null;
    $("win-dialog").hidden = true;
    $("level-title").textContent = `Уровень ${index + 1}…`;
    try {
      const levelData = await getLevel(index);
      if (seq !== levelSeq || packId !== state.packId) return;
      state.game = window.ParkingGame.createGame(levelData);
      localStorage.setItem(lastIndexKey(), String(index));
      clearError();
      render();
    } catch (error) {
      if (seq !== levelSeq || packId !== state.packId) return;
      reportError(error);
    }
  }

  function nextLevel() {
    if (state.index < state.count - 1) loadLevel(state.index + 1);
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
    if (!pack) return;
    state.pack = pack;
    state.packId = pack.id;
    state.levels = [];
    state.legacy = null;
    state.count = pack.count;
    if (persist) localStorage.setItem(packKey, state.packId);
    clearError();
    renderPackOptions();
    renderLevels();
    const savedIndex = Number.parseInt(localStorage.getItem(lastIndexKey()), 10);
    const startIndex =
      Number.isInteger(savedIndex) && savedIndex >= 0 && savedIndex < pack.count
        ? savedIndex
        : 0;
    if (request !== packRequest) return;
    await loadLevel(startIndex);
  }

  function openDrawer() {
    $("level-drawer").classList.add("open");
    $("drawer-backdrop").hidden = false;
  }

  function closeDrawer() {
    $("level-drawer").classList.remove("open");
    $("drawer-backdrop").hidden = true;
  }

  function initTelegram() {
    const tg = window.Telegram?.WebApp;
    if (!tg) return;
    try {
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
        // Android WebView may report 0 at startup -> never collapse the shell
        const reported = Math.max(
          tg.viewportStableHeight || 0,
          tg.viewportHeight || 0,
        );
        const px =
          reported > 200
            ? reported
            : window.innerHeight ||
              document.documentElement.clientHeight ||
              700;
        document.documentElement.style.setProperty(
          "--tg-viewport-height",
          `${px}px`,
        );
      };
      applyTheme();
      applyViewport();
      // stable height arrives async after expand() on some platforms
      setTimeout(applyViewport, 300);
      setTimeout(applyViewport, 1200);
      window.addEventListener("resize", applyViewport);
      tg.onEvent("themeChanged", applyTheme);
      tg.onEvent("viewportChanged", applyViewport);
    } catch (error) {
      // Never let WebApp API issues blank the app
      console.warn("telegram webapp init failed", error);
    }
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
      $("levels-toggle").addEventListener("click", openDrawer);
      $("drawer-backdrop").addEventListener("click", closeDrawer);
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
