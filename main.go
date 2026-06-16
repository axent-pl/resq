// main.go (skrótowo)
package main

// - kazdy widzi tylko swoje raporty
// - wybrani widza wiecej (zarzad)
// + eksport do Excel/CSV
// - publikacja
// - logowanie przez konto google

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
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
	log.Fatal(http.ListenAndServe(":1234", sessionMiddleware(passkeyMiddleware(mux))))
}

func passkeyMiddleware(next http.Handler) http.Handler {
	webAuthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "RESQ",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:1234"},
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

func handlePasskey(webAuthn *webauthn.WebAuthn, w http.ResponseWriter, r *http.Request, session *Session) {
	if session == nil {
		http.Error(w, "missing session", http.StatusInternalServerError)
		return
	}

	switch r.URL.Path {
	case "/passkey/register/begin":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Username string `json:"username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		user, err := getOrCreatePasskeyUser(req.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		creation, webAuthnSession, err := webAuthn.BeginRegistration(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		session.WebAuthnSession = webAuthnSession
		session.PasskeyUsername = user.Username
		writeJSON(w, creation)

	case "/passkey/register/finish":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if session.WebAuthnSession == nil || session.PasskeyUsername == "" {
			http.Error(w, "registration not started", http.StatusBadRequest)
			return
		}
		user := getPasskeyUserByName(session.PasskeyUsername)
		if user == nil {
			http.Error(w, "user not found", http.StatusBadRequest)
			return
		}
		credential, err := webAuthn.FinishRegistration(user, *session.WebAuthnSession, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		addPasskeyCredential(user.Username, *credential)
		session.Username = user.Username
		session.WebAuthnSession = nil
		session.PasskeyUsername = ""
		writeJSON(w, map[string]string{"username": user.Username})

	case "/passkey/login/begin":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		assertion, webAuthnSession, err := webAuthn.BeginDiscoverableLogin()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		session.WebAuthnSession = webAuthnSession
		session.PasskeyUsername = ""
		writeJSON(w, assertion)

	case "/passkey/login/finish":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if session.WebAuthnSession == nil {
			http.Error(w, "login not started", http.StatusBadRequest)
			return
		}
		user, credential, err := webAuthn.FinishPasskeyLogin(findPasskeyUser, *session.WebAuthnSession, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		passkeyUser := user.(*passkeyUser)
		updatePasskeyCredential(passkeyUser.Username, *credential)
		session.Username = passkeyUser.Username
		session.WebAuthnSession = nil
		writeJSON(w, map[string]string{"username": passkeyUser.Username})

	case "/passkey/logout":
		session.Username = "anonymous"
		session.WebAuthnSession = nil
		session.PasskeyUsername = ""
		http.Redirect(w, r, "/", http.StatusSeeOther)

	default:
		http.NotFound(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

const passkeyPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>RESQ sign in</title>
<style>
body{font-family:system-ui,-apple-system,Segoe UI,sans-serif;margin:0;min-height:100vh;display:grid;place-items:center;background:#f6f7f9;color:#16181d}
main{width:min(420px,calc(100vw - 32px));background:#fff;border:1px solid #d9dde5;border-radius:8px;padding:24px;box-shadow:0 18px 50px #1b243014}
h1{font-size:22px;margin:0 0 18px}
label{display:block;font-size:14px;font-weight:600;margin-bottom:6px}
input{width:100%;box-sizing:border-box;border:1px solid #b8c0cc;border-radius:6px;padding:10px 12px;font:inherit;margin-bottom:12px}
button{width:100%;border:0;border-radius:6px;background:#145c9e;color:white;font:inherit;font-weight:700;padding:10px 12px;cursor:pointer;margin-top:8px}
button.secondary{background:#354052}
p{min-height:20px;margin:14px 0 0;color:#9b1c1c;font-size:14px}
</style>
</head>
<body>
<main>
<h1>Sign in to RESQ</h1>
<label for="username">Username</label>
<input id="username" autocomplete="username" placeholder="name@example.com">
<button id="login" class="secondary">Sign in with passkey</button>
<button id="register">Register this device</button>
<p id="message"></p>
</main>
<script>
const message = document.getElementById('message');
const enc = v => btoa(String.fromCharCode(...new Uint8Array(v))).replaceAll('+','-').replaceAll('/','_').replaceAll('=','');
const dec = v => Uint8Array.from(atob(v.replaceAll('-','+').replaceAll('_','/') + '==='.slice((v.length + 3) % 4)), c => c.charCodeAt(0));
const post = async (url, body) => {
  const res = await fetch(url, {method:'POST', headers:{'Content-Type':'application/json'}, body: body ? JSON.stringify(body) : '{}'});
  if (!res.ok) throw new Error(await res.text());
  return res.json();
};
const prepCreate = o => {
  o.publicKey.challenge = dec(o.publicKey.challenge);
  o.publicKey.user.id = dec(o.publicKey.user.id);
  (o.publicKey.excludeCredentials || []).forEach(c => c.id = dec(c.id));
  return o;
};
const prepGet = o => {
  o.publicKey.challenge = dec(o.publicKey.challenge);
  (o.publicKey.allowCredentials || []).forEach(c => c.id = dec(c.id));
  return o;
};
const pack = c => {
  const r = c.response;
  const response = {clientDataJSON: enc(r.clientDataJSON)};
  if (r.attestationObject) response.attestationObject = enc(r.attestationObject);
  if (r.authenticatorData) response.authenticatorData = enc(r.authenticatorData);
  if (r.signature) response.signature = enc(r.signature);
  if (r.userHandle) response.userHandle = enc(r.userHandle);
  if (r.getTransports) response.transports = r.getTransports();
  return JSON.stringify({
    id: c.id,
    rawId: enc(c.rawId),
    type: c.type,
    authenticatorAttachment: c.authenticatorAttachment,
    clientExtensionResults: c.getClientExtensionResults(),
    response
  });
};
document.getElementById('register').onclick = async () => {
  try {
    message.textContent = '';
    const username = document.getElementById('username').value.trim();
    const options = prepCreate(await post('/passkey/register/begin', {username}));
    const credential = await navigator.credentials.create(options);
    await fetch('/passkey/register/finish', {method:'POST', headers:{'Content-Type':'application/json'}, body: pack(credential)});
    location.reload();
  } catch (e) { message.textContent = e.message; }
};
document.getElementById('login').onclick = async () => {
  try {
    message.textContent = '';
    const options = prepGet(await post('/passkey/login/begin'));
    const credential = await navigator.credentials.get(options);
    await fetch('/passkey/login/finish', {method:'POST', headers:{'Content-Type':'application/json'}, body: pack(credential)});
    location.reload();
  } catch (e) { message.textContent = e.message; }
};
</script>
</body>
</html>`
