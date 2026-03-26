const state = {
  roomId: null,
  playerId: null,
  playerName: null,
  eventSource: null,
  current: null,
  activity: [],
};

const elements = {
  createForm: document.getElementById("createForm"),
  joinForm: document.getElementById("joinForm"),
  resumeButton: document.getElementById("resumeButton"),
  startGameButton: document.getElementById("startGameButton"),
  advanceWeekButton: document.getElementById("advanceWeekButton"),
  downloadExcelButton: document.getElementById("downloadExcelButton"),
  copyInviteButton: document.getElementById("copyInviteButton"),
  leaveRoomButton: document.getElementById("leaveRoomButton"),
  orderForm: document.getElementById("orderForm"),
  orderInput: document.getElementById("orderInput"),
  roleButtons: document.getElementById("roleButtons"),
  roomTitle: document.getElementById("roomTitle"),
  roomStatus: document.getElementById("roomStatus"),
  roomCode: document.getElementById("roomCode"),
  playerIdentity: document.getElementById("playerIdentity"),
  playerRole: document.getElementById("playerRole"),
  currentWeek: document.getElementById("currentWeek"),
  weekProgress: document.getElementById("weekProgress"),
  playersList: document.getElementById("playersList"),
  weekReadyChip: document.getElementById("weekReadyChip"),
  orderHint: document.getElementById("orderHint"),
  historyTableBody: document.getElementById("historyTableBody"),
  connectionBadge: document.getElementById("connectionBadge"),
  gameTitle: document.getElementById("gameTitle"),
  roleHint: document.getElementById("roleHint"),
  lobbyPanel: document.getElementById("lobbyPanel"),
  noticeTitle: document.getElementById("noticeTitle"),
  noticeText: document.getElementById("noticeText"),
  phaseTitle: document.getElementById("phaseTitle"),
  phaseText: document.getElementById("phaseText"),
  currentOrderValue: document.getElementById("currentOrderValue"),
  currentOrderText: document.getElementById("currentOrderText"),
  systemStatusValue: document.getElementById("systemStatusValue"),
  systemStatusText: document.getElementById("systemStatusText"),
  metricIncomingOrder: document.getElementById("metricIncomingOrder"),
  metricIncomingGoods: document.getElementById("metricIncomingGoods"),
  metricInventory: document.getElementById("metricInventory"),
  metricBacklog: document.getElementById("metricBacklog"),
  metricShipment: document.getElementById("metricShipment"),
  metricCost: document.getElementById("metricCost"),
  analyticsCost: document.getElementById("analyticsCost"),
  analyticsInventory: document.getElementById("analyticsInventory"),
  analyticsBacklog: document.getElementById("analyticsBacklog"),
  analyticsOrders: document.getElementById("analyticsOrders"),
  analyticsSystemCost: document.getElementById("analyticsSystemCost"),
  ordersChart: document.getElementById("ordersChart"),
  inventoryChart: document.getElementById("inventoryChart"),
  backlogChart: document.getElementById("backlogChart"),
  activityFeed: document.getElementById("activityFeed"),
};

const roles = ["factory", "distributor", "wholesaler", "retailer"];
const maxActivityItems = 12;

boot();

function boot() {
  renderRoleButtons();
  bindForms();
  hydrateInviteFromUrl();
  restoreSession();
}

