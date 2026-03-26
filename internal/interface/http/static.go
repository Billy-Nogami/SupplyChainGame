package http

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFiles embed.FS

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/app", http.StatusFound)
}

func (s *Server) handleApp(w http.ResponseWriter, r *http.Request) {
	index, err := fs.ReadFile(staticFiles, "static/index.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load app")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(index)
}

func staticFileHandler() http.Handler {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return http.NotFoundHandler()
	}

	return http.FileServer(http.FS(sub))
}
