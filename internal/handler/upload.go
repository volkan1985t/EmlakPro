package handler

import (
	"net/http"
	"strconv"

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

func getPropTypeAndNo(r *http.Request) (string, int64) {
	propType := r.FormValue("prop_type")
	listingNo, _ := strconv.ParseInt(r.FormValue("listing_no"), 10, 64)
	return propType, listingNo
}

// POST /api/upload/cover
func (h *UploadHandler) Cover(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
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

	propType, listingNo := getPropTypeAndNo(r)
	result, err := h.imageSvc.SaveCover(file, header.Filename, propType, listingNo)
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

// POST /api/upload/gallery
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

	propType, listingNo := getPropTypeAndNo(r)
	result, err := h.imageSvc.SaveGallery(file, header.Filename, propType, listingNo)
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