function bindForms() {
  elements.createForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const name = document.getElementById("createName").value.trim();
      const weeks = Number(document.getElementById("createWeeks").value);
      const room = await api("/rooms", {
        method: "POST",
        body: JSON.stringify({ max_weeks: weeks }),
      });

      await joinRoom(room.id, name);
      pushActivity("Комната создана", `Комната ${room.id} готова. Вы уже подключены как первый игрок.`);
    } catch (error) {
      notifyError(error);
    }
  });

  elements.joinForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const roomId = document.getElementById("joinRoomId").value.trim();
      const name = document.getElementById("joinName").value.trim();
      await joinRoom(roomId, name);
      pushActivity("Подключение выполнено", `Вы вошли в комнату ${roomId}.`);
    } catch (error) {
      notifyError(error);
    }
  });

  elements.resumeButton.addEventListener("click", async () => {
    try {
      const saved = loadSavedSession();
      if (!saved) {
        throw new Error("Сохранённой комнаты пока нет.");
      }

      applySession(saved);
      await refreshState();
      connectEvents();
      pushActivity("Сессия восстановлена", "Подключение к последней комнате восстановлено.");
    } catch (error) {
      notifyError(error);
    }
  });

  elements.copyInviteButton.addEventListener("click", async () => {
    try {
      if (!state.roomId) {
        throw new Error("Сначала подключитесь к комнате.");
      }
      const url = new URL(window.location.href);
      url.pathname = "/app";
      url.search = `room=${encodeURIComponent(state.roomId)}`;
      await navigator.clipboard.writeText(url.toString());
      setNotice("Ссылка скопирована", "Теперь можно отправить ссылку другим участникам комнаты.");
    } catch (error) {
      notifyError(error);
    }
  });

  elements.leaveRoomButton.addEventListener("click", () => {
    disconnectEvents();
    clearSession();
    state.current = null;
    state.activity = [];
    render();
    setNotice("Вы вышли из комнаты", "Можно создать новую комнату или подключиться по другому ID.");
  });

  elements.startGameButton.addEventListener("click", async () => {
    if (!state.roomId) return;
    try {
      await api(`/rooms/${state.roomId}/start`, { method: "POST", body: "{}" });
      pushActivity("Игра началась", "Комната перешла в игровой режим.");
    } catch (error) {
      notifyError(error);
    }
  });

  elements.orderForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!state.roomId || !state.playerId) return;

    try {
      await api(`/rooms/${state.roomId}/orders`, {
        method: "POST",
        body: JSON.stringify({
          player_id: state.playerId,
          order: Number(elements.orderInput.value),
        }),
      });
      pushActivity("Заказ отправлен", `Вы отправили заказ на ${Number(elements.orderInput.value)} ед.`);
    } catch (error) {
      notifyError(error);
    }
  });

  elements.advanceWeekButton.addEventListener("click", async () => {
    if (!state.roomId) return;
    try {
      await api(`/rooms/${state.roomId}/next`, { method: "POST" });
      pushActivity("Неделя завершена", "Система рассчитала новый игровой шаг.");
    } catch (error) {
      notifyError(error);
    }
  });

  elements.downloadExcelButton.addEventListener("click", () => {
    if (!state.roomId) return;
    window.open(`/rooms/${state.roomId}/export`, "_blank");
  });
}

async function joinRoom(roomId, playerName) {
  const room = await api(`/rooms/${roomId}/players`, {
    method: "POST",
    body: JSON.stringify({ name: playerName }),
  });

  const currentPlayer = [...room.players].reverse().find((player) => player.name === playerName);
  if (!currentPlayer) {
    throw new Error("Не удалось определить игрока после входа в комнату.");
  }

  applySession({
    roomId: room.id,
    playerId: currentPlayer.id,
    playerName: currentPlayer.name,
  });

  await refreshState();
  connectEvents();
}

function renderRoleButtons() {
  elements.roleButtons.innerHTML = "";
  roles.forEach((role) => {
    const button = document.createElement("button");
    button.className = "button ghost role-choice";
    button.type = "button";
    button.dataset.role = role;
    button.textContent = role;
    button.addEventListener("click", async () => {
      if (!state.roomId || !state.playerId) return;
      try {
        await api(`/rooms/${state.roomId}/roles`, {
          method: "POST",
          body: JSON.stringify({ player_id: state.playerId, role }),
        });
        pushActivity("Роль выбрана", `За вами закреплена роль ${role}.`);
      } catch (error) {
        notifyError(error);
      }
    });
    elements.roleButtons.appendChild(button);
  });
}

