package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

type APIKey struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Provider    string    `json:"provider"`
	Key         string    `json:"key"`
	Model       string    `json:"model"`
	CustomURL   string    `json:"custom_url,omitempty"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	TotalReqs   int64     `json:"total_requests"`
	ErrorReqs   int64     `json:"error_requests"`
	TotalCost   float64   `json:"total_cost"`
	LastUsed    time.Time `json:"last_used"`
	CreatedAt   time.Time `json:"created_at"`
}

type VirtualKey struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Key           string    `json:"key"`
	OwnerID       string    `json:"owner_id"`
	MonthlyBudget float64   `json:"monthly_budget"`
	UsedThisMonth float64   `json:"used_this_month"`
	TotalReqs     int64     `json:"total_requests"`
	Enabled       bool      `json:"enabled"`
	AllowedModels string    `json:"allowed_models"`
	RateLimit     int       `json:"rate_limit"`
	CreatedAt     time.Time `json:"created_at"`
	LastUsed      time.Time `json:"last_used"`
}

type CacheEntry struct {
	ID        int64     `json:"id"`
	Hash      string    `json:"hash"`
	Response  string    `json:"response"`
	Model     string    `json:"model"`
	HitCount  int64     `json:"hit_count"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type RequestLog struct {
	ID         int64     `json:"id"`
	VirtualKey string    `json:"virtual_key"`
	Provider   string    `json:"provider"`
	Model      string    `json:"model"`
	StatusCode int       `json:"status_code"`
	TokensIn   int       `json:"tokens_in"`
	TokensOut  int       `json:"tokens_out"`
	Cost       float64   `json:"cost"`
	Latency    int64     `json:"latency_ms"`
	CacheHit   bool      `json:"cache_hit"`
	Timestamp  time.Time `json:"timestamp"`
}

