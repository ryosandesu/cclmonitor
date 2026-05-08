package store_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/store"
)

func TestIsDuplicate_FirstCallReturnsFalse(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "dedup.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	dup, err := s.IsDuplicate("session1", "hash1", 60)
	if err != nil {
		t.Fatal(err)
	}
	if dup {
		t.Error("first call should not be duplicate")
	}
}

func TestIsDuplicate_SecondCallWithinWindowReturnsTrue(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "dedup.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if _, err := s.IsDuplicate("session1", "hash1", 60); err != nil {
		t.Fatal(err)
	}
	dup, err := s.IsDuplicate("session1", "hash1", 60)
	if err != nil {
		t.Fatal(err)
	}
	if !dup {
		t.Error("second call within window should be duplicate")
	}
}

func TestIsDuplicate_DifferentHashNotDuplicate(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "dedup.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if _, err := s.IsDuplicate("session1", "hash1", 60); err != nil {
		t.Fatal(err)
	}
	dup, err := s.IsDuplicate("session1", "hash2", 60)
	if err != nil {
		t.Fatal(err)
	}
	if dup {
		t.Error("different hash should not be duplicate")
	}
}

func TestIsDuplicate_DifferentSessionNotDuplicate(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "dedup.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if _, err := s.IsDuplicate("session1", "hash1", 60); err != nil {
		t.Fatal(err)
	}
	dup, err := s.IsDuplicate("session2", "hash1", 60)
	if err != nil {
		t.Fatal(err)
	}
	if dup {
		t.Error("different session should not be duplicate")
	}
}

func TestPurgeExpired_RemovesOldRecords(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "dedup.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// 過去に記録されたレコードを直接挿入
	past := time.Now().Add(-2 * time.Minute)
	if err := s.InsertAt("session1", "hash1", past); err != nil {
		t.Fatal(err)
	}

	// 60秒ウィンドウ外なので duplicate にならないはず
	dup, err := s.IsDuplicate("session1", "hash1", 60)
	if err != nil {
		t.Fatal(err)
	}
	if dup {
		t.Error("expired record should not be duplicate")
	}
}
