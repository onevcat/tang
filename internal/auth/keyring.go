package auth

import (
	"errors"
	"fmt"
	"strings"

	keyringlib "github.com/zalando/go-keyring"
)

const (
	keyringService       = "tang"
	keyringActiveAccount = "_active"
)

var (
	ErrNotFound     = errors.New("session not found")
	ErrLocked       = errors.New("keyring locked or unavailable")
	ErrUnauthorized = errors.New("keyring unauthorized")
)

type SecretStore interface {
	Save(session *Session) error
	Load() (*Session, error)
	Clear() error
}

type KeyringStore struct{}

func (KeyringStore) Save(session *Session) error {
	encoded, err := session.Marshal()
	if err != nil {
		return err
	}
	if err := keyringlib.Set(keyringService, session.Handle, encoded); err != nil {
		return classifyKeyringError(err)
	}
	if err := keyringlib.Set(keyringService, keyringActiveAccount, session.Handle); err != nil {
		return classifyKeyringError(err)
	}
	return nil
}

func (KeyringStore) Load() (*Session, error) {
	handle, err := keyringlib.Get(keyringService, keyringActiveAccount)
	if err != nil {
		return nil, classifyKeyringError(err)
	}
	encoded, err := keyringlib.Get(keyringService, handle)
	if err != nil {
		return nil, classifyKeyringError(err)
	}
	return Unmarshal(encoded)
}

func (KeyringStore) Clear() error {
	handle, err := keyringlib.Get(keyringService, keyringActiveAccount)
	if err != nil && !errors.Is(classifyKeyringError(err), ErrNotFound) {
		return classifyKeyringError(err)
	}
	if handle != "" {
		if err := keyringlib.Delete(keyringService, handle); err != nil && !errors.Is(classifyKeyringError(err), ErrNotFound) {
			return classifyKeyringError(err)
		}
	}
	if err := keyringlib.Delete(keyringService, keyringActiveAccount); err != nil && !errors.Is(classifyKeyringError(err), ErrNotFound) {
		return classifyKeyringError(err)
	}
	return nil
}

func Save(session *Session) error {
	return KeyringStore{}.Save(session)
}

func Load() (*Session, error) {
	return KeyringStore{}.Load()
}

func Clear() error {
	return KeyringStore{}.Clear()
}

func classifyKeyringError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, keyringlib.ErrNotFound) {
		return ErrNotFound
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "locked"), strings.Contains(msg, "unavailable"):
		return fmt.Errorf("%w: %v", ErrLocked, err)
	case strings.Contains(msg, "unauthorized"), strings.Contains(msg, "denied"), strings.Contains(msg, "permission"):
		return fmt.Errorf("%w: %v", ErrUnauthorized, err)
	default:
		return err
	}
}
