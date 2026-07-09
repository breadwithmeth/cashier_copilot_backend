package service

import (
	"encoding/json"
	"log/slog"
	"sync"

	"cashier_copilot_backend/internal/model"

	"github.com/gorilla/websocket"
)

// Hub manages WebSocket connections and message broadcasting.
// It maintains two separate client pools: operator clients and per-POS cashier clients.
type Hub struct {
	// Operator connections (panel of the security operator)
	operatorMu      sync.RWMutex
	operatorClients map[*websocket.Conn]bool

	// Cashier connections keyed by pos_id
	cashierMu      sync.RWMutex
	cashierClients map[string]map[*websocket.Conn]bool

	logger *slog.Logger
}

// NewHub creates a new Hub.
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		operatorClients: make(map[*websocket.Conn]bool),
		cashierClients:  make(map[string]map[*websocket.Conn]bool),
		logger:          logger,
	}
}

// RegisterOperator adds an operator WebSocket connection.
func (h *Hub) RegisterOperator(conn *websocket.Conn) {
	h.operatorMu.Lock()
	defer h.operatorMu.Unlock()
	h.operatorClients[conn] = true
	h.logger.Info("operator client connected", "total", len(h.operatorClients))
}

// UnregisterOperator removes an operator WebSocket connection.
func (h *Hub) UnregisterOperator(conn *websocket.Conn) {
	h.operatorMu.Lock()
	defer h.operatorMu.Unlock()
	delete(h.operatorClients, conn)
	conn.Close()
	h.logger.Info("operator client disconnected", "total", len(h.operatorClients))
}

// RegisterCashier adds a cashier WebSocket connection for a specific POS terminal.
func (h *Hub) RegisterCashier(posID string, conn *websocket.Conn) {
	h.cashierMu.Lock()
	defer h.cashierMu.Unlock()
	if h.cashierClients[posID] == nil {
		h.cashierClients[posID] = make(map[*websocket.Conn]bool)
	}
	h.cashierClients[posID][conn] = true
	h.logger.Info("cashier client connected", "pos_id", posID, "total", len(h.cashierClients[posID]))
}

// UnregisterCashier removes a cashier WebSocket connection.
func (h *Hub) UnregisterCashier(posID string, conn *websocket.Conn) {
	h.cashierMu.Lock()
	defer h.cashierMu.Unlock()
	if clients, ok := h.cashierClients[posID]; ok {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(h.cashierClients, posID)
		}
	}
	conn.Close()
	h.logger.Info("cashier client disconnected", "pos_id", posID)
}

// BroadcastToOperators sends a message to all connected operator clients.
func (h *Hub) BroadcastToOperators(msg model.WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("failed to marshal WS message for operators", "error", err)
		return
	}

	h.operatorMu.RLock()
	defer h.operatorMu.RUnlock()

	for conn := range h.operatorClients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			h.logger.Warn("failed to send WS message to operator", "error", err)
			// Mark for cleanup — actual unregister happens from the read pump
			go func(c *websocket.Conn) {
				h.UnregisterOperator(c)
			}(conn)
		}
	}
}

// SendToCashier sends a message to all WebSocket connections for a specific POS terminal.
func (h *Hub) SendToCashier(posID string, msg model.WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("failed to marshal WS message for cashier", "error", err, "pos_id", posID)
		return
	}

	h.cashierMu.RLock()
	defer h.cashierMu.RUnlock()

	clients, ok := h.cashierClients[posID]
	if !ok {
		return
	}

	for conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			h.logger.Warn("failed to send WS message to cashier", "error", err, "pos_id", posID)
			go func(c *websocket.Conn) {
				h.UnregisterCashier(posID, c)
			}(conn)
		}
	}
}

// BroadcastViolationAlert sends a new violation alert to all operator clients.
func (h *Hub) BroadcastViolationAlert(alert model.ViolationAlert) {
	h.BroadcastToOperators(model.WSMessage{
		Type:    "violation_alert",
		Payload: alert,
	})
}

// BroadcastTaskStatus sends a task completion/failure update to operator clients.
func (h *Hub) BroadcastTaskStatus(update model.TaskStatusUpdate) {
	h.BroadcastToOperators(model.WSMessage{
		Type:    "task_status",
		Payload: update,
	})
}

// SendUpsellCard sends an upsell recommendation card to the cashier terminal.
func (h *Hub) SendUpsellCard(posID string, card model.UpsellCard) {
	h.SendToCashier(posID, model.WSMessage{
		Type:    "upsell_card",
		Payload: card,
	})
}

// SendUpsellStatusUpdate sends an upsell completion notification to the cashier terminal.
func (h *Hub) SendUpsellStatusUpdate(posID string, update model.UpsellStatusUpdate) {
	h.SendToCashier(posID, model.WSMessage{
		Type:    "upsell_status",
		Payload: update,
	})
}
