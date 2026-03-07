package keychain

import (
	"errors"
	"fmt"

	keyring "github.com/zalando/go-keyring"
)

const (
	serviceName = "codefind"
	accountName = "token"
)

var ErrNotFound = keyring.ErrNotFound

type Provider interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

type liveProvider struct{}

func (liveProvider) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (liveProvider) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (liveProvider) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

type Manager struct {
	provider Provider
	service  string
	account  string
}

func ServiceName() string {
	return serviceName
}

func DefaultManager() *Manager {
	return NewManager(liveProvider{})
}

func NewManager(provider Provider) *Manager {
	return &Manager{
		provider: provider,
		service:  serviceName,
		account:  accountName,
	}
}

func (m *Manager) SaveToken(token string) error {
	if token == "" {
		return errors.New("token cannot be empty")
	}
	if err := m.provider.Set(m.service, m.account, token); err != nil {
		return fmt.Errorf("store token in keychain: %w", err)
	}
	return nil
}

func (m *Manager) LoadToken() (string, error) {
	token, err := m.provider.Get(m.service, m.account)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("load token from keychain: %w", err)
	}
	return token, nil
}

func (m *Manager) DeleteToken() error {
	err := m.provider.Delete(m.service, m.account)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("delete token from keychain: %w", err)
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrNotFound
	}
	return nil
}
