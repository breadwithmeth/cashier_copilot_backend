package repository

import (
	"context"
	"fmt"
	"strings"

	"cashier_copilot_backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SpeechRepo handles reading of speech transcript records.
type SpeechRepo struct {
	pool *pgxpool.Pool
}

// NewSpeechRepo creates a new SpeechRepo.
func NewSpeechRepo(pool *pgxpool.Pool) *SpeechRepo {
	return &SpeechRepo{pool: pool}
}

// FetchNew retrieves speech transcripts with ID greater than afterID.
// Used by the background poller.
func (r *SpeechRepo) FetchNew(ctx context.Context, afterID int64) ([]model.SpeechTranscript, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, pos_id, transcript, timestamp_ms, duration_ms,
		        confidence, model_name, weights_version
		 FROM speech_transcripts
		 WHERE id > $1
		 ORDER BY id ASC
		 LIMIT 100`,
		afterID,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch new speech_transcripts: %w", err)
	}
	defer rows.Close()

	var transcripts []model.SpeechTranscript
	for rows.Next() {
		var t model.SpeechTranscript
		if err := rows.Scan(
			&t.ID, &t.PosID, &t.Transcript, &t.TimestampMs, &t.DurationMs,
			&t.Confidence, &t.ModelName, &t.WeightsVersion,
		); err != nil {
			return nil, fmt.Errorf("scan speech_transcript: %w", err)
		}
		transcripts = append(transcripts, t)
	}
	return transcripts, rows.Err()
}

// FindInWindow retrieves speech transcripts for a POS terminal within a time window [fromMs, toMs].
func (r *SpeechRepo) FindInWindow(ctx context.Context, posID string, fromMs, toMs int64) ([]model.SpeechTranscript, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, pos_id, transcript, timestamp_ms, duration_ms,
		        confidence, model_name, weights_version
		 FROM speech_transcripts
		 WHERE pos_id = $1 AND timestamp_ms BETWEEN $2 AND $3
		 ORDER BY timestamp_ms ASC`,
		posID, fromMs, toMs,
	)
	if err != nil {
		return nil, fmt.Errorf("find speech_transcripts in window: %w", err)
	}
	defer rows.Close()

	var transcripts []model.SpeechTranscript
	for rows.Next() {
		var t model.SpeechTranscript
		if err := rows.Scan(
			&t.ID, &t.PosID, &t.Transcript, &t.TimestampMs, &t.DurationMs,
			&t.Confidence, &t.ModelName, &t.WeightsVersion,
		); err != nil {
			return nil, fmt.Errorf("scan speech_transcript: %w", err)
		}
		transcripts = append(transcripts, t)
	}
	return transcripts, rows.Err()
}

// FindWithKeywords searches speech transcripts within a time window that contain any of the given keywords.
// The search is case-insensitive and checks for substring matches in the transcript text.
func (r *SpeechRepo) FindWithKeywords(ctx context.Context, posID string, fromMs, toMs int64, keywords []string) ([]model.SpeechTranscript, error) {
	if len(keywords) == 0 {
		return nil, nil
	}

	// Build a combined ILIKE condition: (LOWER(transcript) LIKE '%keyword1%' OR LOWER(transcript) LIKE '%keyword2%' ...)
	// For simplicity and safety, we use parameterized ILIKE with array_to_string approach.
	// We'll use the application-level filtering after a broader fetch.
	transcripts, err := r.FindInWindow(ctx, posID, fromMs, toMs)
	if err != nil {
		return nil, err
	}

	var matched []model.SpeechTranscript
	for _, t := range transcripts {
		lowerText := strings.ToLower(t.Transcript)
		for _, kw := range keywords {
			if strings.Contains(lowerText, strings.ToLower(kw)) {
				matched = append(matched, t)
				break
			}
		}
	}

	return matched, nil
}
