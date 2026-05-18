package service

import (
	"fmt"
	"strings"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/disintegration/imaging"
	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/google/uuid"
)

// ImageService — resim yükleme, yeniden boyutlandırma ve sıkıştırma
type ImageService struct {
	cfg *config.Config
}

func NewImageService(cfg *config.Config) *ImageService {
	return &ImageService{cfg: cfg}
}

type UploadResult struct {
	Path         string // Sunucu dosya yolu
	PublicURL    string // Nginx'ten erişilebilir URL
	Width        int
	Height       int
	SizeBytes    int64
}

// SaveCover — vitrin resmini 1920x1080 olarak kaydeder
func (s *ImageService) SaveCover(r io.Reader, originalName string) (*UploadResult, error) {
	return s.saveImage(r, originalName, "covers",
		s.cfg.App.MaxImageWidth,
		s.cfg.App.MaxImageHeight,
		s.cfg.App.ImageQuality,
	)
}

// SaveGallery — galeri resmini 1920x1080 olarak kaydeder
func (s *ImageService) SaveGallery(r io.Reader, originalName string) (*UploadResult, error) {
	return s.saveImage(r, originalName, "gallery",
		s.cfg.App.MaxImageWidth,
		s.cfg.App.MaxImageHeight,
		s.cfg.App.ImageQuality,
	)
}

// saveImage — ortak resim işleme fonksiyonu
// - Orijinal boyut MaxW x MaxH'den küçükse olduğu gibi kaydeder
// - Büyükse en-boy oranını koruyarak sığdırır (crop yapmaz)
// - JPEG olarak sıkıştırır
func (s *ImageService) saveImage(r io.Reader, originalName, subDir string, maxW, maxH, quality int) (*UploadResult, error) {
	// Yıl/ay klasörü
	now := time.Now()
	datePath := fmt.Sprintf("%d/%02d", now.Year(), now.Month())
	dir := filepath.Join(s.cfg.App.UploadDir, subDir, datePath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("upload dizini oluşturulamadı: %w", err)
	}

	// Dosyayı önce geçici alana oku
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

	// Resmi decode et (JPEG ve PNG desteklenir)
	img, _, err := image.Decode(tmp)
	if err != nil {
		return nil, fmt.Errorf("geçersiz resim formatı: %w", err)
	}

	// Boyutlandırma: en-boy oranını koruyarak MaxW x MaxH içine sığdır
	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	var processed *image.NRGBA
	if origW > maxW || origH > maxH {
		// Fit: kırpmadan, oranı koruyarak küçült
		processed = imaging.Fit(img, maxW, maxH, imaging.Lanczos)
	} else {
		// Zaten küçük — dönüştür ama boyutlandırma
		processed = imaging.Clone(img)
	}

	finalBounds := processed.Bounds()

	// Hedef dosya adı: UUID + .jpg
	fileName := uuid.New().String() + ".jpg"
	destPath := filepath.Join(dir, fileName)

	f, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("dosya oluşturulamadı: %w", err)
	}
	defer f.Close()

	// JPEG olarak yaz
	if err := jpeg.Encode(f, processed, &jpeg.Options{Quality: quality}); err != nil {
		os.Remove(destPath) // hata durumunda temizle
		return nil, fmt.Errorf("JPEG encode hatası: %w", err)
	}

	// Dosya boyutu
	stat, _ := f.Stat()
	var sizeBytes int64
	if stat != nil {
		sizeBytes = stat.Size()
	}

	// Public URL: /uploads/covers/2024/06/uuid.jpg
	relPath := fmt.Sprintf("%s/%s/%s", subDir, datePath, fileName)
	publicURL := fmt.Sprintf("%s/uploads/%s", s.cfg.App.BaseURL, relPath)

	return &UploadResult{
		Path:      destPath,
		PublicURL: publicURL,
		Width:     finalBounds.Dx(),
		Height:    finalBounds.Dy(),
		SizeBytes: sizeBytes,
	}, nil
}

// DeleteImage — dosyayı diskten siler
func (s *ImageService) DeleteImage(filePath string) error {
	// Güvenlik: sadece upload dizini içindeki dosyaları sil
	absUpload, err := filepath.Abs(s.cfg.App.UploadDir)
	if err != nil {
		return err
	}
	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	// Path traversal koruması
	rel, err := filepath.Rel(absUpload, absFile)
	if err != nil || len(rel) > 0 && rel[0] == '.' {
		return fmt.Errorf("güvenlik hatası: izin verilmeyen dosya yolu")
	}

	if err := os.Remove(absFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("dosya silinemedi: %w", err)
	}
	return nil
}

// DeleteFile — alias for DeleteImage used by task handler
func (s *ImageService) DeleteFile(path string) { s.DeleteImage(path) }

// PathToURL — dosya yolunu public URL'e çevirir
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
