package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/volkan1985t/EmlakPro/internal/model"
	"github.com/volkan1985t/EmlakPro/internal/repository"
	svc "github.com/volkan1985t/EmlakPro/internal/service"
)

// ─── BotHandler ──────────────────────────────────────────────

type BotHandler struct {
	cfg         *config.Config
	tg          *svc.TelegramService
	userRepo    *repository.UserRepository
	listingRepo *repository.ListingRepository
	requestRepo *repository.RequestRepository
	db          *sql.DB
}

func NewBotHandler(
	cfg *config.Config,
	tg *svc.TelegramService,
	db *sql.DB,
	userRepo *repository.UserRepository,
	listingRepo *repository.ListingRepository,
	requestRepo *repository.RequestRepository,
) *BotHandler {
	return &BotHandler{cfg: cfg, tg: tg, db: db,
		userRepo: userRepo, listingRepo: listingRepo, requestRepo: requestRepo}
}

// Handle — gelen her update'i işler
func (h *BotHandler) Handle(u svc.TGUpdate) {
	if u.CallbackQuery != nil {
		h.handleCallback(u.CallbackQuery)
		return
	}
	if u.Message != nil {
		h.handleMessage(u.Message)
	}
}

// ─── Mesaj handler ───────────────────────────────────────────

func (h *BotHandler) handleMessage(msg *svc.TGMessage) {
	chatID := msg.Chat.ID
	text   := strings.TrimSpace(msg.Text)

	// Kullanıcıyı bul
	user := h.getUserByChatID(chatID)

	// /iptal komutu
	if text == "/iptal" || strings.ToLower(text) == "iptal" {
		session := h.getSession(chatID)
		if session != nil && session.Step != "idle" {
			h.clearSession(chatID)
			h.tg.SendMessage(chatID, "❌ İşlem iptal edildi.", svc.MainMenuKeyboard())
		} else {
			h.sendMainMenu(chatID, "Ana Menü:")
		}
		return
	}

	// /start komutu
	if text == "/start" || strings.ToLower(text) == "emlakpro" {
		if user != nil {
			h.sendMainMenu(chatID, fmt.Sprintf("Hoş geldin, <b>%s</b>! 👋", user.FullName))
		} else {
			h.tg.SendMessage(chatID,
				"👋 <b>EmlakPro Bot'a Hoş Geldiniz!</b>\n\n"+
				"Bu bot sadece kayıtlı kullanıcılara hizmet verir.\n"+
				"Erişim için yöneticinizle iletişime geçin.\n\n"+
				"📞 Yönetici size Telegram ID'nizi sisteme ekleyecektir.",
				nil)
		}
		return
	}

	// Kayıtsız kullanıcı
	if user == nil {
		h.tg.SendMessage(chatID,
			"⛔ Sisteme kayıtlı bir kullanıcı değilsiniz.\n"+
			"Yöneticinize başvurun.", nil)
		return
	}

	// Aktif session var mı?
	session := h.getSession(chatID)
	if session != nil && session.Step != "idle" {
		h.handleSessionStep(msg, user, session, session.Data)
		return
	}

	// Komutlar
	switch strings.ToLower(text) {
	case "/menu", "menü", "menu":
		h.sendMainMenu(chatID, "Ana Menü:")
	default:
		h.sendMainMenu(chatID, "Bir seçenek seçin:")
	}
}

// ─── Callback handler ────────────────────────────────────────

