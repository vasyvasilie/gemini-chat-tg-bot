package storage

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"time"

	"go.etcd.io/bbolt"
)

type Message struct {
	Role string
	Text string
}

type ConversationHistory struct {
	Messages []Message
}

type UserSettings struct {
	UserID    int64
	ModelName string
	History   ConversationHistory
}

type Storage struct {
	db *bbolt.DB
}

var ErrUserNotFound = errors.New("user not found")

func NewStorage(path string) (*Storage, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	for _, bucket := range []string{"users", "responses"} {
		if err = db.Update(func(tx *bbolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			return err
		}); err != nil {
			return nil, err
		}
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) SaveUserSettings(userID int64, settings *UserSettings) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		data, err := json.Marshal(settings)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(strconv.FormatInt(userID, 10)), data)
	})
}

func (s *Storage) GetUserSettings(userID int64) (*UserSettings, error) {
	settings := &UserSettings{}
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		data := bucket.Get([]byte(strconv.FormatInt(userID, 10)))
		if data == nil {
			return ErrUserNotFound
		}
		return json.Unmarshal(data, settings)
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

	md5sum := md5.Sum([]byte(strconv.FormatInt(userID, 10) + text))
	key := hex.EncodeToString(md5sum[:])
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("responses"))
		data, err := json.Marshal(response)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(key), data)
	})
	return key, err
}

func (s *Storage) GetDBSize() (float64, error) {
	fileStat, err := os.Stat(s.db.Path())
	if err != nil {
		return 0, err
	}
	size := float64(fileStat.Size()) / 1024 / 1024
	return size, nil
}
