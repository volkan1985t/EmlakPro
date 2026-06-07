package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/volkan1985t/EmlakPro/internal/model"
	"github.com/volkan1985t/EmlakPro/internal/repository"
	svc "github.com/volkan1985t/EmlakPro/internal/service"
)

// ─── Sabitler ────────────────────────────────────────────────

var ilgiTipLabel = map[string]string{
	"bilgi":         "Bilgi talebi",
	"teklif":        "Fiyat teklifi",
	"goruntuleme":   "Görüntüleme",
	"belge":         "Resim / belge",
	"geri_arama":    "Geri arama",
	"baska_portfoy": "Başka portföy",
}

var ilgiSonrakiOneri = map[string]string{
	"bilgi":         "Bilgi ver, 3 gün sonra durumunu sor",
	"teklif":        "Maliki ara ve teklifi sun",
	"goruntuleme":   "Randevu tarihi belirle",
	"belge":         "Belgeleri gönder",
	"geri_arama":    "Belirtilen saatte ara",
	"baska_portfoy": "Ne aradığını öğren, talebe dönüştür",
}

var ilgiDurumLabel = map[string]string{
	"yeni": "🆕 Yeni", "gorusuluyor": "💬 Görüşülüyor",
	"pazarlik": "💰 Pazarlık", "sonuc": "🏁 Sonuç",
}

// sonraki durum zinciri
var ilgiNextStatus = map[string]string{
	"yeni": "gorusuluyor", "gorusuluyor": "pazarlik", "pazarlik": "sonuc",
}

// ─── Menü ────────────────────────────────────────────────────

func (h *BotHandler) sendInterestMenu(chatID int64, user *model.User) {
	h.clearSession(chatID)
	counts, _ := h.interestRepo.CountByStatus(user.ID)
	aktif := counts["yeni"] + counts["gorusuluyor"] + counts["pazarlik"]
	kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
		{{Text: "➕ Yeni İlgi", CallbackData: "int_add"}},
		{{Text: "📋 İlgilerim", CallbackData: "int_list"}},
		{{Text: "📅 Bugün Aranacaklar", CallbackData: "int_today"}},
	}}
	h.tg.SendMessage(chatID,
		fmt.Sprintf("📞 <b>İlan İlgileri</b>\n%d aktif ilgi takipte.", aktif), kb)
}

// ─── Yeni İlgi wizard ────────────────────────────────────────

func (h *BotHandler) startInterestWizard(chatID int64, user *model.User) {
	h.setSession(chatID, user.ID, "interest_wizard", map[string]string{"_iwz": "listing"})
	h.sendInterestListingPicker(chatID, user)
}

// sendInterestListingPicker — wizard'da ilan seçim ekranı (kendi ilanları)
func (h *BotHandler) sendInterestListingPicker(chatID int64, user *model.User) {
	listings, _ := h.listingRepo.List(repository.ListFilter{UserID: user.ID, OnlyMine: true})
	rows := [][]svc.TGInlineButton{}
	limit := len(listings)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		l := listings[i]
		title := l.Fields["title"]
		if title == "" {
			title = fmt.Sprintf("İlan #%d", l.ListingNo)
		}
		if len(title) > 40 {
			title = title[:40]
		}
		rows = append(rows, []svc.TGInlineButton{
			{Text: "🏠 " + title, CallbackData: fmt.Sprintf("iwz_lst_%d", l.ID)},
		})
	}
	rows = append(rows, []svc.TGInlineButton{{Text: "— İlansız / Genel —", CallbackData: "iwz_lst_0"}})
	rows = append(rows, []svc.TGInlineButton{{Text: "❌ İptal", CallbackData: "menu_main"}})
	h.tg.SendMessage(chatID, "➕ <b>Yeni İlgi</b>\nHangi ilana geldi?",
		&svc.TGInlineKeyboard{InlineKeyboard: rows})
}

func (h *BotHandler) interestWizardListing(chatID int64, user *model.User, idStr string) {
	sess := h.getSession(chatID)
	if sess == nil {
		h.sendInterestMenu(chatID, user)
		return
	}
	sess.Data["listing_id"] = idStr
	sess.Data["_iwz"] = "tip"
	h.saveSession(sess)
	kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
		{{Text: "ℹ️ Bilgi talebi", CallbackData: "iwz_tip_bilgi"}},
		{{Text: "💰 Fiyat teklifi", CallbackData: "iwz_tip_teklif"}},
		{{Text: "👁 Görüntüleme", CallbackData: "iwz_tip_goruntuleme"}},
		{{Text: "📄 Resim / belge", CallbackData: "iwz_tip_belge"}},
		{{Text: "📞 Geri arama", CallbackData: "iwz_tip_geri_arama"}},
		{{Text: "🔄 Başka portföy", CallbackData: "iwz_tip_baska_portfoy"}},
	}}
	h.tg.SendMessage(chatID, "Ne için aradı?", kb)
}

