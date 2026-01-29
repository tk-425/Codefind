package keychain

import (
	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the keychain service identifier
	ServiceName = "codefind"
	// AccountName is the keychain account identifier
	AccountName = "manager"
)

// SetAuthKey stores the auth key in the system keychain
func SetAuthKey(authKey string) error {
	return keyring.Set(ServiceName, AccountName, authKey)
}

// GetAuthKey retrieves the auth key from the system keychain
func GetAuthKey() (string, error) {
	return keyring.Get(ServiceName, AccountName)
}

// DeleteAuthKey removes the auth key from the system keychain
func DeleteAuthKey() error {
	return keyring.Delete(ServiceName, AccountName)
}

// HasAuthKey checks if an auth key is stored in the keychain
func HasAuthKey() bool {
	_, err := keyring.Get(ServiceName, AccountName)
	return err == nil
}
