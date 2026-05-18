package service

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/volkan1985t/EmlakPro/internal/config"
)

type ImageService struct {
	cfg *config.Config
}

func NewImageService(cfg *config.Config) *ImageService {
	return &ImageService{cfg: cfg}
}

type UploadResult struct {
	Path      string
	PublicURL string
	Width     int
	Height    int
	SizeBytes int64
}

// SaveCover — vitrin resmi: 800x600 max, en-boy oranı korunur
func (s *ImageService) SaveCover(r io.Reader, originalName, propType string, listingNo int64) (*UploadResult, error) {
	return s.saveImage(r, originalName, propType, listingNo, "cover", 800, 600, 85)
}

// SaveGallery — galeri resmi: 1920x1080 max, en-boy oranı korunur
func (s *ImageService) SaveGallery(r io.Reader, originalName, propType string, listingNo int64) (*UploadResult, error) {
	return s.saveImage(r, originalName, propType, listingNo, "gallery", 1920, 1080, 85)
}

func (s *ImageService) saveImage(r io.Reader, originalName, propType string, listingNo int64, imgType string, maxW, maxH, quality int) (*UploadResult, error) {
	// Klasör: uploads/Daire/1023/
	propDir := sanitizePropType(propType)
	var baseDir string
	if listingNo > 0 {
		baseDir = filepath.Join(s.cfg.App.UploadDir, propDir, fmt.Sprintf("%d", listingNo))
	} else {
		baseDir = filepath.Join(s.cfg.App.UploadDir, propDir, "tmp")
	}
	origDir := filepath.Join(baseDir, "original")

	if err := os.MkdirAll(origDir, 0755); err != nil {
		return nil, fmt.Errorf("dizin oluşturulamadı: %w", err)
	}

	// Geçici dosyaya oku
	tmp, err := os.CreateTemp("", "emlak-upload-*")
	if err != nil {
		return nil, fmt.Errorf("geçici dosya oluşturulamadı: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if _, err := io.Copy(tmp, r); err != nil {
		return nil, fmt.Errorf("dosya okunamadı: %w", err)
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	// Decode
	img, _, err := image.Decode(tmp)
	if err != nil {
		return nil, fmt.Errorf("geçersiz resim formatı: %w", err)
	}

	fileName := uuid.New().String() + ".jpg"

	// Orijinal kopyayı kaydet
	origPath := filepath.Join(origDir, fileName)
	if err := s.saveJPEG(img, origPath, 95); err != nil {
		return nil, fmt.Errorf("orijinal kaydedilemedi: %w", err)
	}

	// Boyutlandır
	bounds := img.Bounds()
	var processed *image.NRGBA
	if bounds.Dx() > maxW || bounds.Dy() > maxH {
		processed = imaging.Fit(img, maxW, maxH, imaging.Lanczos)
	} else {
		processed = imaging.Clone(img)
	}

	// İşlenmiş resmi kaydet
	destPath := filepath.Join(baseDir, fileName)
	if err := s.saveJPEG(processed, destPath, quality); err != nil {
		return nil, fmt.Errorf("resim kaydedilemedi: %w", err)
	}

	stat, _ := os.Stat(destPath)
	var sizeBytes int64
	if stat != nil {
		sizeBytes = stat.Size()
	}

	finalBounds := processed.Bounds()

	// Public URL
	relPath := strings.TrimPrefix(destPath, s.cfg.App.UploadDir+"/")
	relPath = strings.TrimPrefix(relPath, s.cfg.App.UploadDir)
	publicURL := fmt.Sprintf("%s/uploads/%s", s.cfg.App.BaseURL, filepath.ToSlash(relPath))

	return &UploadResult{
		Path:      destPath,
		PublicURL: publicURL,
		Width:     finalBounds.Dx(),
		Height:    finalBounds.Dy(),
		SizeBytes: sizeBytes,
	}, nil
}

func (s *ImageService) saveJPEG(img image.Image, path string, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
}


// MoveFromTmp — tmp klasöründeki resmi ilan no klasörüne taşır
func (s *ImageService) MoveFromTmp(filePath, propType string, listingNo int64) string {
	if filePath == "" || listingNo == 0 {
		return ""
	}
	propDir := sanitizePropType(propType)

	// tmp klasörü içinde mi?
	tmpDir := filepath.Join(s.cfg.App.UploadDir, propDir, "tmp")
	origTmpDir := filepath.Join(tmpDir, "original")

	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return ""
	}

	absTmp, _ := filepath.Abs(tmpDir)
	if !strings.HasPrefix(absFile, absTmp) {
		return "" // zaten tmp'de değil
	}

	fileName := filepath.Base(filePath)

	// Hedef klasörler
	destDir := filepath.Join(s.cfg.App.UploadDir, propDir, fmt.Sprintf("%d", listingNo))
	destOrigDir := filepath.Join(destDir, "original")
	os.MkdirAll(destDir, 0755)
	os.MkdirAll(destOrigDir, 0755)

	// İşlenmiş dosyayı taşı
	destPath := filepath.Join(destDir, fileName)
	os.Rename(filePath, destPath)

	// Orijinal dosyayı taşı
	origSrc := filepath.Join(origTmpDir, fileName)
	origDest := filepath.Join(destOrigDir, fileName)
	os.Rename(origSrc, origDest)

	return destPath
}

// DeleteListingFiles — işlenmiş dosyaları siler, original klasörünü korur
func (s *ImageService) DeleteListingFiles(propType string, listingNo int64) error {
	propDir := sanitizePropType(propType)
	baseDir := filepath.Join(s.cfg.App.UploadDir, propDir, fmt.Sprintf("%d", listingNo))

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.Name() == "original" {
			continue // orijinalleri koru
		}
		os.Remove(filepath.Join(baseDir, e.Name()))
	}
	return nil
}

// DeleteImage — tek dosya sil
func (s *ImageService) DeleteImage(filePath string) error {
	absUpload, err := filepath.Abs(s.cfg.App.UploadDir)
	if err != nil {
		return err
	}
	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absUpload, absFile)
	if err != nil || len(rel) > 0 && rel[0] == '.' {
		return fmt.Errorf("güvenlik hatası: izin verilmeyen dosya yolu")
	}
	if err := os.Remove(absFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("dosya silinemedi: %w", err)
	}
	return nil
}

func (s *ImageService) DeleteFile(path string) { s.DeleteImage(path) }

func (s *ImageService) PathToURL(filePath string) string {
	if filePath == "" {
		return ""
	}
	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		return filePath
	}
	clean := filepath.ToSlash(filePath)
	if idx := strings.Index(clean, "uploads/"); idx >= 0 {
		clean = clean[idx+len("uploads/"):]
	}
	return fmt.Sprintf("%s/uploads/%s", s.cfg.App.BaseURL, clean)
}

func sanitizePropType(propType string) string {
	if propType == "" {
		return "diger"
	}
	r := strings.NewReplacer(
		" ", "_", "/", "_", "\\", "_",
		"..", "", ".", "",
	)
	return r.Replace(propType)
}
