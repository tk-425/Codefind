package keychain

import (
	"errors"
	"testing"
)

type fakeProvider struct {
	token string
	err   error
}

func (f *fakeProvider) Set(_, _, password string) error {
	if f.err != nil {
		return f.err
	}
	f.token = password
	return nil
}

func (f *fakeProvider) Get(_, _ string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if f.token == "" {
		return "", ErrNotFound
	}
	return f.token, nil
}

func (f *fakeProvider) Delete(_, _ string) error {
	if f.err != nil {
		return f.err
	}
	if f.token == "" {
		return ErrNotFound
	}
	f.token = ""
	return nil
}

func TestManagerTokenLifecycle(t *testing.T) {
	t.Parallel()

	manager := NewManager(&fakeProvider{})
	if err := manager.SaveToken("token-123"); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	got, err := manager.LoadToken()
	if err != nil {
		t.Fatalf("LoadToken() error = %v", err)
	}
	if got != "token-123" {
		t.Fatalf("LoadToken() = %q, want %q", got, "token-123")
	}

	if err := manager.DeleteToken(); err != nil {
		t.Fatalf("DeleteToken() error = %v", err)
	}

	_, err = manager.LoadToken()
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("LoadToken() after delete error = %v, want ErrNotFound", err)
	}
}
