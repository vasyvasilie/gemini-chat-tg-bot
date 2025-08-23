package storage

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"go.etcd.io/bbolt"
)

const (
	defaultFileMode    os.FileMode   = 0600
	defaultTimeout     time.Duration = 1 * time.Second
	usersBucket        string        = "users"
	responseBucket     string        = "responses"
	geminiErrorsBucket string        = "geminiErrors"
	bytesInKB                        = 1024
	bytesInMB                        = bytesInKB * bytesInKB
)

type Message struct {
	Role string
	Text string
}

type ConversationHistory struct {
	Messages []Message
}

type UserSettings struct {
	UserID         int64
	ModelName      string
	FavoriteModels []string
	History        ConversationHistory
}

type Storage struct {
	db *bbolt.DB
}

var ErrUserNotFound = errors.New("user not found")

func NewStorage(path string) (*Storage, error) {
	db, err := bbolt.Open(path, defaultFileMode, &bbolt.Options{Timeout: defaultTimeout})
	if err != nil {
		return nil, err
	}

	if err := db.Update(func(tx *bbolt.Tx) error {
		var multiErr error

		buckets := []string{usersBucket, responseBucket, geminiErrorsBucket}
		for _, bucketName := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucketName)); err != nil {
				multiErr = multierror.Append(multiErr, fmt.Errorf("failed to create %s bucket: %w", bucketName, err))
			}
		}

		return multiErr
	}); err != nil {
		return nil, err
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) SaveUserSettings(userID int64, settings *UserSettings) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(usersBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", usersBucket)
		}

		data, err := json.Marshal(settings)
		if err != nil {
			return fmt.Errorf("failed to marshal user settings: %w", err)
		}

		userKey := strconv.FormatInt(userID, 10)
		if err := bucket.Put([]byte(userKey), data); err != nil {
			return fmt.Errorf("failed to save user settings for ID %s: %w", userKey, err)
		}

		return nil
	})
}

func (s *Storage) GetUserSettings(userID int64) (*UserSettings, error) {
	settings := &UserSettings{}
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(usersBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", usersBucket)
		}

		userKey := strconv.FormatInt(userID, 10)
		data := bucket.Get([]byte(userKey))
		if data == nil {
			return ErrUserNotFound
		}

		if err := json.Unmarshal(data, settings); err != nil {
			return fmt.Errorf("failed to unmarshal user settings for ID %s: %w", userKey, err)
		}

		return nil
	})

	return settings, err
}

func (s *Storage) SaveResponse(userID int64, text string) (string, error) {
	response := struct {
		UserID int64
		Text   string
	}{
		UserID: userID,
		Text:   text,
	}

	key := generateHash(userID, text)
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(responseBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", responseBucket)
		}

		data, err := json.Marshal(response)
		if err != nil {
			return fmt.Errorf("failed to marshal response: %w", err)
		}

		if err := bucket.Put([]byte(key), data); err != nil {
			return fmt.Errorf("failed to save response with key %s: %w", key, err)
		}

		return nil
	})

	return key, err
}

type ErrorLog struct {
	Timestamp   time.Time
	UserID      int64
	Model       string
	RequestText string
	Error       string
	History     []Message
}

func (s *Storage) LogGeminiError(userID int64, model, requestText, errorMsg string, history []Message) (string, error) {
	errorLog := ErrorLog{
		Timestamp:   time.Now(),
		UserID:      userID,
		RequestText: requestText,
		Error:       errorMsg,
		Model:       model,
		History:     history,
	}

	key := generateHash(userID, errorMsg)
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(geminiErrorsBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", geminiErrorsBucket)
		}

		data, err := json.Marshal(errorLog)
		if err != nil {
			return fmt.Errorf("failed to marshal error log: %w", err)
		}

		if err := bucket.Put([]byte(key), data); err != nil {
			return fmt.Errorf("failed to save error log: %w", err)
		}

		return nil
	})

	return key, err
}

func (s *Storage) GetDBSize() (float64, error) {
	fileStat, err := os.Stat(s.db.Path())
	if err != nil {
		return 0, fmt.Errorf("failed to get database file stats: %w", err)
	}
	sizeMB := float64(fileStat.Size()) / float64(bytesInMB)
	return sizeMB, nil
}

func generateHash(userID int64, text string) string {
	now := strconv.FormatInt(time.Now().UnixNano(), 10)
	userIDStr := strconv.FormatInt(userID, 10)
	md5sum := md5.Sum([]byte(strings.Join([]string{now, userIDStr, text}, "-")))
	return hex.EncodeToString(md5sum[:])
}
