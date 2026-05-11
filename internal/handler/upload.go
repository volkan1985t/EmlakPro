package handler

import (
	"net/http"

	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/volkan1985t/EmlakPro/internal/service"
)

type UploadHandler struct {
	cfg      *config.Config
	imageSvc *service.ImageService
}

func NewUploadHandler(cfg *config.Config, imageSvc *service.ImageService) *UploadHandler {
	return &UploadHandler{cfg: cfg, imageSvc: imageSvc}
}

// POST /api/upload/cover — vitrin resmi (1920x1080, tek dosya)
func (h *UploadHandler) Cover(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20) // 20 MB limit
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		jsonErr(w, "Dosya çok büyük (maks 20MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("cover")
	if err != nil {
		jsonErr(w, "Dosya alınamadı", http.StatusBadRequest)
		return
	}
	defer file.Close()

	result, err := h.imageSvc.SaveCover(file, header.Filename)
	if err != nil {
		jsonErr(w, "Resim kaydedilemedi: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]interface{}{
		"path":       result.Path,
		"url":        result.PublicURL,
		"width":      result.Width,
		"height":     result.Height,
		"size_bytes": result.SizeBytes,
	})
}

// POST /api/upload/gallery — galeri resmi (1920x1080, tek dosya, çok kez çağrılır)
func (h *UploadHandler) Gallery(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		jsonErr(w, "Dosya çok büyük (maks 20MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		jsonErr(w, "Dosya alınamadı", http.StatusBadRequest)
		return
	}
	defer file.Close()

	result, err := h.imageSvc.SaveGallery(file, header.Filename)
	if err != nil {
		jsonErr(w, "Resim kaydedilemedi: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]interface{}{
		"path":       result.Path,
		"url":        result.PublicURL,
		"width":      result.Width,
		"height":     result.Height,
		"size_bytes": result.SizeBytes,
	})
}
