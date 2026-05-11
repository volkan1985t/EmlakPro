# EmlakPro

Çok kullanıcılı emlak yönetim sistemi. Go + PostgreSQL + Nginx.

## Mimari

```
İnternet
    │
    ▼
┌─────────────────────┐
│   Nginx (80/443)    │  ← SSL, static dosyalar, rate limit
└────────┬────────────┘
         │ proxy_pass :8080
┌────────▼────────────┐
│   Go App :8080      │  ← JWT auth, API, resim işleme
└────────┬────────────┘
         │ TCP 5432
┌────────▼────────────┐
│  PostgreSQL         │  ← 192.168.55.46
│  emlakpro DB        │
└─────────────────────┘

Upload Dizini: /var/www/emlakpro/uploads/
  ├── covers/YYYY/MM/   ← Vitrin resimleri (1920x1080 JPEG)
  └── gallery/YYYY/MM/  ← Galeri resimleri (1920x1080 JPEG)
```

---

## Kurulum — Adım Adım

### Ön Gereksinimler

| Sunucu | IP | İşletim Sistemi |
|--------|-----|-----------------|
| DB Sunucusu | 192.168.55.46 | Turnkey Linux PostgreSQL |
| App Sunucusu | 192.168.55.45 | Ubuntu 22.04+ |

---

### ADIM 1 — DB Sunucusu (192.168.55.46)

```bash
# Scripti düzenle
nano scripts/01_db_server_setup.sh

# Şu satırları değiştir:
DB_PASS="güçlü_bir_şifre_girin"
APP_SERVER_IP="192.168.55.45"   # Go app sunucusunun IP'si

# Çalıştır
chmod +x scripts/01_db_server_setup.sh
sudo bash scripts/01_db_server_setup.sh
```

**Script şunları yapar:**
- PostgreSQL 16 kurar (kurulu değilse)
- `emlakpro` veritabanı ve `emlakuser` oluşturur
- Tüm tabloları migrate eder
- Sadece app sunucusu IP'sine port 5432 izni verir
- Her gece 02:00'de otomatik yedek kurar

---

### ADIM 2 — Bağlantı Testi (App sunucusundan)

```bash
# DB_PASS'ı aynı değere ayarla
nano scripts/03_db_connection_test.sh

chmod +x scripts/03_db_connection_test.sh
bash scripts/03_db_connection_test.sh
```

Beklenen çıktı:
```
[✓] Port 5432 erişilebilir.
[✓] Veritabanı bağlantısı başarılı!
[✓] Tablo mevcut: listing_images
[✓] Tablo mevcut: listings
[✓] Tablo mevcut: refresh_tokens
[✓] Tablo mevcut: requests
[✓] Tablo mevcut: users
```

---

### ADIM 3 — App Sunucusu

```bash
# Scripti düzenle — şu değişkenleri güncelle:
nano scripts/02_app_server_setup.sh

DB_PASS="adım1deki_şifre"
JWT_SECRET="en_az_32_karakter_rastgele_string"
ADMIN_PASS="admin_şifresi"
APP_SERVER_IP="bu_sunucunun_ip_adresi"
DOMAIN=""   # Alan adı varsa: "emlakpro.com"

# Çalıştır
chmod +x scripts/02_app_server_setup.sh
sudo bash scripts/02_app_server_setup.sh
```

---

### ADIM 4 — Kaynak Kodu Deploy

```bash
# Yerel makineden sunucuya kopyala
scp -r ./emlakpro user@APP_SUNUCU_IP:/opt/emlakpro/src

# Sunucuda derle ve başlat
ssh user@APP_SUNUCU_IP "sudo bash /opt/emlakpro/deploy.sh"
```

---

### ADIM 5 — Durum Kontrolü

```bash
# Servis durumu
systemctl status emlakpro

# Canlı log
journalctl -u emlakpro -f

# Nginx durumu
nginx -t && systemctl status nginx

# Sağlık kontrolü
curl http://localhost:8080/api/health
```

---

## Güncelleme (Deploy)

```bash
ssh user@SUNUCU_IP "sudo bash /opt/emlakpro/deploy.sh"
```

---

## config.json Referansı

| Alan | Açıklama |
|------|----------|
| `app.max_image_width` | Maksimum resim genişliği (px). Varsayılan: 1920 |
| `app.max_image_height` | Maksimum resim yüksekliği (px). Varsayılan: 1080 |
| `app.image_quality` | JPEG kalite (1-100). Varsayılan: 85 |
| `app.max_gallery_images` | Bir ilana maksimum galeri resmi. Varsayılan: 12 |
| `listing_fields.card_fields` | Mülk tipine göre kart görünüm alanları |
| `listing_fields.all_fields` | Tüm ilan alanları tanımı |

### Yeni Alan Ekleme

`config.json` dosyasındaki `listing_fields.all_fields` dizisine ekleyin:

```json
{
  "key": "parking",
  "label": "Otopark",
  "type": "select",
  "required": false,
  "source": "parking_options",
  "searchable": false
}
```

Kaynak listeyi de ekleyin:
```json
"parking_options": ["Var", "Yok", "Kapalı", "Açık"]
```

Veritabanı değişikliği **gerekmez** — tüm alanlar JSONB olarak saklanır.

---

## Dizin Yapısı

```
emlakpro/
├── cmd/server/main.go          # Giriş noktası
├── internal/
│   ├── auth/jwt.go             # JWT token
│   ├── config/config.go        # Config yükleme
│   ├── handler/                # HTTP handler'lar
│   │   ├── auth.go
│   │   ├── listing.go
│   │   ├── request.go
│   │   ├── upload.go
│   │   └── admin.go
│   ├── middleware/auth.go       # JWT middleware
│   ├── model/models.go         # Veri modelleri
│   ├── repository/             # DB sorguları
│   │   ├── db.go
│   │   ├── user.go
│   │   ├── listing.go
│   │   └── request.go
│   └── service/
│       └── image.go            # Resim sıkıştırma (1920x1080)
├── frontend/
│   ├── static/                 # CSS, JS, resimler
│   └── templates/              # HTML şablonlar
├── scripts/
│   ├── 01_db_server_setup.sh   # DB sunucu kurulumu
│   ├── 02_app_server_setup.sh  # App sunucu kurulumu
│   └── 03_db_connection_test.sh
├── config.json                 # Ana konfigürasyon
├── go.mod
└── README.md
```

---

## Yedekleme

Otomatik yedekler: `/var/backups/emlakpro/emlakpro_YYYYMMDD_HHMMSS.sql.gz`

Manuel yedek:
```bash
sudo -u postgres bash /usr/local/bin/emlakpro-backup.sh
```

Geri yükleme:
```bash
gunzip -c /var/backups/emlakpro/emlakpro_20240601_020000.sql.gz \
  | psql -h 192.168.55.46 -U emlakuser emlakpro
```