async function refreshState() {
  if (!state.roomId || !state.playerId) return;
  state.current = await api(`/rooms/${state.roomId}/state?player_id=${encodeURIComponent(state.playerId)}`);
  render();
}

function connectEvents() {
  if (!state.roomId || !state.playerId) return;
  disconnectEvents();

  const url = `/rooms/${state.roomId}/events?player_id=${encodeURIComponent(state.playerId)}`;
  state.eventSource = new EventSource(url);
  setConnectionState("connecting");

  state.eventSource.onopen = () => {
    setConnectionState("online");
  };

  const eventNames = [
    "room.snapshot",
    "room.player_joined",
    "room.role_assigned",
    "game.started",
    "game.order_submitted",
    "game.week_advanced",
  ];

  eventNames.forEach((eventName) => {
    state.eventSource.addEventListener(eventName, (event) => {
      const payload = JSON.parse(event.data);
      state.current = payload.state;
      render();
      pushActivity(eventTitle(eventName), eventDescription(eventName, payload.state));
    });
  });

  state.eventSource.onerror = async () => {
    setConnectionState("reconnecting");
    try {
      await refreshState();
    } catch (error) {
      console.error(error);
    }
  };
}

function disconnectEvents() {
  if (state.eventSource) {
    state.eventSource.close();
    state.eventSource = null;
  }
  setConnectionState("offline");
}

function render() {
  const current = state.current;
  if (!current) {
    renderEmpty();
    return;
  }

  elements.roomTitle.textContent = current.room_id;
  elements.roomStatus.textContent = current.room_status;
  elements.roomCode.textContent = `ID: ${current.room_id}`;
  elements.playerIdentity.textContent = current.player_name;
  elements.playerRole.textContent = current.role || "role not selected";
  elements.currentWeek.textContent = `${current.current_week}/${current.max_weeks}`;
  elements.weekProgress.textContent = `${current.orders_submitted}/${current.orders_expected}`;
  elements.gameTitle.textContent = current.role ? `Ваше звено: ${current.role}` : "Выберите роль";
  elements.weekReadyChip.textContent = current.week_ready ? "можно завершать неделю" : `ходы: ${current.orders_submitted}/${current.orders_expected}`;
  elements.orderHint.textContent = current.own_order_submitted
    ? `Ваш заказ на неделю: ${current.own_current_order}`
    : "Заказ ещё не отправлен.";

  renderPlayers(current.players, current.player_name);
  renderRoleAvailability(current);
  renderMetrics(current.own_node);
  renderAnalytics(current);
  renderHistory(current.own_history);
  renderCharts(current.own_history);
  renderActivity();
  renderPhase(current);
  updateControls(current);
  updateUrlState();
}

function renderEmpty() {
  elements.roomTitle.textContent = "Нет активной комнаты";
  elements.roomStatus.textContent = "waiting";
  elements.roomCode.textContent = "-";
  elements.playerIdentity.textContent = "-";
  elements.playerRole.textContent = "-";
  elements.currentWeek.textContent = "-";
  elements.weekProgress.textContent = "-";
  elements.historyTableBody.innerHTML = "";
  elements.playersList.innerHTML = "";
  elements.activityFeed.innerHTML = "";
  renderMetrics(null);
  renderAnalytics({});
  renderCharts([]);
}

function renderPlayers(players, currentName) {
  elements.playersList.innerHTML = "";
  players.forEach((player) => {
    const card = document.createElement("div");
    card.className = "player-card";
    if (player.name === currentName) {
      card.classList.add("current");
    }
    const badge = player.role ? player.role : "role pending";
    card.innerHTML = `
      <div>
        <strong>${escapeHTML(player.name)}</strong><br />
        <small>${player.connected ? "connected" : "offline"}</small>
      </div>
      <span class="player-badge">${escapeHTML(badge)}</span>
    `;
    elements.playersList.appendChild(card);
  });
}

