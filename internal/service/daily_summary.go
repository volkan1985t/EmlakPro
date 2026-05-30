package service

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

type DailySummaryService struct {
	db  *sql.DB
	tg  *TelegramService
	cfg DailySummaryCfg
}

type DailySummaryCfg struct {
	Enabled    bool
	Hour       int
	SendSunday bool
}

func NewDailySummaryService(db *sql.DB, tg *TelegramService, cfg DailySummaryCfg) *DailySummaryService {
	return &DailySummaryService{db: db, tg: tg, cfg: cfg}
}

// Start — goroutine olarak çalıştır, her gün belirlenen saatte özet gönder
func (s *DailySummaryService) Start() {
	if !s.cfg.Enabled {
		log.Println("[daily-summary] devre dışı")
		return
	}
	log.Printf("[daily-summary] başlatıldı — her gün %02d:00'da", s.cfg.Hour)
	go func() {
		for {
			next := s.nextRunTime()
			log.Printf("[daily-summary] sonraki çalışma: %s", next.Format("02.01.2006 15:04"))
			time.Sleep(time.Until(next))
			s.run()
		}
	}()
}

func (s *DailySummaryService) nextRunTime() time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), s.cfg.Hour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	// Pazar günü atla
	if !s.cfg.SendSunday {
		for next.Weekday() == time.Sunday {
			next = next.Add(24 * time.Hour)
		}
	}
	return next
}

func (s *DailySummaryService) run() {
	log.Println("[daily-summary] özet gönderiliyor...")
	users, err := s.getUsersWithChatID()
	if err != nil {
		log.Printf("[daily-summary] kullanıcılar alınamadı: %v", err)
		return
	}
	for _, u := range users {
		msg, err := s.buildSummary(u)
		if err != nil {
			log.Printf("[daily-summary] özet oluşturulamadı user=%d: %v", u.id, err)
			continue
		}
		if err := s.tg.SendNotification(u.chatID, msg); err != nil {
			log.Printf("[daily-summary] gönderilemedi user=%d: %v", u.id, err)
		} else {
			log.Printf("[daily-summary] gönderildi user=%d", u.id)
		}
	}
}

type summaryUser struct {
	id       int64
	fullName string
	chatID   int64
}

