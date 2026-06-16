package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-webauthn/webauthn/webauthn"
)

const passkeyUsersPath = "data/passkeys.json"

type passkeyUser struct {
	ID          []byte
	Username    string
	Credentials []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte                         { return u.ID }
func (u *passkeyUser) WebAuthnName() string                       { return u.Username }
func (u *passkeyUser) WebAuthnDisplayName() string                { return u.Username }
func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }

var passkeyUsers = struct {
	sync.RWMutex
	byName map[string]*passkeyUser
	byID   map[string]*passkeyUser
}{
	byName: make(map[string]*passkeyUser),
	byID:   make(map[string]*passkeyUser),
}

func init() {
	if err := loadPasskeyUsers(); err != nil {
		log.Fatalf("load passkey users: %v", err)
	}
}

func loadPasskeyUsers() error {
	f, err := os.Open(passkeyUsersPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	var users []*passkeyUser
	if err := json.NewDecoder(f).Decode(&users); err != nil && err != io.EOF {
		return err
	}

	passkeyUsers.Lock()
	defer passkeyUsers.Unlock()
	passkeyUsers.byName = make(map[string]*passkeyUser, len(users))
	passkeyUsers.byID = make(map[string]*passkeyUser, len(users))
	for _, user := range users {
		if user == nil || user.Username == "" || len(user.ID) == 0 {
			continue
		}
		passkeyUsers.byName[user.Username] = user
		passkeyUsers.byID[string(user.ID)] = user
	}
	return nil
}

func savePasskeyUsersLocked() error {
	dir := filepath.Dir(passkeyUsersPath)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(passkeyUsersPath)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	users := make([]*passkeyUser, 0, len(passkeyUsers.byName))
	for _, user := range passkeyUsers.byName {
		users = append(users, user)
	}

	enc := json.NewEncoder(tmpFile)
	enc.SetIndent("", "  ")
	if err := enc.Encode(users); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	return os.Rename(tmpFile.Name(), passkeyUsersPath)
}

func createPasskeyUser(username string) (*passkeyUser, error) {
	username = strings.TrimSpace(username)
	if username == "" || len(username) > 128 {
		return nil, errors.New("username is required")
	}

	passkeyUsers.Lock()
	defer passkeyUsers.Unlock()
	if user := passkeyUsers.byName[username]; user != nil {
		return nil, errors.New("user already exist")
	}

	id := make([]byte, 32)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	user := &passkeyUser{ID: id, Username: username}
	passkeyUsers.byName[username] = user
	passkeyUsers.byID[string(id)] = user
	if err := savePasskeyUsersLocked(); err != nil {
		delete(passkeyUsers.byName, username)
		delete(passkeyUsers.byID, string(id))
		return nil, err
	}
	return user, nil
}

func getPasskeyUserByName(username string) *passkeyUser {
	passkeyUsers.RLock()
	defer passkeyUsers.RUnlock()
	return passkeyUsers.byName[username]
}

func findPasskeyUser(_ []byte, userHandle []byte) (webauthn.User, error) {
	passkeyUsers.RLock()
	defer passkeyUsers.RUnlock()
	user := passkeyUsers.byID[string(userHandle)]
	if user == nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func addPasskeyCredential(username string, credential webauthn.Credential) {
	passkeyUsers.Lock()
	defer passkeyUsers.Unlock()
	user := passkeyUsers.byName[username]
	if user == nil {
		return
	}
	for i := range user.Credentials {
		if string(user.Credentials[i].ID) == string(credential.ID) {
			user.Credentials[i] = credential
			if err := savePasskeyUsersLocked(); err != nil {
				log.Printf("save passkey users: %v", err)
			}
			return
		}
	}
	user.Credentials = append(user.Credentials, credential)
	if err := savePasskeyUsersLocked(); err != nil {
		log.Printf("save passkey users: %v", err)
	}
}

func updatePasskeyCredential(username string, credential webauthn.Credential) {
	passkeyUsers.Lock()
	defer passkeyUsers.Unlock()
	user := passkeyUsers.byName[username]
	if user == nil {
		return
	}
	for i := range user.Credentials {
		if string(user.Credentials[i].ID) == string(credential.ID) {
			user.Credentials[i] = credential
			if err := savePasskeyUsersLocked(); err != nil {
				log.Printf("save passkey users: %v", err)
			}
			return
		}
	}
}