func (h *BotHandler) interestWizardTip(chatID int64, user *model.User, tip string) {
	sess := h.getSession(chatID)
	if sess == nil {
		h.sendInterestMenu(chatID, user)
		return
	}
	sess.Data["type"] = tip
	// Alıcı bilgisi önceden doluysa (rehberden) alıcı adımlarını atla
	if sess.Data["buyer_name"] != "" {
		if tip == "teklif" {
			sess.Data["_iwz"] = "offer"
			h.saveSession(sess)
			h.tg.SendMessage(chatID, "💰 Teklif tutarını yazın (örn. 4.200.000) — yoksa - yazın:", nil)
			return
		}
		sess.Data["_iwz"] = "note"
		h.saveSession(sess)
		h.tg.SendMessage(chatID, "📝 Kısa not (yoksa - yazın):", nil)
		return
	}
	sess.Data["_iwz"] = "buyer_name"
	h.saveSession(sess)
	h.tg.SendMessage(chatID, "👤 Alıcının adını yazın:", nil)
}

func (h *BotHandler) interestWizardTextStep(msg *svc.TGMessage, user *model.User, session *BotSession) {
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)
	step := session.Data["_iwz"]

	switch step {
	case "buyer_name":
		if text == "" {
			h.tg.SendMessage(chatID, "👤 Alıcının adını yazın:", nil)
			return
		}
		session.Data["buyer_name"] = text
		session.Data["_iwz"] = "buyer_phone"
		h.saveSession(session)
		h.tg.SendMessage(chatID, "📞 Telefon numarasını yazın (yoksa - yazın):", nil)
		return

	case "buyer_phone":
		phone := text
		if phone == "-" {
			phone = ""
		}
		session.Data["buyer_phone"] = phone
		if session.Data["type"] == "teklif" {
			session.Data["_iwz"] = "offer"
			h.saveSession(session)
			h.tg.SendMessage(chatID, "💰 Teklif tutarını yazın (örn. 4.200.000) — yoksa - yazın:", nil)
			return
		}
		session.Data["_iwz"] = "note"
		h.saveSession(session)
		h.tg.SendMessage(chatID, "📝 Kısa not (yoksa - yazın):", nil)
		return

	case "offer":
		offer := text
		if offer == "-" {
			offer = ""
		}
		session.Data["offer_amount"] = offer
		session.Data["_iwz"] = "note"
		h.saveSession(session)
		h.tg.SendMessage(chatID, "📝 Kısa not (yoksa - yazın):", nil)
		return

	case "note":
		note := text
		if note == "-" {
			note = ""
		}
		h.saveInterestFromWizard(chatID, user, session, note)
		return
	}

	// bilinmeyen alt-adım
	h.clearSession(chatID)
	h.sendInterestMenu(chatID, user)
}

func (h *BotHandler) saveInterestFromWizard(chatID int64, user *model.User, session *BotSession, note string) {
	lid, _ := strconv.ParseInt(session.Data["listing_id"], 10, 64)
	tip := session.Data["type"]
	it := &model.Interest{
		UserID:      user.ID,
		ListingID:   lid,
		BuyerName:   session.Data["buyer_name"],
		BuyerPhone:  session.Data["buyer_phone"],
		Type:        tip,
		Status:      "yeni",
		OfferAmount: session.Data["offer_amount"],
		NextStep:    ilgiSonrakiOneri[tip],
		Notes:       note,
	}
	h.clearSession(chatID)
	if err := h.interestRepo.Create(it); err != nil {
		h.tg.SendMessage(chatID, "❌ İlgi kaydedilemedi: "+err.Error(), nil)
		return
	}
	summary := fmt.Sprintf("✅ <b>İlgi kaydedildi</b>\n\n👤 %s\n📌 %s\n📋 Sonraki: %s",
		it.BuyerName, ilgiTipLabel[tip], it.NextStep)
	if tip == "baska_portfoy" {
		summary += "\n\n🔄 <i>Bu kişi başka portföy arıyor — Taleplerim'den talep girebilirsiniz.</i>"
	}
	kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
		{{Text: "📋 İlgilerim", CallbackData: "int_list"}},
		{{Text: "🏠 Ana Menü", CallbackData: "menu_main"}},
	}}
	h.tg.SendMessage(chatID, summary, kb)
}

