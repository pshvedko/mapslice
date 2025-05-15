package w2c

import (
	"net/http"
	"sync"
)

type WaitGroupHandler struct {
	sync.WaitGroup
	http.Handler
}

func (h *WaitGroupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.WaitGroup.Add(1)
	h.Handler.ServeHTTP(w, r)
	h.WaitGroup.Done()
}

func NewHandler(handler http.Handler) *WaitGroupHandler {
	return &WaitGroupHandler{
		Handler: handler,
	}
}
