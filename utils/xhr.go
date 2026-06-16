package utils

import "net/http"

type XhrEvent string

func IsXhr(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func TriggerXhrEvent(w http.ResponseWriter, event XhrEvent) {
	w.Header().Set("HX-Trigger", string(event))
}
