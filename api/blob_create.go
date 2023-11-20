package api

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

func (h *BlobHandler) CreateBlob(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ctx := r.Context()
	logger := log.Ctx(ctx)
	logger.Trace().Msg("create prompt request received")

	file, header, err := r.FormFile("upload")
	defer file.Close()

	name := r.FormValue("name")
	if name == "" {
		name = header.Filename
	}

	h.ctrl.BlobCreate(ctx, file, name)
	url, err := h.ctrl.Blob(ctx, name)
	if err != nil {
		log.Err(err).Msg("unable to create blob")
	}
	writeResponse(w, r, http.StatusCreated, url)
}
