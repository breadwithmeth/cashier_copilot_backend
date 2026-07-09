package service

import (
	"fmt"
	"log/slog"
	"sync"

	"cashier_copilot_backend/internal/model"
)

// FSMManager manages per-POS-terminal state machines in a thread-safe manner.
type FSMManager struct {
	mu       sync.RWMutex
	machines map[string]*CashierFSM
	logger   *slog.Logger
}

// CashierFSM holds the current state context of a single POS terminal.
type CashierFSM struct {
	PosID       string
	State       model.CashierState
	ReceiptID   string
	LastEventMs int64
}

// NewFSMManager creates a new FSMManager.
func NewFSMManager(logger *slog.Logger) *FSMManager {
	return &FSMManager{
		machines: make(map[string]*CashierFSM),
		logger:   logger,
	}
}

// GetOrCreate returns the FSM for a given pos_id, creating one in Idle state if it doesn't exist.
func (m *FSMManager) GetOrCreate(posID string) *CashierFSM {
	m.mu.Lock()
	defer m.mu.Unlock()

	fsm, exists := m.machines[posID]
	if !exists {
		fsm = &CashierFSM{
			PosID: posID,
			State: model.StateIdle,
		}
		m.machines[posID] = fsm
		m.logger.Info("FSM created for POS terminal",
			"pos_id", posID,
			"initial_state", model.StateIdle,
		)
	}
	return fsm
}

// GetState returns the current state of a POS terminal (thread-safe read).
func (m *FSMManager) GetState(posID string) model.CashierState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fsm, exists := m.machines[posID]
	if !exists {
		return model.StateIdle
	}
	return fsm.State
}

// TransitionPosEvent handles state transitions triggered by POS events from 1C.
// Returns (oldState, newState, error). An error indicates an invalid transition.
func (m *FSMManager) TransitionPosEvent(posID string, event *model.PosEvent) (model.CashierState, model.CashierState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	fsm := m.getOrCreateLocked(posID)
	oldState := fsm.State

	var newState model.CashierState
	var err error

	switch event.EventType {
	case "receipt_opened":
		switch fsm.State {
		case model.StateIdle:
			// Warning: receipt opened without customer detected — allowed per FSM diagram
			newState = model.StateReceiptOpened
		case model.StateCustomerDetected:
			newState = model.StateReceiptOpened
		default:
			err = fmt.Errorf("invalid transition: receipt_opened in state %s", fsm.State)
			newState = fsm.State
		}
		if err == nil {
			fsm.ReceiptID = event.ReceiptID
		}

	case "item_scanned":
		switch fsm.State {
		case model.StateReceiptOpened:
			newState = model.StateScanning
		case model.StateScanning:
			newState = model.StateScanning // stay in Scanning
		default:
			err = fmt.Errorf("invalid transition: item_scanned in state %s", fsm.State)
			newState = fsm.State
		}

	case "item_removed":
		switch fsm.State {
		case model.StateScanning:
			newState = model.StateScanning // stay in Scanning
		default:
			err = fmt.Errorf("invalid transition: item_removed in state %s", fsm.State)
			newState = fsm.State
		}

	case "receipt_cancelled":
		// Cancellation can happen from ReceiptOpened or Scanning
		switch fsm.State {
		case model.StateReceiptOpened, model.StateScanning:
			newState = model.StateIdle
			fsm.ReceiptID = ""
		default:
			err = fmt.Errorf("invalid transition: receipt_cancelled in state %s", fsm.State)
			newState = fsm.State
		}

	case "loyalty_card_applied":
		// Loyalty card doesn't change FSM state, but is valid during Scanning or Payment
		newState = fsm.State

	case "payment_started":
		switch fsm.State {
		case model.StateScanning:
			newState = model.StatePayment
		default:
			err = fmt.Errorf("invalid transition: payment_started in state %s", fsm.State)
			newState = fsm.State
		}

	case "receipt_closed":
		switch fsm.State {
		case model.StateReceiptOpened, model.StateScanning, model.StatePayment:
			newState = model.StateReceiptClosed
		default:
			err = fmt.Errorf("invalid transition: receipt_closed in state %s", fsm.State)
			newState = fsm.State
		}

	default:
		err = fmt.Errorf("unknown POS event type: %s", event.EventType)
		newState = fsm.State
	}

	if err == nil && newState != oldState {
		fsm.State = newState
		fsm.LastEventMs = event.TimestampMs
		m.logger.Info("FSM state transition (POS)",
			"pos_id", posID,
			"old_state", oldState,
			"new_state", newState,
			"event_type", event.EventType,
			"receipt_id", event.ReceiptID,
		)
	} else if err != nil {
		m.logger.Warn("FSM invalid transition (POS)",
			"pos_id", posID,
			"state", fsm.State,
			"event_type", event.EventType,
			"error", err,
		)
	}

	return oldState, newState, err
}

// TransitionCvEvent handles state transitions triggered by CV detection events.
// Returns (oldState, newState, error).
func (m *FSMManager) TransitionCvEvent(posID string, cvEvent *model.CvEvent) (model.CashierState, model.CashierState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	fsm := m.getOrCreateLocked(posID)
	oldState := fsm.State

	var newState model.CashierState

	switch cvEvent.EventType {
	case "customer_present":
		switch fsm.State {
		case model.StateIdle:
			newState = model.StateCustomerDetected
		default:
			// Customer presence doesn't change state when already in a transaction
			newState = fsm.State
		}

	case "customer_left":
		switch fsm.State {
		case model.StateCustomerDetected:
			// Customer left without opening a receipt
			newState = model.StateIdle
		case model.StateReceiptClosed:
			// Customer leaves after receipt close — back to Idle
			newState = model.StateIdle
			fsm.ReceiptID = ""
		default:
			newState = fsm.State
		}

	default:
		// Other CV events (hand_to_drawer, item_in_bag, etc.) don't trigger FSM transitions
		newState = fsm.State
	}

	if newState != oldState {
		fsm.State = newState
		fsm.LastEventMs = cvEvent.TimestampMs
		m.logger.Info("FSM state transition (CV)",
			"pos_id", posID,
			"old_state", oldState,
			"new_state", newState,
			"cv_event_type", cvEvent.EventType,
			"camera_id", cvEvent.CameraID,
		)
	}

	return oldState, newState, nil
}

// GetReceiptID returns the current active receipt ID for a POS terminal.
func (m *FSMManager) GetReceiptID(posID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fsm, exists := m.machines[posID]
	if !exists {
		return ""
	}
	return fsm.ReceiptID
}

// getOrCreateLocked returns the FSM without acquiring a lock (caller must hold the lock).
func (m *FSMManager) getOrCreateLocked(posID string) *CashierFSM {
	fsm, exists := m.machines[posID]
	if !exists {
		fsm = &CashierFSM{
			PosID: posID,
			State: model.StateIdle,
		}
		m.machines[posID] = fsm
	}
	return fsm
}
