package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrMissingPDS = errors.New("session PDS is required")

type Session struct {
	DID        string    `json:"did"`
	Handle     string    `json:"handle"`
	PDS        string    `json:"pds"`
	AccessJWT  string    `json:"accessJwt"`
	RefreshJWT string    `json:"refreshJwt"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

func NewSession(did, handle, pds, accessJWT, refreshJWT string) (*Session, error) {
	if strings.TrimSpace(pds) == "" {
		return nil, ErrMissingPDS
	}
	return &Session{
		DID:        did,
		Handle:     handle,
		PDS:        strings.TrimRight(pds, "/"),
		AccessJWT:  accessJWT,
		RefreshJWT: refreshJWT,
		ExpiresAt:  jwtExpiry(accessJWT),
	}, nil
}

func (s *Session) Validate() error {
	if strings.TrimSpace(s.PDS) == "" {
		return ErrMissingPDS
	}
	if s.Handle == "" || s.DID == "" {
		return fmt.Errorf("session is missing identity")
	}
	return nil
}

func (s *Session) Marshal() (string, error) {
	if err := s.Validate(); err != nil {
		return "", err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func Unmarshal(data string) (*Session, error) {
	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}
	if err := session.Validate(); err != nil {
		return nil, err
	}
	return &session, nil
}

func jwtExpiry(token string) time.Time {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}
	}
	return time.Unix(claims.Exp, 0)
}
