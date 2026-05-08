package store

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS dedup (
			session_id TEXT NOT NULL,
			input_hash TEXT NOT NULL,
			seen_at    INTEGER NOT NULL,
			PRIMARY KEY (session_id, input_hash)
		)
	`); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// IsDuplicate checks whether (sessionID, inputHash) was seen within windowSec seconds.
// If not, it records the entry and returns false.
func (s *Store) IsDuplicate(sessionID, inputHash string, windowSec int) (bool, error) {
	cutoff := time.Now().Add(-time.Duration(windowSec) * time.Second).Unix()

	var seenAt int64
	err := s.db.QueryRow(
		`SELECT seen_at FROM dedup WHERE session_id = ? AND input_hash = ?`,
		sessionID, inputHash,
	).Scan(&seenAt)

	if err == nil {
		// レコードあり：ウィンドウ内かチェック
		if seenAt >= cutoff {
			return true, nil
		}
		// 期限切れ → 更新して false
		_, err = s.db.Exec(
			`UPDATE dedup SET seen_at = ? WHERE session_id = ? AND input_hash = ?`,
			time.Now().Unix(), sessionID, inputHash,
		)
		return false, err
	}
	if err != sql.ErrNoRows {
		return false, err
	}

	// レコードなし → 新規挿入
	_, err = s.db.Exec(
		`INSERT INTO dedup (session_id, input_hash, seen_at) VALUES (?, ?, ?)`,
		sessionID, inputHash, time.Now().Unix(),
	)
	return false, err
}

// InsertAt inserts a record with a custom timestamp (for testing).
func (s *Store) InsertAt(sessionID, inputHash string, at time.Time) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO dedup (session_id, input_hash, seen_at) VALUES (?, ?, ?)`,
		sessionID, inputHash, at.Unix(),
	)
	return err
}

// PurgeExpired deletes records older than windowSec seconds.
func (s *Store) PurgeExpired(windowSec int) error {
	cutoff := time.Now().Add(-time.Duration(windowSec) * time.Second).Unix()
	_, err := s.db.Exec(`DELETE FROM dedup WHERE seen_at < ?`, cutoff)
	return err
}