type BlockedRequest struct {
	ID        int64     `json:"id"`
	Reason    string    `json:"reason"`
	Pattern   string    `json:"pattern"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type FailoverLog struct {
	ID               int64     `json:"id"`
	FromProvider     string    `json:"from_provider"`
	FromModel        string    `json:"from_model"`
	ToProvider       string    `json:"to_provider"`
	ToModel          string    `json:"to_model"`
	TriggerStatus    int       `json:"trigger_status"`
	Compacted        bool      `json:"compacted"`
	OriginalTokens   int       `json:"original_tokens"`
	CompactedTokens  int       `json:"compacted_tokens"`
	Success          bool      `json:"success"`
	Timestamp        time.Time `json:"timestamp"`
}

type KeyStats struct {
	TotalRequests   int64          `json:"total_requests"`
	ErrorRequests   int64          `json:"error_requests"`
	ProviderStats   []ProviderStat `json:"provider_stats"`
	RecentErrors    []ErrorRecord  `json:"recent_errors"`
	CacheHits       int64          `json:"cache_hits"`
	TotalCost       float64        `json:"total_cost"`
	BlockedRequests int64          `json:"blocked_requests"`
}

type ProviderStat struct {
	Provider      string  `json:"provider"`
	TotalRequests int64   `json:"total_requests"`
	ErrorRequests int64   `json:"error_requests"`
	ActiveKeys    int     `json:"active_keys"`
	TotalCost     float64 `json:"total_cost"`
}

type ErrorRecord struct {
	KeyID      int64     `json:"key_id"`
	Provider   string    `json:"provider"`
	Error      string    `json:"error"`
	StatusCode int       `json:"status_code"`
	Timestamp  time.Time `json:"timestamp"`
}

type Store struct {
	db      *sql.DB
	mu      sync.RWMutex
	keyring *Keyring
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	keyring, err := NewKeyring(dbPath)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db, keyring: keyring}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT DEFAULT '',
			provider TEXT NOT NULL,
			key TEXT NOT NULL,
			model TEXT DEFAULT '',
			custom_url TEXT DEFAULT '',
			priority INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1,
			total_requests INTEGER DEFAULT 0,
			error_requests INTEGER DEFAULT 0,
			total_cost REAL DEFAULT 0,
			last_used DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS error_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key_id INTEGER,
			provider TEXT,
			error TEXT,
			status_code INTEGER,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS virtual_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			key TEXT NOT NULL UNIQUE,
			owner_id TEXT DEFAULT '',
			monthly_budget REAL DEFAULT 0,
			used_this_month REAL DEFAULT 0,
			total_requests INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1,
			allowed_models TEXT DEFAULT '*',
			rate_limit INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_used DATETIME
		);
		CREATE TABLE IF NOT EXISTS cache_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hash TEXT NOT NULL UNIQUE,
			response TEXT NOT NULL,
			model TEXT NOT NULL,
			hit_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME
		);
		CREATE TABLE IF NOT EXISTS request_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			virtual_key TEXT DEFAULT '',
			provider TEXT DEFAULT '',
			model TEXT DEFAULT '',
			status_code INTEGER DEFAULT 0,
			tokens_in INTEGER DEFAULT 0,
			tokens_out INTEGER DEFAULT 0,
			cost REAL DEFAULT 0,
			latency_ms INTEGER DEFAULT 0,
			cache_hit INTEGER DEFAULT 0,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS blocked_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			reason TEXT DEFAULT '',
			pattern TEXT DEFAULT '',
			message TEXT DEFAULT '',
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS failover_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_provider TEXT DEFAULT '',
			from_model TEXT DEFAULT '',
			to_provider TEXT DEFAULT '',
			to_model TEXT DEFAULT '',
			trigger_status INTEGER DEFAULT 0,
			compacted INTEGER DEFAULT 0,
			original_tokens INTEGER DEFAULT 0,
			compacted_tokens INTEGER DEFAULT 0,
			success INTEGER DEFAULT 1,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM settings WHERE key = 'unified_api_key'").Scan(&count)
	if count == 0 {
		key := generateKey()
		encrypted := s.keyring.Encrypt(key)
		s.db.Exec("INSERT INTO settings (key, value) VALUES ('unified_api_key', ?)", encrypted)
	}

	s.db.Exec("ALTER TABLE api_keys ADD COLUMN name TEXT DEFAULT ''")
	s.db.Exec("ALTER TABLE api_keys ADD COLUMN model TEXT DEFAULT ''")
	s.db.Exec("ALTER TABLE api_keys ADD COLUMN total_cost REAL DEFAULT 0")

	return nil
}

func generateKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "gw-" + hex.EncodeToString(b)
}

func generateVirtualKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "vk-" + hex.EncodeToString(b)
}

func (s *Store) GetUnifiedKey() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var val string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = 'unified_api_key'").Scan(&val)
	if err != nil {
		return "", err
	}
	return s.keyring.Decrypt(val), nil
}

func (s *Store) ValidateUnifiedKey(key string) bool {
	stored, err := s.GetUnifiedKey()
	if err != nil {
		return false
	}
	return stored == key
}

func (s *Store) RegenerateUnifiedKey() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	newKey := generateKey()
	encrypted := s.keyring.Encrypt(newKey)
	_, err := s.db.Exec("UPDATE settings SET value = ? WHERE key = 'unified_api_key'", encrypted)
	return newKey, err
}

func (s *Store) AddKey(name, provider, key, model, customURL string, priority int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	encrypted := s.keyring.Encrypt(key)
	result, err := s.db.Exec(
		"INSERT INTO api_keys (name, provider, key, model, custom_url, priority, enabled) VALUES (?, ?, ?, ?, ?, ?, 1)",
		name, provider, encrypted, model, customURL, priority,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) GetKeys() ([]APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT id, name, provider, key, model, custom_url, priority, enabled, total_requests, error_requests, total_cost, last_used, created_at FROM api_keys ORDER BY priority DESC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []APIKey
	for rows.Next() {
		var k APIKey
		var lastUsed, createdAt sql.NullTime
		err := rows.Scan(&k.ID, &k.Name, &k.Provider, &k.Key, &k.Model, &k.CustomURL, &k.Priority, &k.Enabled, &k.TotalReqs, &k.ErrorReqs, &k.TotalCost, &lastUsed, &createdAt)
		if err != nil {
			return nil, err
		}
		k.Key = s.keyring.Decrypt(k.Key)
		if lastUsed.Valid {
			k.LastUsed = lastUsed.Time
		}
		if createdAt.Valid {
			k.CreatedAt = createdAt.Time
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *Store) GetKeysMasked() ([]APIKey, error) {
	keys, err := s.GetKeys()
	if err != nil {
		return nil, err
	}
	for i := range keys {
		keys[i].Key = MaskKey(keys[i].Key)
	}
	return keys, nil
}

func (s *Store) GetKeysByProvider(provider string) ([]APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(
		"SELECT id, name, provider, key, model, custom_url, priority, enabled, total_requests, error_requests, total_cost, last_used, created_at FROM api_keys WHERE provider = ? AND enabled = 1 ORDER BY priority DESC, id ASC",
		provider,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []APIKey
	for rows.Next() {
		var k APIKey
		var lastUsed, createdAt sql.NullTime
		err := rows.Scan(&k.ID, &k.Name, &k.Provider, &k.Key, &k.Model, &k.CustomURL, &k.Priority, &k.Enabled, &k.TotalReqs, &k.ErrorReqs, &k.TotalCost, &lastUsed, &createdAt)
		if err != nil {
			return nil, err
		}
		k.Key = s.keyring.Decrypt(k.Key)
		if lastUsed.Valid {
			k.LastUsed = lastUsed.Time
		}
		if createdAt.Valid {
			k.CreatedAt = createdAt.Time
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *Store) GetKeyByName(name string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var k APIKey
	var lastUsed, createdAt sql.NullTime
	err := s.db.QueryRow(
		"SELECT id, name, provider, key, model, custom_url, priority, enabled, total_requests, error_requests, total_cost, last_used, created_at FROM api_keys WHERE name = ? AND enabled = 1", name,
	).Scan(&k.ID, &k.Name, &k.Provider, &k.Key, &k.Model, &k.CustomURL, &k.Priority, &k.Enabled, &k.TotalReqs, &k.ErrorReqs, &k.TotalCost, &lastUsed, &createdAt)
	if err != nil {
		return nil, err
	}
	k.Key = s.keyring.Decrypt(k.Key)
	if lastUsed.Valid {
		k.LastUsed = lastUsed.Time
	}
	if createdAt.Valid {
		k.CreatedAt = createdAt.Time
	}
	return &k, nil
}

func (s *Store) DeleteKey(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("DELETE FROM api_keys WHERE id = ?", id)
	return err
}

func (s *Store) ToggleKey(id int64, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("UPDATE api_keys SET enabled = ? WHERE id = ?", enabled, id)
	return err
}

func (s *Store) UpdatePriority(id int64, priority int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("UPDATE api_keys SET priority = ? WHERE id = ?", priority, id)
	return err
}

func (s *Store) RecordRequest(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Exec("UPDATE api_keys SET total_requests = total_requests + 1, last_used = CURRENT_TIMESTAMP WHERE id = ?", id)
}

func (s *Store) RecordCost(id int64, cost float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Exec("UPDATE api_keys SET total_cost = total_cost + ? WHERE id = ?", cost, id)
}

func (s *Store) RecordError(id int64, provider, errMsg string, statusCode int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	s.db.Exec("UPDATE api_keys SET error_requests = error_requests + 1 WHERE id = ?", id)
	s.db.Exec("INSERT INTO error_log (key_id, provider, error, status_code) VALUES (?, ?, ?, ?)", id, provider, errMsg, statusCode)
}

func (s *Store) GetStats() (*KeyStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &KeyStats{}

	s.db.QueryRow("SELECT COALESCE(SUM(total_requests), 0), COALESCE(SUM(error_requests), 0), COALESCE(SUM(total_cost), 0) FROM api_keys").Scan(&stats.TotalRequests, &stats.ErrorRequests, &stats.TotalCost)
	s.db.QueryRow("SELECT COUNT(*) FROM cache_entries").Scan(&stats.CacheHits)
	s.db.QueryRow("SELECT COUNT(*) FROM blocked_requests").Scan(&stats.BlockedRequests)

	rows, err := s.db.Query("SELECT provider, SUM(total_requests), SUM(error_requests), COUNT(CASE WHEN enabled = 1 THEN 1 END), COALESCE(SUM(total_cost), 0) FROM api_keys GROUP BY provider")
	if err != nil {
		return stats, nil
	}
	defer rows.Close()
	for rows.Next() {
		var ps ProviderStat
		rows.Scan(&ps.Provider, &ps.TotalRequests, &ps.ErrorRequests, &ps.ActiveKeys, &ps.TotalCost)
		stats.ProviderStats = append(stats.ProviderStats, ps)
	}

	errRows, err := s.db.Query("SELECT key_id, provider, error, status_code, timestamp FROM error_log ORDER BY timestamp DESC LIMIT 50")
	if err != nil {
		return stats, nil
	}
	defer errRows.Close()
	for errRows.Next() {
		var er ErrorRecord
		errRows.Scan(&er.KeyID, &er.Provider, &er.Error, &er.StatusCode, &er.Timestamp)
		stats.RecentErrors = append(stats.RecentErrors, er)
	}

	return stats, nil
}

func (s *Store) GetStatsJSON() (string, error) {
	stats, err := s.GetStats()
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(stats)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Store) GetCacheHits() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var total int64
	s.db.QueryRow("SELECT COALESCE(SUM(hit_count), 0) FROM cache_entries").Scan(&total)
	return total
}

func (s *Store) Close() error {
	return s.db.Close()
}

var validProviders = map[string]bool{
	"openai": true, "anthropic": true, "gemini": true, "mistral": true,
	"groq": true, "together": true, "deepseek": true, "openrouter": true,
	"fireworks": true, "nvidia": true, "custom": true,
}

func ValidateProvider(p string) bool {
	return validProviders[p]
}

func MaskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-3:]
}

func (s *Store) GetKeyByID(id int64) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var k APIKey
	var lastUsed, createdAt sql.NullTime
	err := s.db.QueryRow(
		"SELECT id, name, provider, key, model, custom_url, priority, enabled, total_requests, error_requests, total_cost, last_used, created_at FROM api_keys WHERE id = ?", id,
	).Scan(&k.ID, &k.Name, &k.Provider, &k.Key, &k.Model, &k.CustomURL, &k.Priority, &k.Enabled, &k.TotalReqs, &k.ErrorReqs, &k.TotalCost, &lastUsed, &createdAt)
	if err != nil {
		return nil, err
	}
	if lastUsed.Valid {
		k.LastUsed = lastUsed.Time
	}
	if createdAt.Valid {
		k.CreatedAt = createdAt.Time
	}
	return &k, nil
}

func (s *Store) UpdateKey(id int64, name, provider, key, model, customURL string, priority int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		"UPDATE api_keys SET name = ?, provider = ?, key = ?, model = ?, custom_url = ?, priority = ? WHERE id = ?",
		name, provider, key, model, customURL, priority, id,
	)
	return err
}

func (s *Store) ResetStats() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("UPDATE api_keys SET total_requests = 0, error_requests = 0, total_cost = 0")
	return err
}

func (s *Store) ExportConfig() (string, error) {
	keys, err := s.GetKeys()
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Store) ImportConfig(jsonStr string) (int, error) {
	var keys []APIKey
	if err := json.Unmarshal([]byte(jsonStr), &keys); err != nil {
		return 0, fmt.Errorf("invalid JSON: %w", err)
	}
	count := 0
	for _, k := range keys {
		if k.Provider == "" || k.Key == "" {
			continue
		}
		_, err := s.AddKey(k.Name, k.Provider, k.Key, k.Model, k.CustomURL, k.Priority)
		if err != nil {
			continue
		}
		count++
	}
	return count, nil
}

func (s *Store) AddVirtualKey(name, ownerID string, budget float64, allowedModels string, rateLimit int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := generateVirtualKey()
	_, err := s.db.Exec(
		"INSERT INTO virtual_keys (name, key, owner_id, monthly_budget, allowed_models, rate_limit, enabled) VALUES (?, ?, ?, ?, ?, ?, 1)",
		name, key, ownerID, budget, allowedModels, rateLimit,
	)
	if err != nil {
		return "", err
	}
	return key, nil
}

func (s *Store) GetVirtualKeys() ([]VirtualKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT id, name, key, owner_id, monthly_budget, used_this_month, total_requests, enabled, allowed_models, rate_limit, created_at, last_used FROM virtual_keys ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []VirtualKey
	for rows.Next() {
		var k VirtualKey
		var createdAt, lastUsed sql.NullTime
		err := rows.Scan(&k.ID, &k.Name, &k.Key, &k.OwnerID, &k.MonthlyBudget, &k.UsedThisMonth, &k.TotalReqs, &k.Enabled, &k.AllowedModels, &k.RateLimit, &createdAt, &lastUsed)
		if err != nil {
			return nil, err
		}
		if createdAt.Valid {
			k.CreatedAt = createdAt.Time
		}
		if lastUsed.Valid {
			k.LastUsed = lastUsed.Time
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *Store) GetVirtualKeysMasked() ([]VirtualKey, error) {
	keys, err := s.GetVirtualKeys()
	if err != nil {
		return nil, err
	}
	for i := range keys {
		keys[i].Key = MaskKey(keys[i].Key)
	}
	return keys, nil
}

func (s *Store) ValidateVirtualKey(key string) (*VirtualKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var k VirtualKey
	var createdAt, lastUsed sql.NullTime
	err := s.db.QueryRow(
		"SELECT id, name, key, owner_id, monthly_budget, used_this_month, total_requests, enabled, allowed_models, rate_limit, created_at, last_used FROM virtual_keys WHERE key = ? AND enabled = 1", key,
	).Scan(&k.ID, &k.Name, &k.Key, &k.OwnerID, &k.MonthlyBudget, &k.UsedThisMonth, &k.TotalReqs, &k.Enabled, &k.AllowedModels, &k.RateLimit, &createdAt, &lastUsed)
	if err != nil {
		return nil, err
	}
	if createdAt.Valid {
		k.CreatedAt = createdAt.Time
	}
	if lastUsed.Valid {
		k.LastUsed = lastUsed.Time
	}
	return &k, nil
}

func (s *Store) CanUseModel(vk *VirtualKey, model string) bool {
	if vk.AllowedModels == "*" || vk.AllowedModels == "" {
		return true
	}
	allowed := strings.Split(vk.AllowedModels, ",")
	for _, a := range allowed {
		a = strings.TrimSpace(a)
		if a == model || strings.HasPrefix(model, a+"/") {
			return true
		}
	}
	return false
}

func (s *Store) RecordVirtualKeyUsage(key string, cost float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Exec("UPDATE virtual_keys SET used_this_month = used_this_month + ?, total_requests = total_requests + 1, last_used = CURRENT_TIMESTAMP WHERE key = ?", cost, key)
}

func (s *Store) DeleteVirtualKey(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("DELETE FROM virtual_keys WHERE id = ?", id)
	return err
}

func (s *Store) ToggleVirtualKey(id int64, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("UPDATE virtual_keys SET enabled = ? WHERE id = ?", enabled, id)
	return err
}

func (s *Store) ResetMonthlyUsage() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("UPDATE virtual_keys SET used_this_month = 0")
	return err
}

func (s *Store) GetVirtualKeyByID(id int64) (*VirtualKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var k VirtualKey
	var createdAt, lastUsed sql.NullTime
	err := s.db.QueryRow(
		"SELECT id, name, key, owner_id, monthly_budget, used_this_month, total_requests, enabled, allowed_models, rate_limit, created_at, last_used FROM virtual_keys WHERE id = ?", id,
	).Scan(&k.ID, &k.Name, &k.Key, &k.OwnerID, &k.MonthlyBudget, &k.UsedThisMonth, &k.TotalReqs, &k.Enabled, &k.AllowedModels, &k.RateLimit, &createdAt, &lastUsed)
	if err != nil {
		return nil, err
	}
	if createdAt.Valid {
		k.CreatedAt = createdAt.Time
	}
	if lastUsed.Valid {
		k.LastUsed = lastUsed.Time
	}
	return &k, nil
}

func (s *Store) UpdateVirtualKey(id int64, name, ownerID string, budget float64, allowedModels string, rateLimit int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		"UPDATE virtual_keys SET name = ?, owner_id = ?, monthly_budget = ?, allowed_models = ?, rate_limit = ? WHERE id = ?",
		name, ownerID, budget, allowedModels, rateLimit, id,
	)
	return err
}

func (s *Store) GetCacheByHash(hash string) (*CacheEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var c CacheEntry
	var createdAt, expiresAt sql.NullTime
	err := s.db.QueryRow(
		"SELECT id, hash, response, model, hit_count, created_at, expires_at FROM cache_entries WHERE hash = ? AND (expires_at IS NULL OR expires_at > datetime('now'))", hash,
	).Scan(&c.ID, &c.Hash, &c.Response, &c.Model, &c.HitCount, &createdAt, &expiresAt)
	if err != nil {
		return nil, err
	}
	if createdAt.Valid {
		c.CreatedAt = createdAt.Time
	}
	if expiresAt.Valid {
		c.ExpiresAt = expiresAt.Time
	}
	return &c, nil
}

func (s *Store) SetCache(hash, response, model string, ttlMinutes int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO cache_entries (hash, response, model, hit_count, expires_at) VALUES (?, ?, ?, 0, datetime('now', '+' || ? || ' minutes'))",
		hash, response, model, ttlMinutes,
	)
	return err
}

func (s *Store) IncrementCacheHit(hash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Exec("UPDATE cache_entries SET hit_count = hit_count + 1 WHERE hash = ?", hash)
}

func (s *Store) ClearExpiredCache() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("DELETE FROM cache_entries WHERE expires_at IS NOT NULL AND expires_at < datetime('now')")
	return err
}

func (s *Store) LogRequest(virtualKey, provider, model string, statusCode, tokensIn, tokensOut int, cost float64, latencyMs int64, cacheHit bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cacheHitInt := 0
	if cacheHit {
		cacheHitInt = 1
	}
	s.db.Exec(
		"INSERT INTO request_log (virtual_key, provider, model, status_code, tokens_in, tokens_out, cost, latency_ms, cache_hit) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		virtualKey, provider, model, statusCode, tokensIn, tokensOut, cost, latencyMs, cacheHitInt,
	)
}

func (s *Store) GetRequestLogs(limit int) ([]RequestLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT id, virtual_key, provider, model, status_code, tokens_in, tokens_out, cost, latency_ms, cache_hit, timestamp FROM request_log ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []RequestLog
	for rows.Next() {
		var l RequestLog
		var timestamp sql.NullTime
		var cacheHitInt int
		err := rows.Scan(&l.ID, &l.VirtualKey, &l.Provider, &l.Model, &l.StatusCode, &l.TokensIn, &l.TokensOut, &l.Cost, &l.Latency, &cacheHitInt, &timestamp)
		if err != nil {
			return nil, err
		}
		l.CacheHit = cacheHitInt == 1
		if timestamp.Valid {
			l.Timestamp = timestamp.Time
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (s *Store) LogBlockedRequest(reason, pattern, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(message) > 500 {
		message = message[:500]
	}
	s.db.Exec("INSERT INTO blocked_requests (reason, pattern, message) VALUES (?, ?, ?)", reason, pattern, message)
}

func (s *Store) GetBlockedRequests(limit int) ([]BlockedRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT id, reason, pattern, message, timestamp FROM blocked_requests ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var blocked []BlockedRequest
	for rows.Next() {
		var b BlockedRequest
		var timestamp sql.NullTime
		err := rows.Scan(&b.ID, &b.Reason, &b.Pattern, &b.Message, &timestamp)
		if err != nil {
			return nil, err
		}
		if timestamp.Valid {
			b.Timestamp = timestamp.Time
		}
		blocked = append(blocked, b)
	}
	return blocked, nil
}

func (s *Store) AddAlias(name, targetModel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)",
		"alias_"+name, targetModel,
	)
	return err
}

func (s *Store) GetAlias(name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var val string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", "alias_"+name).Scan(&val)
	return val, err
}

func (s *Store) GetAliases() (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT key, value FROM settings WHERE key LIKE 'alias_%'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	aliases := make(map[string]string)
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		aliases[strings.TrimPrefix(k, "alias_")] = v
	}
	return aliases, nil
}

func (s *Store) DeleteAlias(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("DELETE FROM settings WHERE key = ?", "alias_"+name)
	return err
}

func (s *Store) AddGuardrail(name, pattern, action string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)",
		"guardrail_"+name, fmt.Sprintf("%s:%s", action, pattern),
	)
	return err
}

func (s *Store) GetGuardrails() (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query("SELECT key, value FROM settings WHERE key LIKE 'guardrail_%'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	guardrails := make(map[string]string)
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		guardrails[strings.TrimPrefix(k, "guardrail_")] = v
	}
	return guardrails, nil
}

func (s *Store) DeleteGuardrail(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("DELETE FROM settings WHERE key = ?", "guardrail_"+name)
	return err
}

func (s *Store) GetFailoverConfig() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	config := map[string]string{
		"enabled":              "false",
		"compact":              "false",
		"compact_model":        "",
		"provider":             "",
		"model":                "",
		"triggers":             "402",
	}
	rows, err := s.db.Query("SELECT key, value FROM settings WHERE key LIKE 'failover_%'")
	if err != nil {
		return config
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		config[strings.TrimPrefix(k, "failover_")] = v
	}
	return config
}

func (s *Store) SetFailoverConfig(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)",
		"failover_"+key, value,
	)
	return err
}

func (s *Store) LogFailover(fromProvider, fromModel, toProvider, toModel string, triggerStatus int, compacted bool, origTokens, compTokens int, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	compactedInt := 0
	if compacted {
		compactedInt = 1
	}
	successInt := 0
	if success {
		successInt = 1
	}
	s.db.Exec(
		"INSERT INTO failover_log (from_provider, from_model, to_provider, to_model, trigger_status, compacted, original_tokens, compacted_tokens, success) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		fromProvider, fromModel, toProvider, toModel, triggerStatus, compactedInt, origTokens, compTokens, successInt,
	)
}

func (s *Store) GetFailoverLogs(limit int) []FailoverLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query("SELECT id, from_provider, from_model, to_provider, to_model, trigger_status, compacted, original_tokens, compacted_tokens, success, timestamp FROM failover_log ORDER BY id DESC LIMIT ?", limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var logs []FailoverLog
	for rows.Next() {
		var l FailoverLog
		var compacted, success int
		var ts time.Time
		rows.Scan(&l.ID, &l.FromProvider, &l.FromModel, &l.ToProvider, &l.ToModel, &l.TriggerStatus, &compacted, &l.OriginalTokens, &l.CompactedTokens, &success, &ts)
		l.Compacted = compacted == 1
		l.Success = success == 1
		l.Timestamp = ts
		logs = append(logs, l)
	}
	return logs
}