func (h *BotHandler) handleCallback(cb *svc.TGCallback) {
	chatID := cb.Message.Chat.ID
	data   := cb.Data
	user   := h.getUserByChatID(chatID)

	if user == nil {
		h.tg.SendMessage(chatID, "⛔ Yetkisiz erişim.", nil)
		return
	}

	// Ana menüye dön
	if data == "menu_main" {
		h.clearSession(chatID)
		h.sendMainMenu(chatID, "Ana Menü:")
		return
	}

	switch {
	// ── İlanları Listele ──────────────────────────────────────
	case data == "menu_list":
		h.tg.SendMessage(chatID, "📋 <b>İlanları Listele</b>\nMülk tipi seçin:",
			svc.PropertyTypeKeyboard("list"))

	case strings.HasPrefix(data, "list_"):
		propType := strings.TrimPrefix(data, "list_")
		h.setSession(chatID, user.ID, "list_district", map[string]string{"property_type": propType})
		h.tg.SendMessage(chatID,
			fmt.Sprintf("📍 <b>%s İlanları</b>\nİlçe seçin (veya tüm ilçeler için 'Tümü'):", propType),
			h.districtKeyboardWithAll("list2_"+propType))

	case strings.HasPrefix(data, "list2_"):
		parts := strings.SplitN(strings.TrimPrefix(data, "list2_"), "_", 2)
		if len(parts) == 2 {
			propType := parts[0]
			district := parts[1]
			h.sendListings(chatID, user, propType, district, false)
		}

	// ── Benim İlanlarım ───────────────────────────────────────
	case data == "menu_mine":
		h.tg.SendMessage(chatID, "🏠 <b>Benim İlanlarım</b>\nMülk tipi seçin:",
			svc.PropertyTypeKeyboard("mine"))

	case strings.HasPrefix(data, "mine_"):
		propType := strings.TrimPrefix(data, "mine_")
		h.sendListings(chatID, user, propType, "", true)

	// ── İlan Gir ──────────────────────────────────────────────
	case data == "menu_add_listing":
		h.tg.SendMessage(chatID, "➕ <b>İlan Gir</b>\nMülk tipi seçin:",
			svc.PropertyTypeKeyboard("add_listing"))

	case strings.HasPrefix(data, "add_listing_"):
		propType := strings.TrimPrefix(data, "add_listing_")
		h.startListingWizard(chatID, user, propType)

	// ── Talep Gir ─────────────────────────────────────────────
	case data == "menu_add_request":
		h.startRequestWizard(chatID, user)

	// ── Görev (placeholder) ───────────────────────────────────
	case data == "menu_my_requests":
		h.sendMyRequests(chatID, user)

	case data == "wizard_cancel":
		h.clearSession(chatID)
		h.tg.SendMessage(chatID, "❌ İşlem iptal edildi.", svc.MainMenuKeyboard())

	case data == "menu_tasks":
		h.tg.SendMessage(chatID,
			"✅ <b>Görev Ekle</b>\n\n🚧 Bu özellik yakında aktif olacak!", nil)

	// ── Bildirimler ───────────────────────────────────────────
	case data == "menu_notify":
		notifyOn := user.NotifyTelegram
		status := "🔔 Aktif"
		if !notifyOn { status = "🔕 Kapalı" }
		h.tg.SendMessage(chatID,
			fmt.Sprintf("🔔 <b>Bildirim Ayarları</b>\n\nDurum: %s", status),
			svc.YesNoKeyboard("notify_on", "notify_off"))

	case data == "notify_on":
		h.setNotify(chatID, user.ID, true)
		h.tg.SendMessage(chatID, "🔔 Bildirimler <b>açıldı</b>.", nil)

	case data == "notify_off":
		h.setNotify(chatID, user.ID, false)
		h.tg.SendMessage(chatID, "🔕 Bildirimler <b>kapatıldı</b>.", nil)

	// ── Wizard adımları (callback'le gelen seçimler) ──────────
	default:
		session := h.getSession(chatID)
		if session != nil {
			h.handleWizardCallback(cb, user, session, data)
		}
	}
}

// ─── İlan Listele ─────────────────────────────────────────────

func (h *BotHandler) sendListings(chatID int64, user *model.User, propType, district string, onlyMine bool) {
	f := repository.ListFilter{
		PropertyType: propType,
		District:     district,
		IsAdmin:      user.Role == model.RoleAdmin,
	}
	if onlyMine { f.UserID = user.ID; f.OnlyMine = true } else { f.UserID = user.ID }

	listings, err := h.listingRepo.List(f)
	if err != nil || len(listings) == 0 {
		h.tg.SendMessage(chatID, "📭 Bu kriterlere uygun ilan bulunamadı.", nil)
		return
	}

	title := fmt.Sprintf("📋 <b>%s İlanları", propType)
	if district != "" && district != "Tümü" { title += " — " + district }
	title += fmt.Sprintf("</b>\n%d ilan bulundu:\n\n", len(listings))

	h.tg.SendMessage(chatID, title, nil)
	for i, il := range listings {
		if i >= 10 {
			h.tg.SendMessage(chatID,
				fmt.Sprintf("📌 ... ve <b>%d ilan daha</b> var.", len(listings)-10), nil)
			break
		}
		price := il.Fields["price_max"]
		if price == "" { price = il.Fields["price"] }

		details := ""
		fieldMap := []struct{ emoji, key string }{
			{"🏘️", "property_type"}, {"🏷️", "listing_type"},
			{"📍", "district"},      {"🏠", "neighborhood"},
			{"📐", "area_m2"},       {"🛏️", "rooms"},
			{"🏢", "floor"},         {"🔥", "heating"},
			{"📋", "zoning"},
		}
		for _, f := range fieldMap {
			if v := il.Fields[f.key]; v != "" {
				details += fmt.Sprintf("%s %s\n", f.emoji, v)
			}
		}

		ilanText := fmt.Sprintf(
			"<b>%s</b>  <code>#%d</code>\n"+
			"━━━━━━━━━━━━━━━\n"+
			"%s"+
			"💰 <b>%s ₺</b>\n"+
			"👤 %s",
			il.Fields["title"], il.ListingNo,
			details,
			formatTGPrice(price),
			il.OwnerName,
		)
		h.tg.SendMessage(chatID, ilanText, nil)
	}
}