// ─── Listeleme ───────────────────────────────────────────────

func (h *BotHandler) sendInterestList(chatID int64, user *model.User, todayOnly bool) {
	var items []model.Interest
	if todayOnly {
		items, _ = h.interestRepo.List(repository.InterestFilter{UserID: user.ID, TodayOnly: true})
	} else {
		all, _ := h.interestRepo.List(repository.InterestFilter{UserID: user.ID})
		for _, it := range all {
			if it.Status != "sonuc" {
				items = append(items, it)
			}
		}
	}
	if len(items) == 0 {
		msg := "📭 Aktif ilgi yok."
		if todayOnly {
			msg = "✅ Bugün aranacak kimse yok."
		}
		h.tg.SendMessage(chatID, msg, nil)
		return
	}
	title := fmt.Sprintf("📋 <b>İlgilerim</b> (%d)", len(items))
	if todayOnly {
		title = fmt.Sprintf("📅 <b>Bugün Aranacaklar</b> (%d)", len(items))
	}
	h.tg.SendMessage(chatID, title, nil)

	limit := len(items)
	if limit > 15 {
		limit = 15
	}
	for i := 0; i < limit; i++ {
		it := items[i]
		h.tg.SendMessage(chatID, h.interestCardText(&it), h.interestCardKeyboard(&it))
	}
	if len(items) > 15 {
		h.tg.SendMessage(chatID, fmt.Sprintf("… ve %d ilgi daha.", len(items)-15), nil)
	}
}

func (h *BotHandler) interestCardText(it *model.Interest) string {
	s := fmt.Sprintf("%s · %s\n👤 <b>%s</b>", ilgiDurumLabel[it.Status], ilgiTipLabel[it.Type], it.BuyerName)
	if it.BuyerPhone != "" {
		s += "\n📞 " + it.BuyerPhone
	}
	if it.ListingTitle != "" {
		s += "\n🏠 " + it.ListingTitle
	}
	if it.OfferAmount != "" {
		s += "\n💰 " + it.OfferAmount
	}
	if it.NextStep != "" {
		s += "\n📋 " + it.NextStep
	}
	if it.NextDate != "" {
		s += "\n📅 " + it.NextDate
	}
	return s
}

func (h *BotHandler) interestCardKeyboard(it *model.Interest) *svc.TGInlineKeyboard {
	rows := [][]svc.TGInlineButton{}
	if it.Status != "sonuc" {
		rows = append(rows, []svc.TGInlineButton{
			{Text: "▶️ İlerlet", CallbackData: fmt.Sprintf("int_adv_%d", it.ID)},
			{Text: "✅ Kazanıldı", CallbackData: fmt.Sprintf("int_won_%d", it.ID)},
			{Text: "❌ Kaybedildi", CallbackData: fmt.Sprintf("int_lost_%d", it.ID)},
		})
	}
	rows = append(rows, []svc.TGInlineButton{
		{Text: "👤 Müşteri Yap", CallbackData: fmt.Sprintf("int_cust_%d", it.ID)},
	})
	return &svc.TGInlineKeyboard{InlineKeyboard: rows}
}

// ─── Durum işlemleri ─────────────────────────────────────────

func (h *BotHandler) interestAdvance(chatID int64, user *model.User, idStr string) {
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if !h.interestRepo.IsOwner(id, user.ID) {
		h.tg.SendMessage(chatID, "⛔ Bu kayıt size ait değil.", nil)
		return
	}
	it, err := h.interestRepo.GetByID(id)
	if err != nil {
		h.tg.SendMessage(chatID, "❌ Kayıt bulunamadı.", nil)
		return
	}
	next := ilgiNextStatus[it.Status]
	if next == "" {
		h.tg.SendMessage(chatID, "ℹ️ Bu ilgi zaten son aşamada.", nil)
		return
	}
	it.Status = next
	if err := h.interestRepo.Update(it); err != nil {
		h.tg.SendMessage(chatID, "❌ Güncellenemedi.", nil)
		return
	}
	h.tg.SendMessage(chatID, fmt.Sprintf("✅ <b>%s</b> → %s", it.BuyerName, ilgiDurumLabel[next]),
		&svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
			{{Text: "📋 İlgilerim", CallbackData: "int_list"}},
		}})
}

