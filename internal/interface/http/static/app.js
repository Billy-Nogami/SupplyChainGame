const state = {
  roomId: null,
  playerId: null,
  playerName: null,
  eventSource: null,
  current: null,
};

const elements = {
  createForm: document.getElementById("createForm"),
  joinForm: document.getElementById("joinForm"),
  resumeButton: document.getElementById("resumeButton"),
  startGameButton: document.getElementById("startGameButton"),
  advanceWeekButton: document.getElementById("advanceWeekButton"),
  downloadExcelButton: document.getElementById("downloadExcelButton"),
  orderForm: document.getElementById("orderForm"),
  orderInput: document.getElementById("orderInput"),
  roleButtons: document.getElementById("roleButtons"),
  roomTitle: document.getElementById("roomTitle"),
  roomStatus: document.getElementById("roomStatus"),
  playerIdentity: document.getElementById("playerIdentity"),
  playerRole: document.getElementById("playerRole"),
  currentWeek: document.getElementById("currentWeek"),
  playersList: document.getElementById("playersList"),
  weekReadyChip: document.getElementById("weekReadyChip"),
  orderHint: document.getElementById("orderHint"),
  historyTableBody: document.getElementById("historyTableBody"),
  connectionBadge: document.getElementById("connectionBadge"),
  gameTitle: document.getElementById("gameTitle"),
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
};

const roles = ["factory", "distributor", "wholesaler", "retailer"];

boot();

function boot() {
  renderRoleButtons();
  bindForms();
  restoreSession();
}

function bindForms() {
  elements.createForm.addEventListener("submit", async (event) => {
    event.preventDefault();

    const name = document.getElementById("createName").value.trim();
    const weeks = Number(document.getElementById("createWeeks").value);
    const room = await api("/rooms", {
      method: "POST",
      body: JSON.stringify({ max_weeks: weeks }),
    });

    await joinRoom(room.id, name);
  });

  elements.joinForm.addEventListener("submit", async (event) => {
    event.preventDefault();

    const roomId = document.getElementById("joinRoomId").value.trim();
    const name = document.getElementById("joinName").value.trim();
    await joinRoom(roomId, name);
  });

  elements.resumeButton.addEventListener("click", async () => {
    const saved = loadSavedSession();
    if (!saved) {
      alert("Сохранённой комнаты пока нет.");
      return;
    }

    state.roomId = saved.roomId;
    state.playerId = saved.playerId;
    state.playerName = saved.playerName;
    await refreshState();
    connectEvents();
  });

  elements.startGameButton.addEventListener("click", async () => {
    if (!state.roomId) return;
    await api(`/rooms/${state.roomId}/start`, { method: "POST", body: "{}" });
  });

  elements.orderForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!state.roomId || !state.playerId) return;

    await api(`/rooms/${state.roomId}/orders`, {
      method: "POST",
      body: JSON.stringify({
        player_id: state.playerId,
        order: Number(elements.orderInput.value),
      }),
    });
  });

  elements.advanceWeekButton.addEventListener("click", async () => {
    if (!state.roomId) return;
    await api(`/rooms/${state.roomId}/next`, { method: "POST" });
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

  state.roomId = room.id;
  state.playerId = currentPlayer.id;
  state.playerName = currentPlayer.name;
  persistSession();
  await refreshState();
  connectEvents();
}

function renderRoleButtons() {
  elements.roleButtons.innerHTML = "";
  roles.forEach((role) => {
    const button = document.createElement("button");
    button.className = "button ghost";
    button.type = "button";
    button.textContent = role;
    button.addEventListener("click", async () => {
      if (!state.roomId || !state.playerId) return;
      await api(`/rooms/${state.roomId}/roles`, {
        method: "POST",
        body: JSON.stringify({ player_id: state.playerId, role }),
      });
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
  if (state.eventSource) {
    state.eventSource.close();
  }

  const url = `/rooms/${state.roomId}/events?player_id=${encodeURIComponent(state.playerId)}`;
  state.eventSource = new EventSource(url);
  elements.connectionBadge.textContent = "connecting";
  elements.connectionBadge.classList.remove("online");

  state.eventSource.onopen = () => {
    elements.connectionBadge.textContent = "online";
    elements.connectionBadge.classList.add("online");
  };

  state.eventSource.onmessage = () => {};

  [
    "room.snapshot",
    "room.player_joined",
    "room.role_assigned",
    "game.started",
    "game.order_submitted",
    "game.week_advanced",
  ].forEach((eventName) => {
    state.eventSource.addEventListener(eventName, (event) => {
      const payload = JSON.parse(event.data);
      state.current = payload.state;
      render();
    });
  });

  state.eventSource.onerror = async () => {
    elements.connectionBadge.textContent = "reconnecting";
    elements.connectionBadge.classList.remove("online");
    try {
      await refreshState();
    } catch (error) {
      console.error(error);
    }
  };
}

function render() {
  const current = state.current;
  if (!current) return;

  elements.roomTitle.textContent = current.room_id;
  elements.roomStatus.textContent = current.room_status;
  elements.playerIdentity.textContent = current.player_name;
  elements.playerRole.textContent = current.role || "role not selected";
  elements.currentWeek.textContent = `${current.current_week}/${current.max_weeks}`;
  elements.gameTitle.textContent = current.role ? `Ваше звено: ${current.role}` : "Выберите роль";
  elements.weekReadyChip.textContent = current.week_ready ? "можно завершать неделю" : `ходы: ${current.orders_submitted}/${current.orders_expected}`;
  elements.orderHint.textContent = current.own_order_submitted
    ? `Ваш заказ на неделю: ${current.own_current_order}`
    : "Заказ ещё не отправлен.";

  renderPlayers(current.players, current.player_name);
  renderMetrics(current.own_node);
  renderAnalytics(current);
  renderHistory(current.own_history);

  elements.startGameButton.disabled = current.players.length < 4 || current.room_status !== "waiting";
  elements.advanceWeekButton.disabled = !current.week_ready;
  elements.orderInput.disabled = current.room_status !== "active";
}

function renderPlayers(players, currentName) {
  elements.playersList.innerHTML = "";
  players.forEach((player) => {
    const card = document.createElement("div");
    card.className = "player-card";
    if (player.name === currentName) {
      card.classList.add("current");
    }
    card.innerHTML = `<strong>${player.name}</strong><br /><small>${player.role || "role not chosen yet"}</small>`;
    elements.playersList.appendChild(card);
  });
}

function renderMetrics(node) {
  const empty = { incoming_order: "-", incoming_goods: "-", inventory: "-", backlog: "-", actual_shipment: "-", weekly_cost: "-" };
  const value = node || empty;
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

async function api(path, options = {}) {
  const response = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(error || `Request failed: ${response.status}`);
  }

  const contentType = response.headers.get("Content-Type") || "";
  if (contentType.includes("application/json")) {
    return response.json();
  }

  return response.text();
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
  } catch (error) {
    return null;
  }
}

function restoreSession() {
  const saved = loadSavedSession();
  if (!saved) return;

  state.roomId = saved.roomId;
  state.playerId = saved.playerId;
  state.playerName = saved.playerName;
  refreshState().then(connectEvents).catch(console.error);
}
