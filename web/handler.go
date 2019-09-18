package web

import (
	"net/http"
	"strings"
)

type handler struct {
	store Store
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metric := strings.TrimPrefix(r.URL.Path, "/")
	if metric == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h.store.AddMetric(metric)
	w.WriteHeader(http.StatusOK)
}
