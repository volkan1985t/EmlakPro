package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/volkan1985t/EmlakPro/internal/middleware"
	"github.com/volkan1985t/EmlakPro/internal/model"
)

type DashboardHandler struct {
	db *sql.DB
}

func NewDashboardHandler(db *sql.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// GET /api/dashboard
func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())

	stats, err := h.getStats(userID, isAdmin)
	if err != nil {
		jsonErr(w, "İstatistikler yüklenemedi", http.StatusInternalServerError)
		return
	}
	jsonOK(w, stats)
}

func (h *DashboardHandler) getStats(userID int64, isAdmin bool) (*model.DashboardStats, error) {
	userFilter := ""
	args := []interface{}{}
	if !isAdmin {
		userFilter = "WHERE user_id = $1"
		args = append(args, userID)
	}

	stats := &model.DashboardStats{
		ByStatus:      map[string]int{},
		ByType:        map[string]int{},
		ByDistrict:    []model.DistrictCount{},
		MonthlyAdded:  []model.MonthlyCount{},
		MonthlyClosed: []model.MonthlyCount{},
		TopAgents:     []model.AgentCount{},
	}

	// Özet sayılar
	err := h.db.QueryRow(fmt.Sprintf(`
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE is_active),
			COUNT(*) FILTER (WHERE NOT is_active),
			COUNT(*) FILTER (WHERE is_active AND is_listed),
			COUNT(*) FILTER (WHERE is_active AND NOT is_listed)
		FROM listings %s`, userFilter), args...,
	).Scan(&stats.TotalListings, &stats.ActiveListings, &stats.PassiveListings,
		&stats.ListedListings, &stats.UnlistedListings)
	if err != nil { return nil, err }

	// Duruma göre dağılım
	rows, err := h.db.Query(fmt.Sprintf(`
		SELECT COALESCE(status,'aktif'), COUNT(*) FROM listings %s GROUP BY status`,
		userFilter), args...)
	if err != nil { return nil, err }
	for rows.Next() {
		var s string; var c int
		rows.Scan(&s, &c)
		stats.ByStatus[s] = c
	}
	rows.Close()

	// İlan tipine göre (satılık/kiralık)
	rows, err = h.db.Query(fmt.Sprintf(`
		SELECT COALESCE(fields->>'listing_type','—'), COUNT(*)
		FROM listings %s
		GROUP BY fields->>'listing_type'`, userFilter), args...)
	if err != nil { return nil, err }
	for rows.Next() {
		var s string; var c int
		rows.Scan(&s, &c)
		stats.ByType[s] = c
	}
	rows.Close()

	// İlçe dağılımı (top 10)
	rows, err = h.db.Query(fmt.Sprintf(`
		SELECT COALESCE(fields->>'district','—'), COUNT(*)
		FROM listings %s
		GROUP BY fields->>'district'
		ORDER BY count DESC LIMIT 10`, userFilter), args...)
	if err != nil { return nil, err }
	for rows.Next() {
		var d model.DistrictCount
		rows.Scan(&d.District, &d.Count)
		stats.ByDistrict = append(stats.ByDistrict, d)
	}
	rows.Close()

	// Son 12 ay eklenen ilanlar
	rows, err = h.db.Query(fmt.Sprintf(`
		SELECT TO_CHAR(DATE_TRUNC('month', created_at),'YYYY-MM'), COUNT(*)
		FROM listings
		WHERE created_at >= NOW() - INTERVAL '12 months'
		%s
		GROUP BY 1 ORDER BY 1`,
		strings.Replace(userFilter, "WHERE", "AND", 1)), args...)
	if err != nil { return nil, err }
	for rows.Next() {
		var m model.MonthlyCount
		rows.Scan(&m.Month, &m.Count)
		stats.MonthlyAdded = append(stats.MonthlyAdded, m)
	}
	rows.Close()

	// Son 12 ay kapanan ilanlar (satıldı/kiralandı)
	rows, err = h.db.Query(fmt.Sprintf(`
		SELECT TO_CHAR(DATE_TRUNC('month', updated_at),'YYYY-MM'), COUNT(*)
		FROM listings
		WHERE status IN ('satildi','kiralandi')
		  AND updated_at >= NOW() - INTERVAL '12 months'
		%s
		GROUP BY 1 ORDER BY 1`,
		strings.Replace(userFilter, "WHERE", "AND", 1)), args...)
	if err != nil { return nil, err }
	for rows.Next() {
		var m model.MonthlyCount
		rows.Scan(&m.Month, &m.Count)
		stats.MonthlyClosed = append(stats.MonthlyClosed, m)
	}
	rows.Close()

	// Top danışmanlar (sadece admin görebilir)
	if isAdmin {
		rows, err = h.db.Query(`
			SELECT u.full_name, COUNT(l.id)
			FROM listings l JOIN users u ON u.id = l.user_id
			GROUP BY u.full_name ORDER BY count DESC LIMIT 10`)
		if err != nil { return nil, err }
		for rows.Next() {
			var a model.AgentCount
			rows.Scan(&a.Name, &a.Count)
			stats.TopAgents = append(stats.TopAgents, a)
		}
		rows.Close()
	}

	return stats, nil
}