func formatTGPrice(s string) string {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil { return s }
	return svc.FormatPrice(n)
}

// ─── İlan Ekleme Sihirbazı ────────────────────────────────────

type wizardStep struct {
	Key      string
	Prompt   string
	Keyboard func() interface{}
}

func (h *BotHandler) listingSteps(propType string) []wizardStep {
	cfg := h.cfg
	steps := []wizardStep{
		{Key: "listing_type", Prompt: "İlan tipi seçin:", Keyboard: func() interface{} {
			return svc.ListingTypeKeyboard("wiz_lt")
		}},
		{Key: "title", Prompt: "📝 İlan başlığını yazın:", Keyboard: nil},
		{Key: "contact", Prompt: "📞 İletişim bilgisini yazın (isim-telefon):", Keyboard: nil},
		{Key: "district", Prompt: "📍 İlçe seçin:", Keyboard: func() interface{} {
			return svc.DistrictKeyboard(cfg.Districts, "wiz_dist")
		}},
		{Key: "neighborhood", Prompt: "🏘️ Mahalle seçin:", Keyboard: func() interface{} {
			return svc.DistrictKeyboard(cfg.Neighborhoods, "wiz_hood")
		}},
	}

	switch propType {
	case "Daire":
		steps = append(steps,
			wizardStep{Key: "rooms", Prompt: "🛏️ Oda sayısı seçin:", Keyboard: func() interface{} {
				return svc.OptionsKeyboard(cfg.RoomOptions, "wiz_rooms")
			}},
		)
	case "Arsa":
		steps = append(steps,
			wizardStep{Key: "area_m2", Prompt: "📐 Alan (m²) yazın:", Keyboard: nil},
			wizardStep{Key: "zoning", Prompt: "📋 İmar durumu seçin:", Keyboard: func() interface{} {
				return svc.OptionsKeyboard(cfg.ZoningOptions, "wiz_zoning")
			}},
		)
	case "Villa", "Ticari":
		steps = append(steps,
			wizardStep{Key: "area_m2", Prompt: "📐 Alan (m²) yazın:", Keyboard: nil},
		)
	}

	steps = append(steps,
		wizardStep{Key: "price", Prompt: "💰 Fiyat yazın (₺):", Keyboard: nil},
		wizardStep{Key: "description", Prompt: "📄 Açıklama yazın (geçmek için - yazın):", Keyboard: nil},
	)
	return steps
}

func (h *BotHandler) startListingWizard(chatID int64, user *model.User, propType string) {
	steps := h.listingSteps(propType)
	data  := map[string]string{"property_type": propType, "_step_idx": "0"}
	h.setSession(chatID, user.ID, "listing_wizard", data)
	step := steps[0]
	var kb interface{}
	if step.Keyboard != nil { kb = step.Keyboard() }
	h.tg.SendMessage(chatID, step.Prompt, kb)
}

func (h *BotHandler) handleSessionStep(msg *svc.TGMessage, user *model.User, session *BotSession, _ map[string]string) {
	if session.Step != "listing_wizard" && session.Step != "request_wizard" {
		h.sendMainMenu(msg.Chat.ID, "Ana Menü:")
		return
	}
	if session.Step == "listing_wizard" {
		h.listingWizardTextStep(msg, user, session)
	} else {
		h.requestWizardTextStep(msg, user, session)
	}
}

func (h *BotHandler) handleWizardCallback(cb *svc.TGCallback, user *model.User, session *BotSession, data string) {
	chatID := cb.Message.Chat.ID
	if session.Step == "listing_wizard" {
		h.listingWizardCallbackStep(chatID, user, session, data)
	} else if session.Step == "request_wizard" {
		h.requestWizardCallbackStep(chatID, user, session, data)
	}
}

