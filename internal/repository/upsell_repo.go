package repository

import (
	"context"
	"fmt"

	"cashier_copilot_backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UpsellRepo handles reading of AI Co-Pilot upsell rules.
type UpsellRepo struct {
	pool *pgxpool.Pool
}

// NewUpsellRepo creates a new UpsellRepo.
func NewUpsellRepo(pool *pgxpool.Pool) *UpsellRepo {
	return &UpsellRepo{pool: pool}
}

// FindByCategory retrieves upsell rules matching a product category.
// Uses ILIKE for case-insensitive prefix/substring matching
// (e.g., category "Алкоголь/Пиво" matches trigger_category "Алкоголь" or "Алкоголь/Пиво").
func (r *UpsellRepo) FindByCategory(ctx context.Context, category string) ([]model.UpsellRule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, trigger_category, required_keywords, suggestion_text, suggestion_image_url
		 FROM upsell_rules
		 WHERE $1 ILIKE trigger_category || '%'
		 ORDER BY id ASC`,
		category,
	)
	if err != nil {
		return nil, fmt.Errorf("find upsell_rules by category: %w", err)
	}
	defer rows.Close()

	var rules []model.UpsellRule
	for rows.Next() {
		var rule model.UpsellRule
		if err := rows.Scan(&rule.ID, &rule.TriggerCategory, &rule.RequiredKeywords,
			&rule.SuggestionText, &rule.SuggestionImageURL); err != nil {
			return nil, fmt.Errorf("scan upsell_rule: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}