func (s *DailySummaryService) getUsersWithChatID() ([]summaryUser, error) {
	rows, err := s.db.Query(`
		SELECT id, full_name, telegram_chat_id::bigint
		FROM users
		WHERE is_active = true
		  AND telegram_chat_id IS NOT NULL
		  AND notify_telegram = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []summaryUser
	for rows.Next() {
		var u summaryUser
		if err := rows.Scan(&u.id, &u.fullName, &u.chatID); err != nil {
			continue
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *DailySummaryService) buildSummary(u summaryUser) (string, error) {
	today := time.Now().Format("02.01.2006")
	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	firstName := strings.Split(u.fullName, " ")[0]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🌅 <b>Günaydın %s!</b> — %s\n\n", firstName, today))

	// ── Bugünkü görevler ────────────────────────────────────
	tasks, err := s.getTodayTasks(u.id)
	if err != nil {
		log.Printf("[daily-summary] görevler alınamadı: %v", err)
	}
	if len(tasks) > 0 {
		sb.WriteString(fmt.Sprintf("📋 <b>BUGÜNÜN GÖREVLERİ (%d)</b>\n", len(tasks)))
		for _, t := range tasks {
			emoji := "⏳"
			if t.priority == "acil" {
				emoji = "🔴"
			} else if t.priority == "yuksek" {
				emoji = "🟠"
			}
			sb.WriteString(fmt.Sprintf("%s %s\n", emoji, t.title))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("📋 Bugün görevin yok 🎉\n\n")
	}

	// ── Genel istatistik ─────────────────────────────────────
	stats, err := s.getStats(u.id)
	if err == nil {
		sb.WriteString("📊 <b>DURUMUN</b>\n")
		sb.WriteString(fmt.Sprintf("🏠 Aktif ilanların: %d\n", stats.listings))
		sb.WriteString(fmt.Sprintf("🎯 Açık taleplerin: %d\n", stats.requests))
		sb.WriteString(fmt.Sprintf("👥 Müşterilerin: %d\n\n", stats.customers))
	}

	// ── Son 1 haftadaki eşleşmeler ───────────────────────────
	matches, err := s.getRecentMatches(u.id, weekAgo)
	if err != nil {
		log.Printf("[daily-summary] eşleşmeler alınamadı: %v", err)
	}
	if len(matches) > 0 {
		sb.WriteString(fmt.Sprintf("🔔 <b>SON 1 HAFTA EŞLEŞMELERİ (%d)</b>\n", len(matches)))
		for i, m := range matches {
			if i >= 5 {
				sb.WriteString(fmt.Sprintf("...ve %d tane daha\n", len(matches)-5))
				break
			}
			sb.WriteString(fmt.Sprintf("• %s → #%d %s\n", m.clientName, m.listingNo, m.listingTitle))
		}
		sb.WriteString("\n")
	}

	// ── Eski talepler uyarısı ────────────────────────────────
	oldRequests, err := s.getOldRequests(u.id)
	if err == nil && oldRequests > 0 {
		sb.WriteString(fmt.Sprintf("⚠️ %d talebiniz 7 günden eski — takip etmeyi unutmayın!\n", oldRequests))
	}

	return sb.String(), nil
}

type taskSummary struct {
	title    string
	priority string
}

func (s *DailySummaryService) getTodayTasks(userID int64) ([]taskSummary, error) {
	today := time.Now().Format("2006-01-02")
	rows, err := s.db.Query(`
		SELECT DISTINCT t.title, t.priority
		FROM tasks t
		LEFT JOIN task_assignees ta ON ta.task_id = t.id
		WHERE (t.created_by = $1 OR ta.user_id = $1)
		  AND t.status NOT IN ('tamamlandi','iptal')
		  AND (t.due_date = $2 OR t.due_date IS NULL)
		ORDER BY t.priority DESC
		LIMIT 10`, userID, today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []taskSummary
	for rows.Next() {
		var t taskSummary
		rows.Scan(&t.title, &t.priority)
		tasks = append(tasks, t)
	}
	return tasks, nil
}

type userStats struct {
	listings  int
	requests  int
	customers int
}

func (s *DailySummaryService) getStats(userID int64) (userStats, error) {
	var st userStats
	s.db.QueryRow(`SELECT COUNT(*) FROM listings WHERE user_id=$1 AND is_active=true`, userID).Scan(&st.listings)
	s.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE user_id=$1 AND is_active=true`, userID).Scan(&st.requests)
	s.db.QueryRow(`SELECT COUNT(*) FROM customers WHERE user_id=$1 AND is_active=true`, userID).Scan(&st.customers)
	return st, nil
}

type matchSummary struct {
	clientName   string
	listingNo    int64
	listingTitle string
}

func (s *DailySummaryService) getRecentMatches(userID int64, since string) ([]matchSummary, error) {
	// Son 1 haftada eklenen ve kullanıcının taleplerine eşleşen ilanlar
	rows, err := s.db.Query(`
		SELECT
			r.fields->>'client_name' as client_name,
			l.listing_no,
			COALESCE(l.fields->>'title','') as title
		FROM requests r
		CROSS JOIN listings l
		WHERE r.user_id = $1
		  AND r.is_active = true
		  AND l.is_active = true
		  AND l.created_at >= $2
		  AND r.fields->>'property_type' = l.fields->>'property_type'
		  AND (r.fields->>'district' = '' OR r.fields->>'district' = l.fields->>'district')
		ORDER BY l.created_at DESC
		LIMIT 10`, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var matches []matchSummary
	for rows.Next() {
		var m matchSummary
		rows.Scan(&m.clientName, &m.listingNo, &m.listingTitle)
		matches = append(matches, m)
	}
	return matches, nil
}

func (s *DailySummaryService) getOldRequests(userID int64) (int, error) {
	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM requests
		WHERE user_id=$1 AND is_active=true AND created_at < $2`,
		userID, weekAgo).Scan(&count)
	return count, err
}
