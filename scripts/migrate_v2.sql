-- EmlakPro V2 Migration
-- Çalıştır: psql -U emlakpro -d emlakpro -f migrate_v2.sql

-- 1. İlanlara is_listed (vitrin görünürlüğü) ve durum alanları
ALTER TABLE listings ADD COLUMN IF NOT EXISTS is_listed BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE listings ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'aktif';
ALTER TABLE listings ADD COLUMN IF NOT EXISTS closing_price BIGINT;

-- 2. İlan tarihçe tablosu
CREATE TABLE IF NOT EXISTS listing_history (
    id          BIGSERIAL PRIMARY KEY,
    listing_id  BIGINT NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    action      VARCHAR(50) NOT NULL, -- created, updated, activated, deactivated, listed, unlisted
    status      VARCHAR(20),
    closing_price BIGINT,
    note        TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_lhistory_listing  ON listing_history(listing_id);
CREATE INDEX IF NOT EXISTS idx_lhistory_created  ON listing_history(created_at);

-- 3. Müşteri (CRM) tablosu
CREATE TABLE IF NOT EXISTS customers (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id),
    name       VARCHAR(200) NOT NULL,
    phone      VARCHAR(50),
    email      VARCHAR(200),
    source     VARCHAR(100),
    notes      TEXT,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_customers_user ON customers(user_id);

-- 4. Müşteri-İlan ilişki tablosu
CREATE TABLE IF NOT EXISTS customer_listings (
    id          BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    listing_id  BIGINT NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    note        TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(customer_id, listing_id)
);
CREATE INDEX IF NOT EXISTS idx_custlist_customer ON customer_listings(customer_id);
CREATE INDEX IF NOT EXISTS idx_custlist_listing  ON customer_listings(listing_id);