func (h *BotHandler) interestSetWon(chatID int64, user *model.User, idStr string) {
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if !h.interestRepo.IsOwner(id, user.ID) {
		h.tg.SendMessage(chatID, "⛔ Bu kayıt size ait değil.", nil)
		return
	}
	it, err := h.interestRepo.GetByID(id)
	if err != nil {
		h.tg.SendMessage(chatID, "❌ Kayıt bulunamadı.", nil)
		return
	}
	it.Status = "sonuc"
	it.Outcome = "kazanildi"
	h.interestRepo.Update(it)
	// Görev paketi öner
	kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
		{{Text: "🏷️ Satış Paketi", CallbackData: fmt.Sprintf("int_pkg_satis_%d", id)}},
		{{Text: "🔑 Kiralama Paketi", CallbackData: fmt.Sprintf("int_pkg_kiralama_%d", id)}},
		{{Text: "Vazgeç", CallbackData: "menu_main"}},
	}}
	h.tg.SendMessage(chatID,
		fmt.Sprintf("🎉 <b>%s — Kazanıldı!</b>\n\nGörev paketi açılsın mı?", it.BuyerName), kb)
}

func (h *BotHandler) interestSetOutcome(chatID int64, user *model.User, idStr, outcome string) {
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if !h.interestRepo.IsOwner(id, user.ID) {
		h.tg.SendMessage(chatID, "⛔ Bu kayıt size ait değil.", nil)
		return
	}
	it, err := h.interestRepo.GetByID(id)
	if err != nil {
		h.tg.SendMessage(chatID, "❌ Kayıt bulunamadı.", nil)
		return
	}
	it.Status = "sonuc"
	it.Outcome = outcome
	h.interestRepo.Update(it)
	h.tg.SendMessage(chatID, fmt.Sprintf("🏁 <b>%s</b> kapatıldı (kaybedildi).", it.BuyerName),
		&svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
			{{Text: "📋 İlgilerim", CallbackData: "int_list"}},
		}})
}

func (h *BotHandler) interestToCustomer(chatID int64, user *model.User, idStr string) {
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if !h.interestRepo.IsOwner(id, user.ID) {
		h.tg.SendMessage(chatID, "⛔ Bu kayıt size ait değil.", nil)
		return
	}
	it, err := h.interestRepo.GetByID(id)
	if err != nil {
		h.tg.SendMessage(chatID, "❌ Kayıt bulunamadı.", nil)
		return
	}
	name := it.BuyerName
	if name == "" {
		name = it.BuyerPhone
	}
	if dup, _ := h.customerRepo.FindDuplicate(user.ID, name, it.BuyerPhone); dup != nil {
		h.tg.SendMessage(chatID, fmt.Sprintf("ℹ️ <b>%s</b> zaten müşterilerinizde.", dup.Name), nil)
		return
	}
	c := &model.Customer{UserID: user.ID, Name: name, Phone: it.BuyerPhone, Source: "İlgi", IsActive: true}
	if err := h.customerRepo.Create(c); err != nil {
		h.tg.SendMessage(chatID, "❌ Müşteri eklenemedi.", nil)
		return
	}
	h.tg.SendMessage(chatID, fmt.Sprintf("✅ <b>%s</b> müşterilere eklendi.", name), nil)
}

// int_pkg_<tur>_<id>
func (h *BotHandler) interestTaskPackage(chatID int64, user *model.User, rest string) {
	parts := strings.SplitN(rest, "_", 2)
	if len(parts) != 2 {
		h.sendInterestMenu(chatID, user)
		return
	}
	tur := parts[0]
	id, _ := strconv.ParseInt(parts[1], 10, 64)
	key := "interest_tasks_satis"
	turLabel := "Satış"
	if tur == "kiralama" {
		key = "interest_tasks_kiralama"
		turLabel = "Kiralama"
	}
	list := h.cfg.CustomLists[key]
	if len(list) == 0 {
		h.tg.SendMessage(chatID, "⚠️ Görev şablonu tanımlı değil (config: "+key+").", nil)
		return
	}
	var who string
	if it, err := h.interestRepo.GetByID(id); err == nil {
		who = it.BuyerName
	}
	suffix := ""
	if who != "" {
		suffix = " — " + who
	}
	ok := 0
	for _, title := range list {
		req := &model.CreateTaskRequest{Title: title + suffix, Priority: "normal", Status: "bekliyor"}
		if _, err := h.taskRepo.Create(req, user.ID); err == nil {
			ok++
		}
	}
	h.tg.SendMessage(chatID,
		fmt.Sprintf("✅ <b>%d görev açıldı</b> (%s paketi).\nGörevler menüsünden takip edebilirsiniz.", ok, turLabel),
		&svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
			{{Text: "✅ Görevler", CallbackData: "menu_tasks"}},
			{{Text: "🏠 Ana Menü", CallbackData: "menu_main"}},
		}})
}
