package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"durablewindows/internal/models"
	bolt "go.etcd.io/bbolt"
)

// Store manages persistent storage using BoltDB.
type Store struct {
	mu       sync.Mutex
	db       *bolt.DB
	dbPath   string
	lockPath string
	lockFile *os.File // held open for the lifetime of the process
}

// NewStore creates or opens a BoltDB store at the given path.
// It acquires a singleton lock file to prevent multiple instances.
func NewStore(dataDir, productName, version string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, fmt.Sprintf("%s.%s.db", productName, version))
	lockPath := filepath.Join(dataDir, fmt.Sprintf("%s.db.lock", productName))

	// Try to acquire the singleton lock. If the lock file already exists,
	// check whether the owning process is still alive.
	lockFile, err := tryAcquireLock(lockPath)
	if err != nil {
		return nil, fmt.Errorf("another instance is already running (lock file %s): %w", lockPath, err)
	}

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		lockFile.Close()
		os.Remove(lockPath)
		return nil, fmt.Errorf("opening database: %w", err)
	}

	return &Store{
		db:       db,
		dbPath:   dbPath,
		lockPath: lockPath,
		lockFile: lockFile,
	}, nil
}

// Close closes the database, releases the lock, and removes the lock file.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var dbErr error
	if s.db != nil {
		dbErr = s.db.Close()
		s.db = nil
	}

	// Release singleton lock
	if s.lockFile != nil {
		s.lockFile.Close()
		s.lockFile = nil
	}
	os.Remove(s.lockPath)

	return dbErr
}

// tryAcquireLock attempts to create and lock the singleton lock file.
// If the file already exists, it checks whether the owning PID is still running.
// If the old process is dead, it removes the stale lock and creates a new one.
func tryAcquireLock(lockPath string) (*os.File, error) {
	// Try to create a new lock file exclusively
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err == nil {
		// We created the file — write our PID and return
		fmt.Fprintf(f, "%d\n", os.Getpid())
		return f, nil
	}

	// File already exists — check if the old process is still alive
	if !os.IsExist(err) {
		return nil, err
	}

	existing, readErr := os.ReadFile(lockPath)
	if readErr != nil {
		// Can't read it — remove stale lock and retry
		os.Remove(lockPath)
		f, err = os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(f, "%d\n", os.Getpid())
		return f, nil
	}

	// Parse the PID from the lock file
	var oldPid int
	if _, scanErr := fmt.Sscanf(string(existing), "%d", &oldPid); scanErr != nil {
		// Corrupt lock file — remove and retry
		os.Remove(lockPath)
		f, err = os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(f, "%d\n", os.Getpid())
		return f, nil
	}

	// Check if the old process is still running
	if oldPid > 0 && isProcessRunning(oldPid) {
		return nil, fmt.Errorf("process %d still holds the lock", oldPid)
	}

	// Old process is dead — remove stale lock and create a new one
	os.Remove(lockPath)
	f, err = os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(f, "%d\n", os.Getpid())
	return f, nil
}

// SaveWindowMetrics stores window metrics for a display key.
func (s *Store) SaveWindowMetrics(displayKey string, metrics map[uintptr][]*models.WindowMetrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(displayKey))
		if err != nil {
			return err
		}

		for hwnd, mList := range metrics {
			key := []byte(fmt.Sprintf("hwnd_%d", hwnd))
			data, err := json.Marshal(mList)
			if err != nil {
				return err
			}
			if err := bucket.Put(key, data); err != nil {
				return err
			}
		}

		return nil
	})
}

// LoadWindowMetrics loads window metrics for a display key.
func (s *Store) LoadWindowMetrics(displayKey string) (map[uintptr][]*models.WindowMetrics, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[uintptr][]*models.WindowMetrics)

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(displayKey))
		if bucket == nil {
			return nil // no data for this key
		}

		return bucket.ForEach(func(k, v []byte) error {
			var hwnd uintptr
			fmt.Sscanf(string(k), "hwnd_%d", &hwnd)

			var mList []*models.WindowMetrics
			if err := json.Unmarshal(v, &mList); err != nil {
				return nil // skip corrupted entries
			}
			result[hwnd] = mList
			return nil
		})
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// ListDisplayKeys returns all display configuration keys in the database.
func (s *Store) ListDisplayKeys() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var keys []string

	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			keys = append(keys, string(name))
			return nil
		})
	})

	if err != nil {
		return nil, err
	}
	return keys, nil
}

// DisplayKeyExists checks if a display key has data in the database.
func (s *Store) DisplayKeyExists(displayKey string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var exists bool
	err := s.db.View(func(tx *bolt.Tx) error {
		exists = tx.Bucket([]byte(displayKey)) != nil
		return nil
	})
	return exists, err
}

// SaveSnapshotTimes persists snapshot capture timestamps to BoltDB.
func (s *Store) SaveSnapshotTimes(times map[string]map[int]time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("_snapshot_times"))
		if err != nil {
			return err
		}
		for dk, entries := range times {
			for id, t := range entries {
				key := []byte(fmt.Sprintf("%s_%d", dk, id))
				data, _ := t.MarshalBinary()
				bucket.Put(key, data)
			}
		}
		return nil
	})
}

// LoadSnapshotTimes loads snapshot capture timestamps from BoltDB.
func (s *Store) LoadSnapshotTimes() (map[string]map[int]time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make(map[string]map[int]time.Time)
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("_snapshot_times"))
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			key := string(k)
			last := strings.LastIndexByte(key, '_')
			if last < 0 {
				return nil
			}
			dk := key[:last]
			var id int
			fmt.Sscanf(key[last+1:], "%d", &id)
			var t time.Time
			t.UnmarshalBinary(v)
			if result[dk] == nil {
				result[dk] = make(map[int]time.Time)
			}
			result[dk][id] = t
			return nil
		})
	})
	return result, err
}

// SaveParkedWindows persists the list of parked-window HWNDs to BoltDB.
// An empty list clears the bucket so the DB stays clean on normal exit.
func (s *Store) SaveParkedWindows(hwnds []uintptr) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Update(func(tx *bolt.Tx) error {
		if len(hwnds) == 0 {
			tx.DeleteBucket([]byte("_parked_windows"))
			return nil
		}
		bucket, err := tx.CreateBucketIfNotExists([]byte("_parked_windows"))
		if err != nil {
			return err
		}
		// Encode as hex strings for readability when inspecting the DB.
		encoded := make([]string, len(hwnds))
		for i, h := range hwnds {
			encoded[i] = fmt.Sprintf("%x", h)
		}
		data, _ := json.Marshal(encoded)
		return bucket.Put([]byte("entries"), data)
	})
}

// LoadParkedWindows loads the parked-window HWND list from BoltDB.
// Returns nil if the bucket doesn't exist (first run or clean shutdown).
func (s *Store) LoadParkedWindows() ([]uintptr, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []uintptr
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("_parked_windows"))
		if bucket == nil {
			return nil
		}
		data := bucket.Get([]byte("entries"))
		if data == nil {
			return nil
		}
		var encoded []string
		if err := json.Unmarshal(data, &encoded); err != nil {
			return err
		}
		for _, s := range encoded {
			var h uintptr
			fmt.Sscanf(s, "%x", &h)
			result = append(result, h)
		}
		return nil
	})
	return result, err
}
