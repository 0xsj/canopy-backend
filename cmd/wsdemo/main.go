package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/0xsj/canopy-backend/pkg/httpserver"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/websocket"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log := logger.NewConsole(
		logger.WithLevel(logger.LevelDebug),
		logger.WithColor(true),
		logger.WithTimestamps(true),
	)

	hub := websocket.NewHub(websocket.Config{SendBufferSize: 256}, log)
	upgrader := websocket.NewNhooyrUpgrader()

	mux := http.NewServeMux()

	// Serve the browser UI.
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, indexHTML)
	})

	// WebSocket endpoint.
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		wsID := r.URL.Query().Get("workspace_id")
		if wsID == "" {
			http.Error(w, "workspace_id required", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			log.Error("upgrade failed", logger.Err(err))
			return
		}

		log.Info("client connected",
			logger.String("workspace", wsID),
			logger.Int("room_size", hub.RoomSize(wsID)+1),
			logger.Int("total", hub.TotalConnections()+1),
		)

		hub.ServeConn(r.Context(), conn, wsID, func(msg websocket.Message) {
			// Echo to the entire room.
			data, _ := msg.Encode()
			hub.Broadcast(wsID, data)
		})

		log.Info("client disconnected",
			logger.String("workspace", wsID),
			logger.Int("room_size", hub.RoomSize(wsID)),
			logger.Int("total", hub.TotalConnections()),
		)
	})

	// Room stats API.
	mux.HandleFunc("GET /api/rooms", func(w http.ResponseWriter, r *http.Request) {
		httpserver.JSON(w, http.StatusOK, map[string]any{
			"total_connections": hub.TotalConnections(),
		})
	})

	mux.HandleFunc("GET /api/rooms/{workspace_id}", func(w http.ResponseWriter, r *http.Request) {
		wsID := r.PathValue("workspace_id")
		httpserver.JSON(w, http.StatusOK, map[string]any{
			"workspace_id": wsID,
			"connections":  hub.RoomSize(wsID),
		})
	})

	// Broadcast from server (POST a message to a room).
	mux.HandleFunc("POST /api/rooms/{workspace_id}/broadcast", func(w http.ResponseWriter, r *http.Request) {
		wsID := r.PathValue("workspace_id")
		msgType := r.URL.Query().Get("type")
		if msgType == "" {
			msgType = "server_announcement"
		}
		text := r.URL.Query().Get("text")
		if text == "" {
			text = "hello from server"
		}

		msg, _ := websocket.NewMessage(msgType, wsID, map[string]string{
			"text": text,
		})
		hub.BroadcastMessage(wsID, msg)

		httpserver.JSON(w, http.StatusOK, map[string]string{
			"status":       "broadcast sent",
			"workspace_id": wsID,
			"type":         msgType,
		})
	})

	handler := httpserver.Chain(
		httpserver.Recovery(log),
		httpserver.Logging(log),
	)(mux)

	srv := httpserver.New(httpserver.Config{
		Host: "0.0.0.0",
		Port: 8081,
	}, handler, log)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", logger.Err(err))
			stop()
		}
	}()

	log.Info("wsdemo ready — open http://localhost:8081 in your browser")

	<-ctx.Done()
	log.Info("shutting down...")
	srv.Shutdown(context.Background())
	log.Info("goodbye")
}