function renderRoleAvailability(current) {
  const taken = new Set(current.players.map((player) => player.role).filter(Boolean));
  Array.from(elements.roleButtons.children).forEach((button) => {
    const role = button.dataset.role;
    button.classList.remove("active", "taken");
    button.disabled = false;

    if (current.role === role) {
      button.classList.add("active");
    } else if (taken.has(role)) {
      button.classList.add("taken");
      button.disabled = true;
    }
    if (current.room_status !== "waiting") {
      button.disabled = true;
    }
  });

  if (!current.role && current.room_status === "waiting") {
    elements.roleHint.textContent = "Свободные роли можно выбрать до запуска игры.";
  } else if (current.role) {
    elements.roleHint.textContent = `За вами закреплена роль ${current.role}.`;
  } else {
    elements.roleHint.textContent = "После старта игры менять роли уже нельзя.";
  }
}

function renderMetrics(node) {
  const value = node || {
    incoming_order: "-",
    incoming_goods: "-",
    inventory: "-",
    backlog: "-",
    actual_shipment: "-",
    weekly_cost: "-",
  };
  elements.metricIncomingOrder.textContent = value.incoming_order;
  elements.metricIncomingGoods.textContent = value.incoming_goods;
  elements.metricInventory.textContent = value.inventory;
  elements.metricBacklog.textContent = value.backlog;
  elements.metricShipment.textContent = value.actual_shipment;
  elements.metricCost.textContent = value.weekly_cost;
}

function renderAnalytics(current) {
  const analytics = current.own_analytics || {};
  elements.analyticsCost.textContent = analytics.total_cost ?? "-";
  elements.analyticsInventory.textContent = analytics.average_inventory != null ? analytics.average_inventory.toFixed(2) : "-";
  elements.analyticsBacklog.textContent = analytics.total_backlog ?? "-";
  elements.analyticsOrders.textContent = analytics.total_orders ?? "-";
  elements.analyticsSystemCost.textContent = current.total_system_cost ?? "-";
}

function renderHistory(history) {
  elements.historyTableBody.innerHTML = "";
  history.forEach((row, index) => {
    const tr = document.createElement("tr");
    tr.innerHTML = `
      <td>${index + 1}</td>
      <td>${row.placed_order}</td>
      <td>${row.incoming_goods}</td>
      <td>${row.inventory}</td>
      <td>${row.backlog}</td>
      <td>${row.actual_shipment}</td>
      <td>${row.weekly_cost}</td>
    `;
    elements.historyTableBody.appendChild(tr);
  });
}

function renderCharts(history) {
  renderBarChart(elements.ordersChart, history.map((item) => item.placed_order), false);
  renderBarChart(elements.inventoryChart, history.map((item) => item.inventory), false);
  renderBarChart(elements.backlogChart, history.map((item) => item.backlog), true);
}

function renderBarChart(container, values, warn) {
  container.innerHTML = "";
  if (!values.length) {
    const placeholder = document.createElement("div");
    placeholder.className = "hint";
    placeholder.textContent = "Пока нет данных";
    container.appendChild(placeholder);
    return;
  }

  const max = Math.max(...values, 1);
  values.forEach((value) => {
    const bar = document.createElement("div");
    bar.className = `bar${warn ? " warn" : ""}`;
    bar.style.height = `${Math.max(10, (value / max) * 72)}px`;
    bar.title = String(value);
    container.appendChild(bar);
  });
}

function renderActivity() {
  elements.activityFeed.innerHTML = "";
  if (!state.activity.length) {
    elements.activityFeed.innerHTML = `<div class="activity-item"><strong>Пока пусто</strong><span class="activity-time">События комнаты появятся после подключения.</span></div>`;
    return;
  }

  state.activity.forEach((item) => {
    const node = document.createElement("div");
    node.className = "activity-item";
    node.innerHTML = `
      <strong>${escapeHTML(item.title)}</strong>
      <div>${escapeHTML(item.text)}</div>
      <div class="activity-time">${escapeHTML(item.time)}</div>
    `;
    elements.activityFeed.appendChild(node);
  });
}