func (h *BotHandler) listingWizardTextStep(msg *svc.TGMessage, user *model.User, session *BotSession) {
	chatID   := msg.Chat.ID
	propType := session.Data["property_type"]
	steps    := h.listingSteps(propType)
	idxStr   := session.Data["_step_idx"]
	idx, _   := strconv.Atoi(idxStr)

	if idx >= len(steps) { return }
	currentStep := steps[idx]

	if currentStep.Keyboard != nil {
		kb := currentStep.Keyboard()
		h.tg.SendMessage(chatID, "Lütfen aşağıdan seçin:", kb)
		return
	}

	val := strings.TrimSpace(msg.Text)
	if val == "-" { val = "" }
	session.Data[currentStep.Key] = val

	nextIdx := idx + 1
	session.Data["_step_idx"] = strconv.Itoa(nextIdx)
	h.saveSession(session)

	if nextIdx >= len(steps) {
		h.finalizeListing(chatID, user, session.Data)
		return
	}

	nextStep := steps[nextIdx]
	var kb interface{}
	if nextStep.Keyboard != nil {
		if nextStep.Key == "neighborhood" {
			district := session.Data["district"]
			hoods := h.cfg.NeighborhoodsFor(district)
			kb = svc.NeighborhoodKeyboard(hoods, "wiz_hood")
		} else {
			kb = nextStep.Keyboard()
		}
	}
	if kb != nil {
		h.tg.SendMessage(chatID, nextStep.Prompt, kb)
	} else {
		combined := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{{{Text: "❌ İptal", CallbackData: "wizard_cancel"}}}}
		h.tg.SendMessage(chatID, nextStep.Prompt, combined)
	}
}

func (h *BotHandler) listingWizardCallbackStep(chatID int64, user *model.User, session *BotSession, cbData string) {
	propType := session.Data["property_type"]
	steps    := h.listingSteps(propType)
	idx, _   := strconv.Atoi(session.Data["_step_idx"])
	if idx >= len(steps) { return }
	currentStep := steps[idx]

	prefixes := []string{"wiz_lt_", "wiz_dist_", "wiz_hood_", "wiz_rooms_", "wiz_zoning_"}
	val := cbData
	for _, p := range prefixes {
		if strings.HasPrefix(cbData, p) { val = strings.TrimPrefix(cbData, p); break }
	}
	session.Data[currentStep.Key] = val

	nextIdx := idx + 1
	session.Data["_step_idx"] = strconv.Itoa(nextIdx)
	h.saveSession(session)

	if nextIdx >= len(steps) {
		h.finalizeListing(chatID, user, session.Data)
		return
	}
	nextStep := steps[nextIdx]
	var kb interface{}
	if nextStep.Keyboard != nil {
		if nextStep.Key == "neighborhood" {
			district := session.Data["district"]
			hoods := h.cfg.NeighborhoodsFor(district)
			kb = svc.NeighborhoodKeyboard(hoods, "wiz_hood")
		} else {
			kb = nextStep.Keyboard()
		}
	}
	if kb != nil {
		h.tg.SendMessage(chatID, nextStep.Prompt, kb)
	} else {
		combined := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{{{Text: "❌ İptal", CallbackData: "wizard_cancel"}}}}
		h.tg.SendMessage(chatID, nextStep.Prompt, combined)
	}
}

func (h *BotHandler) finalizeListing(chatID int64, user *model.User, data map[string]string) {
	fields := map[string]string{
		"title":         data["title"],
		"listing_type":  data["listing_type"],
		"property_type": data["property_type"],
		"district":      data["district"],
		"neighborhood":  data["neighborhood"],
		"area_m2":       data["area_m2"],
		"rooms":         data["rooms"],
		"zoning":        data["zoning"],
		"price":         data["price"],
		"price_max":     data["price"],
		"description":   data["description"],
		"notes":         "Telegram ile eklendi. İletişim: " + data["contact"],
	}

	listing := &model.Listing{
		UserID:   user.ID,
		Fields:   fields,
		IsActive: true,
	}
	if err := h.listingRepo.Create(listing); err != nil {
		log.Printf("Bot ilan oluşturma hatası: %v", err)
		h.tg.SendMessage(chatID, "❌ İlan kaydedilirken hata oluştu.", nil)
		return
	}

	h.clearSession(chatID)
	h.tg.SendMessage(chatID,
		fmt.Sprintf("✅ <b>İlan Eklendi!</b>\n\nİlan No: #%d\nBaşlık: %s",
			listing.ListingNo, fields["title"]),
		svc.MainMenuKeyboard())
}

