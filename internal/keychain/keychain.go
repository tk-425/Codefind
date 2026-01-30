package keychain

import (
	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the keychain service identifier
	ServiceName = "codefind"
	// AccountName is the keychain account identifier for auth key
	AccountName = "manager"
	// EmailAccountName is the keychain account identifier for email
	EmailAccountName = "manager-email"
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

// SetEmail stores the email in the system keychain
func SetEmail(email string) error {
	return keyring.Set(ServiceName, EmailAccountName, email)
}

// GetEmail retrieves the email from the system keychain
func GetEmail() (string, error) {
	return keyring.Get(ServiceName, EmailAccountName)
}

// DeleteEmail removes the email from the system keychain
func DeleteEmail() error {
	return keyring.Delete(ServiceName, EmailAccountName)
}

// HasEmail checks if an email is stored in the keychain
func HasEmail() bool {
	_, err := keyring.Get(ServiceName, EmailAccountName)
	return err == nil
}
