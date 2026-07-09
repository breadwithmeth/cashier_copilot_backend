package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"cashier_copilot_backend/internal/model"
	"cashier_copilot_backend/internal/service"

	"github.com/gorilla/websocket"
)

// WSHandler handles WebSocket upgrade requests for operator and cashier clients.
type WSHandler struct {
	hub      *service.Hub
	auth     *service.AuthService
	upgrader websocket.Upgrader
	logger   *slog.Logger
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(hub *service.Hub, auth *service.AuthService, logger *slog.Logger) *WSHandler {
	return &WSHandler{
		hub:  hub,
		auth: auth,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Allow all origins for development — tighten in production
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		logger: logger,
	}
}

// HandleOperatorWS upgrades the connection and registers it as an operator client.
// GET /ws/operator
func (h *WSHandler) HandleOperatorWS(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticateWS(w, r, model.RoleAdmin, model.RoleOperator); !ok {
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("ws_handler: failed to upgrade operator connection", "error", err)
		return
	}

	h.hub.RegisterOperator(conn)

	// Start read pump in a goroutine to handle ping/pong and detect disconnects
	go h.readPump(conn, func() {
		h.hub.UnregisterOperator(conn)
	})
}

// HandleCashierWS upgrades the connection and registers it as a cashier client for a specific POS terminal.
// GET /ws/cashier?pos_id=XXX
func (h *WSHandler) HandleCashierWS(w http.ResponseWriter, r *http.Request) {
	posID := r.URL.Query().Get("pos_id")
	if posID == "" {
		http.Error(w, "missing pos_id query parameter", http.StatusBadRequest)
		return
	}

	user, ok := h.authenticateWS(w, r, model.RoleAdmin, model.RoleOperator, model.RoleCashier)
	if !ok {
		return
	}
	if !service.UserCanAccessPos(user, posID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("ws_handler: failed to upgrade cashier connection",
			"error", err,
			"pos_id", posID,
		)
		return
	}

	h.hub.RegisterCashier(posID, conn)

	// Start read pump
	go h.readPump(conn, func() {
		h.hub.UnregisterCashier(posID, conn)
	})
}

// readPump reads messages from the WebSocket connection to detect disconnects
// and handle ping/pong keepalive. It runs until the connection is closed.
func (h *WSHandler) readPump(conn *websocket.Conn, onClose func()) {
	defer onClose()

	// Configure read deadline and pong handler for keepalive
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Ping writer goroutine
	go func() {
		for range ticker.C {
			if err := conn.WriteControl(
				websocket.PingMessage,
				nil,
				time.Now().Add(10*time.Second),
			); err != nil {
				return
			}
		}
	}()

	// Read loop — we discard incoming messages (clients don't send data)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				h.logger.Warn("ws_handler: unexpected close", "error", err)
			}
			return
		}
	}
}

func (h *WSHandler) authenticateWS(w http.ResponseWriter, r *http.Request, roles ...string) (*model.AuthUser, bool) {
	token := r.URL.Query().Get("token")
	if token == "" {
		token = bearerToken(r.Header.Get("Authorization"))
	}
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return nil, false
	}

	user, err := h.auth.ValidateAccessToken(r.Context(), token)
	if err != nil {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return nil, false
	}

	for _, role := range roles {
		if strings.EqualFold(user.Role, role) {
			return user, true
		}
	}

	http.Error(w, "forbidden", http.StatusForbidden)
	return nil, false
}
