package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"

	"cashier_copilot_backend/internal/model"
	"cashier_copilot_backend/internal/repository"
)

// CoPilot implements the AI Co-Pilot feature that provides upsell recommendations
// to cashiers and tracks their verbal completion.
type CoPilot struct {
	upsellRepo *repository.UpsellRepo
	speechRepo *repository.SpeechRepo
	fsm        *FSMManager
	hub        *Hub
	logger     *slog.Logger

	// activeUpsells tracks pending upsell suggestions per receipt
	// key: receiptID
	mu            sync.RWMutex
	activeUpsells map[string]activeUpsell
}

type activeUpsell struct {
	card             model.UpsellCard
	requiredKeywords []string
}

// NewCoPilot creates a new CoPilot.
func NewCoPilot(
	upsellRepo *repository.UpsellRepo,
	speechRepo *repository.SpeechRepo,
	fsm *FSMManager,
	hub *Hub,
	logger *slog.Logger,
) *CoPilot {
	return &CoPilot{
		upsellRepo:    upsellRepo,
		speechRepo:    speechRepo,
		fsm:           fsm,
		hub:           hub,
		logger:        logger,
		activeUpsells: make(map[string]activeUpsell),
	}
}

// HandleItemScanned processes an item_scanned event to check for upsell opportunities.
// If a matching upsell rule exists for the item's category, it sends a recommendation card
// to the cashier's terminal via WebSocket.
func (cp *CoPilot) HandleItemScanned(ctx context.Context, posEvent *model.PosEvent) {
	// Parse item details to get category
	var details model.PosEventDetails
	if err := json.Unmarshal(posEvent.Details, &details); err != nil {
		cp.logger.Debug("copilot: failed to parse item details", "error", err)
		return
	}

	if details.Category == "" {
		return
	}

	cp.logger.Info("copilot: checking upsell rules",
		"pos_id", posEvent.PosID,
		"category", details.Category,
		"item_name", details.ItemName,
	)

	// Query upsell rules for this category
	rules, err := cp.upsellRepo.FindByCategory(ctx, details.Category)
	if err != nil {
		cp.logger.Error("copilot: failed to query upsell rules", "error", err)
		return
	}

	if len(rules) == 0 {
		return
	}

	// Use the first matching rule
	rule := rules[0]

	receiptID := cp.fsm.GetReceiptID(posEvent.PosID)
	if receiptID == "" {
		receiptID = posEvent.ReceiptID
	}
	if receiptID == "" {
		cp.logger.Warn("copilot: cannot track upsell without receipt_id", "pos_id", posEvent.PosID)
		return
	}

	// Check if we already have an active upsell for this receipt (avoid duplicates)
	cp.mu.Lock()
	if _, exists := cp.activeUpsells[receiptID]; exists {
		cp.mu.Unlock()
		cp.logger.Debug("copilot: upsell already active for receipt", "receipt_id", receiptID)
		return
	}

	// Create and send upsell card
	card := model.UpsellCard{
		PosID:           posEvent.PosID,
		ReceiptID:       receiptID,
		TriggerItem:     details.ItemName,
		SuggestionText:  rule.SuggestionText,
		SuggestionImage: rule.SuggestionImageURL,
		Status:          "pending",
	}

	cp.activeUpsells[receiptID] = activeUpsell{
		card:             card,
		requiredKeywords: append([]string(nil), rule.RequiredKeywords...),
	}
	cp.mu.Unlock()

	cp.hub.SendUpsellCard(posEvent.PosID, card)

	cp.logger.Info("copilot: upsell card sent to cashier",
		"pos_id", posEvent.PosID,
		"receipt_id", receiptID,
		"trigger_item", details.ItemName,
		"suggestion", rule.SuggestionText,
	)
}

// CheckUpsellCompletion checks speech transcripts for the current receipt to determine
// if the cashier verbally offered the upsell product. Called by the speech poller.
func (cp *CoPilot) CheckUpsellCompletion(ctx context.Context, transcript *model.SpeechTranscript) {
	posID := transcript.PosID
	receiptID := cp.fsm.GetReceiptID(posID)
	if receiptID == "" {
		return
	}

	cp.mu.RLock()
	active, exists := cp.activeUpsells[receiptID]
	cp.mu.RUnlock()
	if !exists {
		return
	}

	card := active.card
	if card.Status == "completed" {
		return
	}

	if len(active.requiredKeywords) == 0 {
		return
	}

	lowerTranscript := strings.ToLower(transcript.Transcript)

	for _, keyword := range active.requiredKeywords {
		if strings.Contains(lowerTranscript, strings.ToLower(keyword)) {
			card.Status = "completed"
			active.card = card

			cp.mu.Lock()
			cp.activeUpsells[receiptID] = active
			cp.mu.Unlock()

			cp.hub.SendUpsellStatusUpdate(posID, model.UpsellStatusUpdate{
				PosID:     posID,
				ReceiptID: receiptID,
				Status:    "completed",
			})

			cp.logger.Info("copilot: upsell completed via speech",
				"pos_id", posID,
				"receipt_id", receiptID,
				"matched_keyword", keyword,
				"transcript_id", transcript.ID,
			)
			return
		}
	}
}

// ClearReceipt removes the active upsell tracking for a closed receipt.
func (cp *CoPilot) ClearReceipt(receiptID string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	delete(cp.activeUpsells, receiptID)
}
