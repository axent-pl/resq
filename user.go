package main

import (
	"log"
	"log/slog"
	"slices"

	"github.com/axent-pl/resq/storage"
)

type UserInfo struct {
	Username string   `json:"Username"`
	Roles    []string `json:"Roles"`
}

type UserService struct {
	store *storage.Storage[string, UserInfo]
}

func NewUserService(path string) (*UserService, error) {
	store, err := storage.NewStorage[string, UserInfo](path)
	if err != nil {
		return nil, err
	}
	r := &UserService{
		store: store,
	}
	return r, nil
}

func (s *UserService) HasRole(username, role string) bool {
	u, err := s.store.Read(username)
	if err != nil {
		return false
	}
	slog.Info("checking user roles", "user", u, "role", role, "contains", slices.Contains(u.Roles, role))
	return slices.Contains(u.Roles, role)
}

var userService *UserService

func init() {
	var err error
	userService, err = NewUserService("data/users.json")
	if err != nil {
		log.Fatalf("user service init error: %v", err)
	}
}