// ─── Talep Ekleme Sihirbazı ───────────────────────────────────

var requestSteps = []struct{ Key, Prompt string }{
	{"client_name",  "👤 Müşteri adını yazın:"},
	{"phone",        "📞 Telefon numarasını yazın:"},
	{"listing_type", ""},
	{"property_type", ""},
	{"district",     ""},
	{"neighborhood", ""},
	{"budget_min",   "💰 Minimum bütçe yazın (geçmek için - yazın):"},
	{"budget_max",   "💰 Maksimum bütçe yazın:"},
	{"notes",        "📝 Notlar (geçmek için - yazın):"},
}

func (h *BotHandler) startRequestWizard(chatID int64, user *model.User) {
	h.setSession(chatID, user.ID, "request_wizard", map[string]string{"_step_idx": "0"})
	h.tg.SendMessage(chatID, "ℹ️ İptal etmek için /iptal yazın.", nil)
	h.tg.SendMessage(chatID, requestSteps[0].Prompt, nil)
}

func (h *BotHandler) requestWizardTextStep(msg *svc.TGMessage, user *model.User, session *BotSession) {
	chatID := msg.Chat.ID
	idx, _ := strconv.Atoi(session.Data["_step_idx"])
	if idx >= len(requestSteps) { return }
	step := requestSteps[idx]

	if step.Prompt == "" {
		h.sendRequestStep(chatID, session, idx)
		return
	}

	val := strings.TrimSpace(msg.Text)
	if val == "-" { val = "" }
	session.Data[step.Key] = val

	nextIdx := idx + 1
	session.Data["_step_idx"] = strconv.Itoa(nextIdx)
	h.saveSession(session)

	if nextIdx >= len(requestSteps) {
		h.finalizeRequest(chatID, user, session.Data)
		return
	}
	h.sendRequestStep(chatID, session, nextIdx)
}

func (h *BotHandler) requestWizardCallbackStep(chatID int64, user *model.User, session *BotSession, cbData string) {
	idx, _ := strconv.Atoi(session.Data["_step_idx"])
	if idx >= len(requestSteps) { return }
	step := requestSteps[idx]

	prefixes := []string{"rwiz_lt_", "rwiz_pt_", "rwiz_dist_", "rwiz_hood_"}
	val := cbData
	for _, p := range prefixes {
		if strings.HasPrefix(cbData, p) { val = strings.TrimPrefix(cbData, p); break }
	}
	session.Data[step.Key] = val

	nextIdx := idx + 1
	session.Data["_step_idx"] = strconv.Itoa(nextIdx)
	h.saveSession(session)

	if nextIdx >= len(requestSteps) {
		h.finalizeRequest(chatID, user, session.Data)
		return
	}
	h.sendRequestStep(chatID, session, nextIdx)
}

func (h *BotHandler) sendRequestStep(chatID int64, session *BotSession, idx int) {
	if idx >= len(requestSteps) { return }
	step := requestSteps[idx]

	switch step.Key {
	case "listing_type":
		h.tg.SendMessage(chatID, "🏷️ Satılık mı, Kiralık mı?",
			svc.ListingTypeKeyboard("rwiz_lt"))
	case "property_type":
		h.tg.SendMessage(chatID, "🏠 Mülk tipi seçin:",
			svc.PropertyTypeKeyboard("rwiz_pt"))
	case "district":
		h.tg.SendMessage(chatID, "📍 İlçe tercihi seçin:",
			svc.DistrictKeyboard(h.cfg.Districts, "rwiz_dist"))
	case "neighborhood":
		district := session.Data["district"]
		hoods := h.cfg.NeighborhoodsFor(district)
		h.tg.SendMessage(chatID, "🏘️ Mahalle seçin (geçmek için 'Tümü'):",
			svc.NeighborhoodKeyboard(append([]string{"Tümü"}, hoods...), "rwiz_hood"))
	default:
		h.tg.SendMessage(chatID, step.Prompt, nil)
	}
}