function renderPhase(current) {
  const playerCount = current.players.length;
  const roleAssigned = Boolean(current.role);

  if (current.room_status === "waiting") {
    elements.phaseTitle.textContent = "Лобби";
    elements.phaseText.textContent = roleAssigned
      ? "Роль выбрана. Ждём остальных игроков и запуск игры."
      : "Выберите роль и дождитесь остальных участников.";
    elements.currentOrderValue.textContent = "-";
    elements.currentOrderText.textContent = "Заказы откроются после запуска игры.";
    elements.systemStatusValue.textContent = `${playerCount}/4`;
    elements.systemStatusText.textContent = "Игроков в комнате";
    setNotice(
      roleAssigned ? "Комната собирается" : "Нужно выбрать роль",
      roleAssigned
        ? "Можно отправить ссылку другим участникам и дождаться запуска."
        : "Пока игра не началась, закрепите за собой одно из звеньев."
    );
    return;
  }

  if (current.room_status === "active") {
    elements.phaseTitle.textContent = current.own_order_submitted ? "Ход отправлен" : "Ваш ход";
    elements.phaseText.textContent = current.own_order_submitted
      ? "Ваш заказ записан. Можно ждать остальных игроков."
      : "Оцените своё звено и отправьте заказ наверх по цепи.";
    elements.currentOrderValue.textContent = current.own_order_submitted ? String(current.own_current_order) : "не задан";
    elements.currentOrderText.textContent = current.week_ready
      ? "Все решения собраны, можно завершить неделю."
      : "Система ждёт остальные ходы комнаты.";
    elements.systemStatusValue.textContent = current.week_ready ? "ready" : "syncing";
    elements.systemStatusText.textContent = `Получено ${current.orders_submitted} из ${current.orders_expected} решений`;
    setNotice(
      current.week_ready ? "Неделя готова к расчёту" : "Игра идёт",
      current.week_ready
        ? "Теперь можно нажать «Перейти к следующей неделе»."
        : "Игроки видят только свои данные, поэтому ориентируйтесь на собственную динамику."
    );
    return;
  }

  elements.phaseTitle.textContent = "Игра завершена";
  elements.phaseText.textContent = "Можно скачать Excel и проанализировать историю своего звена.";
  elements.currentOrderValue.textContent = current.own_order_submitted ? String(current.own_current_order) : "-";
  elements.currentOrderText.textContent = "Новые заказы уже не принимаются.";
  elements.systemStatusValue.textContent = "finished";
  elements.systemStatusText.textContent = "Сессия завершена";
  setNotice("Сессия завершена", "Скачайте Excel-отчёт или перейдите в новую комнату.");
}

function updateControls(current) {
  const waiting = current.room_status === "waiting";
  const active = current.room_status === "active";

  elements.startGameButton.disabled = current.players.length < 4 || waiting === false;
  elements.advanceWeekButton.disabled = !current.week_ready || !active;
  elements.orderInput.disabled = !active;
  elements.downloadExcelButton.disabled = current.room_status === "waiting";
  elements.lobbyPanel.classList.toggle("hidden", !waiting);
}

function setConnectionState(status) {
  elements.connectionBadge.classList.remove("online", "error");
  elements.connectionBadge.textContent = status;
  if (status === "online") {
    elements.connectionBadge.classList.add("online");
  }
  if (status === "error") {
    elements.connectionBadge.classList.add("error");
  }
}

function pushActivity(title, text) {
  state.activity.unshift({
    title,
    text,
    time: new Date().toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" }),
  });
  state.activity = state.activity.slice(0, maxActivityItems);
  renderActivity();
}

