package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
)

const sessionCookieName = "resq_session"

type sessionContextKey struct{}

type Session struct {
	Username string
}

var sessions = struct {
	sync.RWMutex
	values map[string]Session
}{
	values: make(map[string]Session),
}

func SessionFromRequest(r *http.Request) *Session {
	session, _ := r.Context().Value(sessionContextKey{}).(*Session)
	return session
}

func sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := ""
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			sessionID = cookie.Value
		}
		if sessionID == "" {
			sessionID = newSessionID()
			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Value:    sessionID,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}

		sessions.RLock()
		session := sessions.values[sessionID]
		sessions.RUnlock()
		if session.Username == "" {
			session.Username = "anonymous"
		}

		reqSession := &session
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionContextKey{}, reqSession)))

		sessions.Lock()
		sessions.values[sessionID] = *reqSession
		sessions.Unlock()
	})
}

func newSessionID() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