const indexHTML = `<!DOCTYPE html>
<html>
<head>
<title>Canopy WebSocket Demo</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, system-ui, sans-serif; background: #0d1117; color: #c9d1d9; padding: 24px; }
  h1 { color: #58a6ff; margin-bottom: 8px; font-size: 20px; }
  .subtitle { color: #8b949e; margin-bottom: 24px; font-size: 14px; }
  .grid { display: grid; grid-template-columns: 300px 1fr; gap: 24px; max-width: 900px; }
  .panel { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 16px; }
  .panel h2 { font-size: 14px; color: #8b949e; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 12px; }
  label { display: block; font-size: 13px; color: #8b949e; margin-bottom: 4px; }
  input, select { width: 100%; padding: 8px 10px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; color: #c9d1d9; font-size: 14px; margin-bottom: 12px; }
  input:focus { outline: none; border-color: #58a6ff; }
  button { padding: 8px 16px; border: none; border-radius: 6px; font-size: 13px; cursor: pointer; font-weight: 600; }
  .btn-connect { background: #238636; color: white; width: 100%; }
  .btn-connect:hover { background: #2ea043; }
  .btn-disconnect { background: #da3633; color: white; width: 100%; }
  .btn-disconnect:hover { background: #f85149; }
  .btn-send { background: #1f6feb; color: white; }
  .btn-send:hover { background: #388bfd; }
  .status { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; font-size: 13px; }
  .dot { width: 8px; height: 8px; border-radius: 50%; }
  .dot.off { background: #484f58; }
  .dot.on { background: #3fb950; }
  .messages { height: 400px; overflow-y: auto; font-family: 'SF Mono', monospace; font-size: 12px; line-height: 1.6; padding: 12px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; }
  .msg { padding: 2px 0; border-bottom: 1px solid #21262d; }
  .msg .time { color: #484f58; }
  .msg .type { color: #d2a8ff; font-weight: 600; }
  .msg .data { color: #8b949e; }
  .msg.system { color: #58a6ff; }
  .msg.error { color: #f85149; }
  .send-row { display: flex; gap: 8px; margin-top: 12px; }
  .send-row input { margin-bottom: 0; flex: 1; }
  .info { font-size: 12px; color: #484f58; margin-top: 8px; }
</style>
</head>
<body>

<h1>Canopy WebSocket Demo</h1>
<p class="subtitle">Connect to workspace rooms and see real-time messages</p>

<div class="grid">
  <div>
    <div class="panel">
      <h2>Connection</h2>
      <div class="status">
        <div class="dot" id="statusDot"></div>
        <span id="statusText">Disconnected</span>
      </div>
      <label>Workspace ID</label>
      <input id="workspaceId" value="ws_demo_room" placeholder="ws_..." />
      <button id="connectBtn" class="btn-connect" onclick="toggleConnection()">Connect</button>
    </div>

    <div class="panel" style="margin-top: 16px;">
      <h2>Quick Actions</h2>
      <button class="btn-send" style="width:100%; margin-bottom:8px;" onclick="sendPing()">Send Ping</button>
      <button class="btn-send" style="width:100%; margin-bottom:8px; background:#8957e5;" onclick="sendLeaf()">Simulate Leaf</button>
      <p class="info">Messages are echoed to all clients in the same room.</p>
    </div>

    <div class="panel" style="margin-top: 16px;">
      <h2>Room Info</h2>
      <p id="roomInfo" class="info">Not connected</p>
      <button class="btn-send" style="width:100%; margin-top:8px; background:#30363d;" onclick="fetchRoomInfo()">Refresh</button>
    </div>
  </div>

  <div>
    <div class="panel">
      <h2>Messages</h2>
      <div class="messages" id="messages"></div>
      <div class="send-row">
        <input id="msgInput" placeholder="Type a message..." onkeydown="if(event.key==='Enter')sendCustom()" />
        <button class="btn-send" onclick="sendCustom()">Send</button>
      </div>
    </div>
  </div>
</div>

<script>
let ws = null;

function toggleConnection() {
  if (ws) { disconnect(); return; }
  const id = document.getElementById('workspaceId').value.trim();
  if (!id) return;
  connect(id);
}

function connect(workspaceId) {
  const url = "ws://" + location.host + "/ws?workspace_id=" + encodeURIComponent(workspaceId);
  ws = new WebSocket(url);

  ws.onopen = () => {
    setStatus(true);
    addMsg('system', 'Connected to room: ' + workspaceId);
    fetchRoomInfo();
  };

  ws.onmessage = (e) => {
    try {
      const msg = JSON.parse(e.data);
      addMsg('msg', '<span class="type">' + msg.type + '</span> <span class="data">' + JSON.stringify(msg.data) + '</span>');
    } catch {
      addMsg('msg', e.data);
    }
  };

  ws.onclose = () => {
    setStatus(false);
    addMsg('system', 'Disconnected');
    ws = null;
  };

  ws.onerror = () => {
    addMsg('error', 'Connection error');
  };
}

function disconnect() {
  if (ws) ws.close();
}

function setStatus(connected) {
  document.getElementById('statusDot').className = 'dot ' + (connected ? 'on' : 'off');
  document.getElementById('statusText').textContent = connected ? 'Connected' : 'Disconnected';
  document.getElementById('connectBtn').textContent = connected ? 'Disconnect' : 'Connect';
  document.getElementById('connectBtn').className = connected ? 'btn-disconnect' : 'btn-connect';
}

function addMsg(cls, html) {
  const el = document.getElementById('messages');
  const t = new Date().toLocaleTimeString();
  el.innerHTML += '<div class="msg ' + cls + '"><span class="time">' + t + '</span> ' + html + '</div>';
  el.scrollTop = el.scrollHeight;
}

function send(msg) {
  if (!ws || ws.readyState !== 1) { addMsg('error', 'Not connected'); return; }
  ws.send(JSON.stringify(msg));
}

function sendPing() {
  send({ type: 'ping', data: { ts: Date.now() } });
}

function sendLeaf() {
  send({
    type: 'leaf_created',
    workspace_id: document.getElementById('workspaceId').value,
    data: { title: 'Brainstorm idea #' + Math.floor(Math.random()*100), author: 'browser' }
  });
}

function sendCustom() {
  const input = document.getElementById('msgInput');
  const text = input.value.trim();
  if (!text) return;
  send({ type: 'chat', data: { text: text } });
  input.value = '';
}

async function fetchRoomInfo() {
  const id = document.getElementById('workspaceId').value.trim();
  try {
    const [rooms, room] = await Promise.all([
      fetch('/api/rooms').then(r => r.json()),
      id ? fetch('/api/rooms/' + encodeURIComponent(id)).then(r => r.json()) : null,
    ]);
    let info = 'Total connections: ' + rooms.total_connections;
    if (room) info += '\nRoom "' + id + '": ' + room.connections + ' client(s)';
    document.getElementById('roomInfo').textContent = info;
  } catch(e) {
    document.getElementById('roomInfo').textContent = 'Error: ' + e.message;
  }
}
</script>
</body>
</html>`