function eventTitle(eventName) {
  switch (eventName) {
    case "room.player_joined":
      return "Новый игрок подключился";
    case "room.role_assigned":
      return "В комнате назначена роль";
    case "game.started":
      return "Игра запущена";
    case "game.order_submitted":
      return "Получено новое решение";
    case "game.week_advanced":
      return "Неделя пересчитана";
    default:
      return "Состояние обновлено";
  }
}

function eventDescription(eventName, current) {
  switch (eventName) {
    case "room.player_joined":
      return `Игроков в комнате: ${current.players.length}.`;
    case "room.role_assigned":
      return `Ваш текущий статус роли: ${current.role || "ещё не выбрана"}.`;
    case "game.started":
      return "Теперь можно анализировать своё звено и отправлять заказы.";
    case "game.order_submitted":
      return `Готовность недели: ${current.orders_submitted}/${current.orders_expected}.`;
    case "game.week_advanced":
      return `Комната перешла к неделе ${current.current_week}.`;
    default:
      return "Получено новое состояние комнаты.";
  }
}

async function api(path, options = {}) {
  const response = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });

  if (!response.ok) {
    const contentType = response.headers.get("Content-Type") || "";
    if (contentType.includes("application/json")) {
      const payload = await response.json();
      throw new Error(payload.error || `Request failed: ${response.status}`);
    }
    throw new Error((await response.text()) || `Request failed: ${response.status}`);
  }

  const contentType = response.headers.get("Content-Type") || "";
  if (contentType.includes("application/json")) {
    return response.json();
  }

  return response.text();
}

function applySession(session) {
  state.roomId = session.roomId;
  state.playerId = session.playerId;
  state.playerName = session.playerName;
  persistSession();
}

function persistSession() {
  localStorage.setItem("supply-chain-player", JSON.stringify({
    roomId: state.roomId,
    playerId: state.playerId,
    playerName: state.playerName,
  }));
}

function loadSavedSession() {
  try {
    return JSON.parse(localStorage.getItem("supply-chain-player"));
  } catch (_error) {
    return null;
  }
}

function clearSession() {
  localStorage.removeItem("supply-chain-player");
  state.roomId = null;
  state.playerId = null;
  state.playerName = null;
  history.replaceState({}, "", "/app");
}

function restoreSession() {
  const params = new URLSearchParams(window.location.search);
  const saved = loadSavedSession();

  if (params.get("room") && params.get("player")) {
    applySession({
      roomId: params.get("room"),
      playerId: params.get("player"),
      playerName: params.get("name") || saved?.playerName || "",
    });
  } else if (saved) {
    applySession(saved);
  } else {
    render();
    return;
  }

  refreshState()
    .then(() => {
      connectEvents();
      pushActivity("Комната восстановлена", "Подписка на события комнаты активна.");
    })
    .catch((error) => {
      notifyError(error);
      disconnectEvents();
    });
}

function hydrateInviteFromUrl() {
  const params = new URLSearchParams(window.location.search);
  const roomId = params.get("room");
  if (roomId) {
    document.getElementById("joinRoomId").value = roomId;
    setNotice("Приглашение получено", `Можно войти в комнату ${roomId} под своим именем.`);
  }
}

function updateUrlState() {
  if (!state.roomId || !state.playerId) return;
  const params = new URLSearchParams(window.location.search);
  params.set("room", state.roomId);
  params.set("player", state.playerId);
  if (state.playerName) {
    params.set("name", state.playerName);
  }
  history.replaceState({}, "", `/app?${params.toString()}`);
}

function setNotice(title, text) {
  elements.noticeTitle.textContent = title;
  elements.noticeText.textContent = text;
}

function notifyError(error) {
  console.error(error);
  setConnectionState("error");
  setNotice("Ошибка", error.message || "Произошла непредвиденная ошибка.");
  pushActivity("Ошибка", error.message || "Произошла непредвиденная ошибка.");
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll("\"", "&quot;")
    .replaceAll("'", "&#039;");
}
