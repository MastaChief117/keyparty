// Copyright 2026 KeyParty Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package store

import (
	"database/sql"
	"fmt"
	"time"
)

// ── COST LIMITS ───────────────────────────────────────────────────────────

type CostLimit struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Period      string    `json:"period"`
	MaxCost     float64   `json:"max_cost"`
	CurrentCost float64   `json:"current_cost"`
	Provider    string    `json:"provider"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Store) AddCostLimit(name, period string, maxCost float64, provider string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, err := s.db.Exec(
		"INSERT INTO cost_limits (name, period, max_cost, provider, enabled) VALUES (?, ?, ?, ?, 1)",
		name, period, maxCost, provider,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) GetCostLimits() ([]CostLimit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT id, name, period, max_cost, current_cost, provider, enabled, created_at FROM cost_limits ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var limits []CostLimit
	for rows.Next() {
		var l CostLimit
		var createdAt sql.NullTime
		var enabled int
		rows.Scan(&l.ID, &l.Name, &l.Period, &l.MaxCost, &l.CurrentCost, &l.Provider, &enabled, &createdAt)
		l.Enabled = enabled == 1
		if createdAt.Valid {
			l.CreatedAt = createdAt.Time
		}
		limits = append(limits, l)
	}
	return limits, nil
}

func (s *Store) DeleteCostLimit(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("DELETE FROM cost_limits WHERE id = ?", id)
	return err
}

// ── PROVIDER UPTIME ──────────────────────────────────────────────────────

type ProviderUptime struct {
	Provider   string  `json:"provider"`
	TotalReqs  int64   `json:"total_requests"`
	FailedReqs int64   `json:"failed_requests"`
	AvgLatency float64 `json:"avg_latency"`
}

func (s *Store) GetProviderUptime(days int) ([]ProviderUptime, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if days <= 0 {
		days = 7
	}
	rows, err := s.db.Query(`
		SELECT provider, 
			COUNT(*) as total,
			COUNT(CASE WHEN status_code >= 400 THEN 1 END) as failed,
			COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM request_log 
		WHERE timestamp >= datetime('now', ?)
		GROUP BY provider
		ORDER BY provider`,
		fmt.Sprintf("-%d days", days),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var uptimes []ProviderUptime
	for rows.Next() {
		var u ProviderUptime
		rows.Scan(&u.Provider, &u.TotalReqs, &u.FailedReqs, &u.AvgLatency)
		uptimes = append(uptimes, u)
	}
	return uptimes, nil
}

// ── CUSTOM MODELS ────────────────────────────────────────────────────────

type CustomModel struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Provider    string    `json:"provider"`
	DisplayName string    `json:"display_name"`
	ContextSize int       `json:"context_size"`
	InputPrice  float64   `json:"input_price"`
	OutputPrice float64   `json:"output_price"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Store) AddCustomModel(name, provider, displayName string, contextSize int, inputPrice, outputPrice float64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, err := s.db.Exec(
		"INSERT INTO custom_models (name, provider, display_name, context_size, input_price, output_price) VALUES (?, ?, ?, ?, ?, ?)",
		name, provider, displayName, contextSize, inputPrice, outputPrice,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) GetCustomModels() ([]CustomModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT id, name, provider, display_name, context_size, input_price, output_price, created_at FROM custom_models ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var models []CustomModel
	for rows.Next() {
		var m CustomModel
		var createdAt sql.NullTime
		rows.Scan(&m.ID, &m.Name, &m.Provider, &m.DisplayName, &m.ContextSize, &m.InputPrice, &m.OutputPrice, &createdAt)
		if createdAt.Valid {
			m.CreatedAt = createdAt.Time
		}
		models = append(models, m)
	}
	return models, nil
}

func (s *Store) DeleteCustomModel(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("DELETE FROM custom_models WHERE id = ?", id)
	return err
}

// ── MIGRATE NEW TABLES ──────────────────────────────────────────────────

func (s *Store) migrateExtraTables() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS cost_limits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			period TEXT DEFAULT 'daily',
			max_cost REAL DEFAULT 0,
			current_cost REAL DEFAULT 0,
			provider TEXT DEFAULT '',
			enabled INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS custom_models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			provider TEXT NOT NULL,
			display_name TEXT DEFAULT '',
			context_size INTEGER DEFAULT 0,
			input_price REAL DEFAULT 0,
			output_price REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}
