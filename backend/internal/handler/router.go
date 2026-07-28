package handler

import "net/http"

// NewMux registers HTTP routes for the API process.
func NewMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", Healthz)
	// Local-dev CORS so the Vite app can probe /healthz from another origin.
	// Tighten this before any real deployment.
	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
