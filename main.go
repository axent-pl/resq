// main.go (skrótowo)
package main

// - kazdy widzi tylko swoje raporty
// - wybrani widza wiecej (zarzad)
// + eksport do Excel/CSV
// - publikacja
// - logowanie przez konto google

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/axent-pl/resq/utils"
	"github.com/go-webauthn/webauthn/webauthn"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", listReportsHandler)
	mux.HandleFunc("/reports", listReportsHandler)
	mux.HandleFunc("/reports/new", createReportHandler)
	mux.HandleFunc("/reports/view/{id}", viewReportHandler)
	mux.HandleFunc("/reports/history/{id}", historyHandler)
	mux.HandleFunc("/reports/edit/{id}", editReportHandler)
	mux.HandleFunc("/reports/version/{id}", newVersionHandler)
	mux.HandleFunc("/reports/export", exportReportsHandler)
	log.Fatal(http.ListenAndServe(listenAddr(), sessionMiddleware(passkeyMiddleware(mux))))
}

func passkeyMiddleware(next http.Handler) http.Handler {
	webAuthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: envString("RP_DISPLAY_NAME", "RESQ"),
		RPID:          envString("RP_ID", "localhost"),
		RPOrigins:     envList("RP_ORIGINS", []string{"http://localhost:1234"}),
	})
	if err != nil {
		log.Fatal(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := SessionFromRequest(r)
		if strings.HasPrefix(r.URL.Path, "/passkey") {
			handlePasskey(webAuthn, w, r, session)
			return
		}
		if session == nil || session.Username == "" || session.Username == "anonymous" {
			if err := ExecuteTemplate(w, "passkey", utils.IsXhr(r), nil); err != nil {
				slog.Error(fmt.Sprintf("could not execute 'passkey' template: %v", err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}

func listenAddr() string {
	port := envString("PORT", "1234")
	if strings.Contains(port, ":") {
		return port
	}
	return ":" + port
}

func envString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envList(key string, fallback []string) []string {
	value := os.Getenv(key)
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
