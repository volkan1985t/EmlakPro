-- v4: İlan İlgileri (lead/ilgi takibi) — sadece kaydı giren danışman görür

CREATE TABLE IF NOT EXISTS interests (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id),                 -- sahibi (danışman) — gizlilik
    listing_id    BIGINT REFERENCES listings(id) ON DELETE SET NULL,    -- hangi ilan (NULL = ilansız)
    customer_id   BIGINT REFERENCES customers(id) ON DELETE SET NULL,   -- alıcı kaydı (opsiyonel)
    buyer_name    VARCHAR(255),                                         -- alıcı adı (serbest)
    buyer_phone   VARCHAR(50),                                          -- alıcı telefon
    type          VARCHAR(30) NOT NULL,                                 -- bilgi|teklif|goruntuleme|belge|geri_arama|baska_portfoy
    status        VARCHAR(30) NOT NULL DEFAULT 'yeni',                  -- yeni|gorusuluyor|pazarlik|sonuc
    outcome       VARCHAR(30),                                          -- kazanildi|kaybedildi|ilgilenmedi
    offer_amount  VARCHAR(50),                                          -- teklif tutarı (serbest metin)
    next_step     VARCHAR(500),                                         -- sonraki adım
    next_date     DATE,                                                 -- sonraki arama/aksiyon tarihi
    notes         TEXT,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_interests_user     ON interests(user_id);
CREATE INDEX IF NOT EXISTS idx_interests_listing  ON interests(listing_id);
CREATE INDEX IF NOT EXISTS idx_interests_status   ON interests(status);
CREATE INDEX IF NOT EXISTS idx_interests_nextdate ON interests(next_date);
