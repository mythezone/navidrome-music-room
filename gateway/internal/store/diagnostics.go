package store

import (
	"context"
	"fmt"
)

// DiagnosticSummary intentionally contains aggregate database health only. It
// never returns room names, identifiers, usernames, invitation data, or audit
// payloads, so it is safe to include in an explicitly requested support bundle.
type DiagnosticSummary struct {
	QuickCheck    string           `json:"quickCheck"`
	SchemaVersion int              `json:"schemaVersion"`
	PageCount     int64            `json:"pageCount"`
	PageSize      int64            `json:"pageSize"`
	TableCounts   map[string]int64 `json:"tableCounts"`
}

func (s *Store) Diagnostics(ctx context.Context) (DiagnosticSummary, error) {
	result := DiagnosticSummary{TableCounts: make(map[string]int64)}
	if err := s.db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&result.QuickCheck); err != nil {
		return DiagnosticSummary{}, fmt.Errorf("database quick check: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&result.SchemaVersion); err != nil {
		return DiagnosticSummary{}, fmt.Errorf("read schema version: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&result.PageCount); err != nil {
		return DiagnosticSummary{}, fmt.Errorf("read database page count: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&result.PageSize); err != nil {
		return DiagnosticSummary{}, fmt.Errorf("read database page size: %w", err)
	}

	// These names are fixed source constants, never caller-controlled values.
	for _, table := range []string{
		"rooms", "members", "invites", "queue", "playback_history",
		"security_audit", "schema_migrations", "update_state",
	} {
		var count int64
		if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			return DiagnosticSummary{}, fmt.Errorf("count %s: %w", table, err)
		}
		result.TableCounts[table] = count
	}
	return result, nil
}