func (h *BotHandler) finalizeRequest(chatID int64, user *model.User, data map[string]string) {
	fields := map[string]string{
		"client_name":   data["client_name"],
		"phone":         data["phone"],
		"listing_type":  data["listing_type"],
		"property_type": data["property_type"],
		"district":      data["district"],
		"neighborhood":  data["neighborhood"],
		"budget_min":    data["budget_min"],
		"budget_max":    data["budget_max"],
		"budget":        data["budget_max"],
		"notes":         data["notes"],
	}
	req := &model.Request{
		UserID:   user.ID,
		Fields:   fields,
		IsActive: true,
		NotifyMe: true,
	}
	if err := h.requestRepo.Create(req); err != nil {
		h.tg.SendMessage(chatID, "❌ Talep kaydedilirken hata oluştu.", nil)
		return
	}
	h.clearSession(chatID)
	h.tg.SendMessage(chatID,
		fmt.Sprintf("✅ <b>Talep Eklendi!</b>\n\nMüşteri: %s\nTelefon: %s",
			fields["client_name"], fields["phone"]),
		svc.MainMenuKeyboard())
}

// ─── Session yönetimi ─────────────────────────────────────────

type BotSession struct {
	ChatID int64
	UserID int64
	Step   string
	Data   map[string]string
}

func (h *BotHandler) getSession(chatID int64) *BotSession {
	var step string
	var dataJSON []byte
	var userID int64
	err := h.db.QueryRow(
		`SELECT user_id, step, data FROM bot_sessions WHERE chat_id=$1`, chatID,
	).Scan(&userID, &step, &dataJSON)
	if err != nil { return nil }
	data := map[string]string{}
	json.Unmarshal(dataJSON, &data)
	return &BotSession{ChatID: chatID, UserID: userID, Step: step, Data: data}
}

func (h *BotHandler) setSession(chatID, userID int64, step string, data map[string]string) {
	dataJSON, _ := json.Marshal(data)
	h.db.Exec(`
		INSERT INTO bot_sessions (chat_id, user_id, step, data, updated_at)
		VALUES ($1,$2,$3,$4,NOW())
		ON CONFLICT (chat_id) DO UPDATE
		SET user_id=$2, step=$3, data=$4, updated_at=NOW()`,
		chatID, userID, step, dataJSON)
}

func (h *BotHandler) saveSession(session *BotSession) {
	h.setSession(session.ChatID, session.UserID, session.Step, session.Data)
}

func (h *BotHandler) clearSession(chatID int64) {
	h.db.Exec(`DELETE FROM bot_sessions WHERE chat_id=$1`, chatID)
}

// ─── Kullanıcı yardımcıları ──────────────────────────────────

func (h *BotHandler) getUserByChatID(chatID int64) *model.User {
	u, err := h.userRepo.GetByTelegramChatID(chatID)
	if err != nil || u == nil { return nil }
	return u
}

func (h *BotHandler) setNotify(chatID, userID int64, on bool) {
	h.db.Exec(`UPDATE users SET notify_telegram=$1 WHERE id=$2`, on, userID)
}

func (h *BotHandler) sendMainMenu(chatID int64, intro string) {
	h.tg.SendMessage(chatID, intro, svc.MainMenuKeyboard())
}

func (h *BotHandler) districtKeyboardWithAll(prefix string) *svc.TGInlineKeyboard {
	districts := append([]string{"Tümü"}, h.cfg.Districts...)
	return svc.DistrictKeyboard(districts, prefix)
}

func (h *BotHandler) sendMyRequests(chatID int64, user *model.User) {
	requests, err := h.requestRepo.List(repository.RequestFilter{
		UserID: user.ID,
	})
	if err != nil || len(requests) == 0 {
		h.tg.SendMessage(chatID, "📭 Henüz talebiniz bulunmuyor.", svc.MainMenuKeyboard())
		return
	}

	h.tg.SendMessage(chatID,
		fmt.Sprintf("🎯 <b>Talepleriniz</b> (%d adet):", len(requests)), nil)

	for i, req := range requests {
		if i >= 10 { break }
		durum := "✅ Aktif"
		if !req.IsActive { durum = "⏸ Pasif" }
		notify := "🔔"
		if !req.NotifyMe { notify = "🔕" }

		budgetMax := req.Fields["budget_max"]
		if budgetMax == "" { budgetMax = req.Fields["budget"] }

		text := fmt.Sprintf(
			"<b>%s</b> %s %s\n"+
			"🏘️ %s / %s %s\n"+
			"📍 %s %s\n"+
			"💰 max %s ₺",
			req.Fields["client_name"],
			durum, notify,
			req.Fields["property_type"],
			req.Fields["listing_type"],
			req.Fields["rooms"],
			req.Fields["district"],
			req.Fields["neighborhood"],
			formatTGPrice(budgetMax),
		)
		h.tg.SendMessage(chatID, text, nil)
	}
	h.tg.SendMessage(chatID, "─────────────", svc.MainMenuKeyboard())
}
