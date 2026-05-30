package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/volkan1985t/EmlakPro/internal/model"
	"github.com/volkan1985t/EmlakPro/internal/repository"
	svc "github.com/volkan1985t/EmlakPro/internal/service"
)

// ─── BotHandler ──────────────────────────────────────────────

type BotHandler struct {
	cfg          *config.Config
	tg           *svc.TelegramService
	imageSvc     *svc.ImageService
	notifySvc    *svc.NotificationService
	userRepo     *repository.UserRepository
	listingRepo  *repository.ListingRepository
	requestRepo  *repository.RequestRepository
	taskRepo     *repository.TaskRepository
	customerRepo *repository.CustomerRepository
	db           *sql.DB
}

func NewBotHandler(
	cfg          *config.Config,
	tg           *svc.TelegramService,
	imageSvc     *svc.ImageService,
	notifySvc    *svc.NotificationService,
	db           *sql.DB,
	userRepo     *repository.UserRepository,
	listingRepo  *repository.ListingRepository,
	requestRepo  *repository.RequestRepository,
	taskRepo     *repository.TaskRepository,
	customerRepo *repository.CustomerRepository,
) *BotHandler {
	return &BotHandler{cfg: cfg, tg: tg, imageSvc: imageSvc, notifySvc: notifySvc, db: db,
		userRepo: userRepo, listingRepo: listingRepo, requestRepo: requestRepo,
		taskRepo: taskRepo, customerRepo: customerRepo}
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

	// Fotoğraf geldi mi?
	if len(msg.Photo) > 0 {
		session := h.getSession(chatID)
		if session != nil && session.Step == "listing_wizard" {
			h.handleListingPhoto(msg, user, session)
			return
		}
		h.tg.SendMessage(chatID, "📸 Fotoğraf alındı ama aktif bir ilan girişi yok. Önce ➕ İlan Gir seçin.", nil)
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
	chatID    := cb.Message.Chat.ID
	data      := cb.Data
	messageID := cb.Message.MessageID
	user      := h.getUserByChatID(chatID)

	// Her callback'te: spinner'ı temizle + keyboard'u kaldır
	h.tg.AnswerCallback(cb.ID)
	h.tg.RemoveKeyboard(chatID, messageID)

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

	case data == "wiz_photos_done":
		session := h.getSession(chatID)
		if session != nil && session.Step == "listing_wizard" {
			h.finalizeListing(chatID, user, session.Data)
		} else {
			h.tg.SendMessage(chatID, "⚠️ Aktif ilan girişi bulunamadı.", nil)
		}

	case data == "wizard_cancel":
		h.clearSession(chatID)
		h.tg.SendMessage(chatID, "❌ İşlem iptal edildi.", svc.MainMenuKeyboard())

	case data == "menu_tasks":
		h.sendMyTasks(chatID, user)

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

	// ── Geç butonu ───────────────────────────────────────────
	case strings.HasPrefix(data, "wiz_skip_"):
		session := h.getSession(chatID)
		if session != nil && session.Step == "listing_wizard" {
			stepKey  := strings.TrimPrefix(data, "wiz_skip_")
			propType := session.Data["property_type"]
			steps    := h.listingSteps(propType)
			idx, _   := strconv.Atoi(session.Data["_step_idx"])
			if idx < len(steps) && steps[idx].Key == stepKey {
				session.Data[stepKey] = ""
				nextIdx := idx + 1
				session.Data["_step_idx"] = strconv.Itoa(nextIdx)
				h.saveSession(session)
				if nextIdx >= len(steps) {
					h.finalizeListing(chatID, user, session.Data)
					return
				}
				h.sendNextListingStep(chatID, session, steps, nextIdx)
			}
		}

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
		wizardStep{Key: "_photos", Prompt: "📸 Fotoğraf gönderin (birden fazla gönderebilirsiniz).\nBitince ✅ Bitti butonuna basın.", Keyboard: func() interface{} {
			return &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
				{{Text: "✅ Bitti — İlanı Kaydet", CallbackData: "wiz_photos_done"}},
				{{Text: "❌ İptal", CallbackData: "wizard_cancel"}},
			}}
		}},
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

	if idx >= len(steps) {
		h.finalizeListing(chatID, user, session.Data)
		return
	}
	currentStep := steps[idx]

	// _photos adımına geldiyse — fotoğraf bekle, kaydet butonu göster
	if currentStep.Key == "_photos" {
		kb := currentStep.Keyboard()
		h.tg.SendMessage(chatID, currentStep.Prompt, kb)
		return
	}

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
		} else if nextStep.Key == "_photos" {
			// _photos adımına geçince doğrudan fotoğraf promptunu göster
			kb = nextStep.Keyboard()
			h.tg.SendMessage(chatID, nextStep.Prompt, kb)
			return
		} else {
			kb = nextStep.Keyboard()
		}
	}
	if kb != nil {
		h.tg.SendMessage(chatID, nextStep.Prompt, kb)
	} else {
		// Zorunlu olmayan metin adımlarında "Geç" butonu
		skipKeys := map[string]bool{"contact":true,"neighborhood":true,"area_m2":true,"description":true}
		if skipKeys[nextStep.Key] {
			h.tg.SendMessage(chatID, nextStep.Prompt, svc.SkipKeyboard(nextStep.Key))
		} else {
			combined := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{{{Text: "❌ İptal", CallbackData: "wizard_cancel"}}}}
			h.tg.SendMessage(chatID, nextStep.Prompt, combined)
		}
	}
}

func (h *BotHandler) listingWizardCallbackStep(chatID int64, user *model.User, session *BotSession, cbData string) {
	propType := session.Data["property_type"]
	steps    := h.listingSteps(propType)
	idx, _   := strconv.Atoi(session.Data["_step_idx"])
	if idx >= len(steps) {
		h.finalizeListing(chatID, user, session.Data)
		return
	}
	currentStep := steps[idx]

	// _photos adımındayken sadece wiz_photos_done beklenir — finalize handleCallback'te yapılır
	if currentStep.Key == "_photos" {
		return
	}

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
		} else if nextStep.Key == "_photos" {
			kb = nextStep.Keyboard()
			h.tg.SendMessage(chatID, nextStep.Prompt, kb)
			return
		} else {
			kb = nextStep.Keyboard()
		}
	}
	if kb != nil {
		h.tg.SendMessage(chatID, nextStep.Prompt, kb)
	} else {
		skipKeys := map[string]bool{"contact":true,"neighborhood":true,"area_m2":true,"description":true}
		if skipKeys[nextStep.Key] {
			h.tg.SendMessage(chatID, nextStep.Prompt, svc.SkipKeyboard(nextStep.Key))
		} else {
			combined := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{{{Text: "❌ İptal", CallbackData: "wizard_cancel"}}}}
			h.tg.SendMessage(chatID, nextStep.Prompt, combined)
		}
	}
}

// handleListingPhoto — wizard sırasında gelen fotoğrafı indir ve kaydet
// sendNextListingStep — wizard'da bir sonraki adımı göster
func (h *BotHandler) sendNextListingStep(chatID int64, session *BotSession, steps []wizardStep, idx int) {
	if idx >= len(steps) { return }
	nextStep := steps[idx]
	skipKeys := map[string]bool{"contact":true,"neighborhood":true,"area_m2":true,"description":true}
	if nextStep.Keyboard != nil {
		if nextStep.Key == "neighborhood" {
			hoods := h.cfg.NeighborhoodsFor(session.Data["district"])
			h.tg.SendMessage(chatID, nextStep.Prompt, svc.NeighborhoodKeyboard(hoods, "wiz_hood"))
		} else if nextStep.Key == "_photos" {
			h.tg.SendMessage(chatID, nextStep.Prompt, nextStep.Keyboard())
		} else {
			h.tg.SendMessage(chatID, nextStep.Prompt, nextStep.Keyboard())
		}
	} else if skipKeys[nextStep.Key] {
		h.tg.SendMessage(chatID, nextStep.Prompt, svc.SkipKeyboard(nextStep.Key))
	} else {
		combined := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{{{Text: "❌ İptal", CallbackData: "wizard_cancel"}}}}
		h.tg.SendMessage(chatID, nextStep.Prompt, combined)
	}
}

func (h *BotHandler) handleListingPhoto(msg *svc.TGMessage, user *model.User, session *BotSession) {
	chatID := msg.Chat.ID
	idxStr := session.Data["_step_idx"]
	steps  := h.listingSteps(session.Data["property_type"])
	idx, _ := strconv.Atoi(idxStr)

	// Henüz fotoğraf adımına gelmediyse bildir
	if idx < len(steps) && steps[idx].Key != "_photos" {
		h.tg.SendMessage(chatID, "⚠️ Önce diğer adımları tamamlayın.", nil)
		return
	}

	// En büyük boyutlu fotoğrafı al (son eleman)
	photo := msg.Photo[len(msg.Photo)-1]
	fileURL, err := h.tg.GetFileURL(photo.FileID)
	if err != nil {
		log.Printf("[bot] fotoğraf URL alınamadı: %v", err)
		h.tg.SendMessage(chatID, "❌ Fotoğraf alınamadı, tekrar deneyin.", nil)
		return
	}
	data, err := h.tg.DownloadFile(fileURL)
	if err != nil {
		log.Printf("[bot] fotoğraf indirilemedi: %v", err)
		h.tg.SendMessage(chatID, "❌ Fotoğraf indirilemedi.", nil)
		return
	}

	// Fotoğraf sayısını kontrol et (max 8)
	photoCount, _ := strconv.Atoi(session.Data["_photo_count"])
	if photoCount >= 8 {
		h.tg.SendMessage(chatID, "⚠️ Maksimum 8 fotoğraf ekleyebilirsiniz. ✅ Bitti butonuna basın.", nil)
		return
	}

	// Geçici olarak session'da file_id listesi tut
	existing := session.Data["_photo_ids"]
	if existing != "" {
		existing += ","
	}
	session.Data["_photo_ids"]    = existing + photo.FileID
	session.Data["_photo_count"]  = strconv.Itoa(photoCount + 1)
	// Ham byte'ı base64 olarak sakla (küçük fotoğraflar için)
	_ = data // ileride direkt kullanılacak
	h.saveSession(session)

	photoCount++
	h.tg.SendMessage(chatID,
		fmt.Sprintf("✅ Fotoğraf %d eklendi. Devam gönderin veya bitirmek için butona basın.", photoCount),
		&svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
			{{Text: "✅ Bitti — İlanı Kaydet", CallbackData: "wiz_photos_done"}},
		}})
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
		log.Printf("[BOT][HATA] ilan oluşturma: %v | user=%d fields=%v", err, user.ID, fields)
		h.clearSession(chatID)
		h.tg.SendMessage(chatID, fmt.Sprintf("❌ İlan kaydedilemedi: %v\nLütfen tekrar deneyin.", err), svc.MainMenuKeyboard())
		return
	}
	log.Printf("[BOT] ilan oluşturuldu: id=%d no=%d user=%d", listing.ID, listing.ListingNo, user.ID)

	// Tüm kullanıcılara bildirim gönder
	if h.notifySvc != nil {
		go h.sendBotListingNotification(listing, user)
	}

	// Fotoğrafları indir ve kaydet
	photoIDs := data["_photo_ids"]
	var coverSaved bool
	if photoIDs != "" {
		for i, fileID := range strings.Split(photoIDs, ",") {
			if fileID == "" { continue }
			fileURL, err := h.tg.GetFileURL(fileID)
			if err != nil {
				log.Printf("[bot] foto URL hatası: %v", err)
				continue
			}
			imgData, err := h.tg.DownloadFile(fileURL)
			if err != nil {
				log.Printf("[bot] foto indirme hatası: %v", err)
				continue
			}
			reader := bytes.NewReader(imgData)
			if i == 0 {
				res, err := h.imageSvc.SaveCover(reader, "tg.jpg", fields["property_type"], listing.ListingNo)
				if err == nil {
					h.listingRepo.UpdateCoverImage(listing.ID, res.Path)
					coverSaved = true
				}
			} else {
				res, err := h.imageSvc.SaveGallery(reader, "tg.jpg", fields["property_type"], listing.ListingNo)
				if err == nil {
					h.listingRepo.AddImage(listing.ID, res.Path, i)
				}
			}
		}
	}
	_ = coverSaved

	// Otomatik "İlan Kontrol" görevi oluştur
	go func() {
		tomorrow := time.Now().Add(24 * time.Hour)
		_, err := h.taskRepo.Create(&model.CreateTaskRequest{
			Title:       fmt.Sprintf("İlan Kontrol: #%d %s", listing.ListingNo, fields["title"]),
			Description: fmt.Sprintf("Telegram üzerinden eklenen ilan kontrol edilmeli.\nİlan No: #%d\nMülk: %s / %s\nFiyat: %s TL", listing.ListingNo, fields["property_type"], fields["district"], fields["price"]),
			Status:      "bekliyor",
			Priority:    "normal",
			DueDate:     &tomorrow,
			Assignees:   []int64{user.ID},
		}, user.ID)
		if err != nil {
			log.Printf("[bot] ilan kontrol görevi oluşturulamadı: %v", err)
		}
	}()

	h.clearSession(chatID)

	// İlan linki (share token ile)
	baseURL := strings.TrimRight(h.cfg.App.BaseURL, "/")
	var linkLine string
	if baseURL != "" && listing.ShareToken != "" {
		listingURL := fmt.Sprintf("%s/api/listings/share/%s", baseURL, listing.ShareToken)
		linkLine = fmt.Sprintf("\n\n🔗 <a href=\"%s\">İlanı Görüntüle</a>", listingURL)
	}

	h.tg.SendMessage(chatID,
		fmt.Sprintf("✅ <b>İlan Eklendi!</b>\n\nİlan No: #%d\nBaşlık: %s\n\n📋 Kontrol görevi oluşturuldu.%s",
			listing.ListingNo, fields["title"], linkLine),
		svc.MainMenuKeyboard())
}

// ─── Talep Ekleme Sihirbazı ───────────────────────────────────

var requestSteps = []struct{ Key, Prompt string }{
	{"_customer",    ""},        // müşteri seç veya elle yaz
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
	h.sendRequestStep(chatID, &BotSession{ChatID: chatID, UserID: user.ID, Step: "request_wizard",
		Data: map[string]string{"_step_idx": "0"}}, 0, user.ID)
}

func (h *BotHandler) requestWizardTextStep(msg *svc.TGMessage, user *model.User, session *BotSession) {
	chatID := msg.Chat.ID
	idx, _ := strconv.Atoi(session.Data["_step_idx"])
	if idx >= len(requestSteps) { return }
	step := requestSteps[idx]

	// _customer adımında: buton kullanmadıysa elle ad yazıyor
	if step.Key == "_customer" {
		text := strings.TrimSpace(msg.Text)
		if text == "" || text == "-" {
			h.tg.SendMessage(chatID, "👤 Müşteri adını yazın:", nil)
			return
		}
		// "Ad Soyad · Tel" formatında mı geldi (listeden seçim)
		parts := strings.SplitN(text, " · ", 2)
		session.Data["client_name"] = strings.TrimSpace(parts[0])
		if len(parts) == 2 {
			session.Data["phone"] = strings.TrimSpace(parts[1])
		} else {
			// Sadece isim — telefon sonra sorulacak
			session.Data["phone"] = ""
			// Telefon adımı için ek sorgu
			nextIdx := idx + 1
			session.Data["_step_idx"] = strconv.Itoa(nextIdx)
			h.saveSession(session)
			h.tg.SendMessage(chatID, "📞 Telefon numarasını yazın:", nil)
			return
		}
		nextIdx := idx + 1
		session.Data["_step_idx"] = strconv.Itoa(nextIdx)
		h.saveSession(session)
		h.sendRequestStep(chatID, session, nextIdx, session.UserID)
		return
	}

	if step.Prompt == "" {
		h.sendRequestStep(chatID, session, idx, session.UserID)
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
	h.sendRequestStep(chatID, session, nextIdx, session.UserID)
}

func (h *BotHandler) requestWizardCallbackStep(chatID int64, user *model.User, session *BotSession, cbData string) {
	idx, _ := strconv.Atoi(session.Data["_step_idx"])
	if idx >= len(requestSteps) { return }
	step := requestSteps[idx]

	prefixes := []string{"rwiz_lt_", "rwiz_pt_", "rwiz_dist_", "rwiz_hood_", "rwiz_cust_"}
	val := cbData
	for _, p := range prefixes {
		if strings.HasPrefix(cbData, p) { val = strings.TrimPrefix(cbData, p); break }
	}
	// Müşteri seçimi: "Ad · Tel" veya "rwiz_cust_elle" (elle yazacak)
	if step.Key == "_customer" {
		if cbData == "rwiz_cust_elle" {
			h.tg.SendMessage(chatID, "👤 Müşteri adını yazın:", nil)
			return
		}
		// "Ad Soyad · Tel" ayrıştır
		parts := strings.SplitN(val, " · ", 2)
		session.Data["client_name"] = strings.TrimSpace(parts[0])
		if len(parts) == 2 {
			session.Data["phone"] = strings.TrimSpace(parts[1])
		}
		nextIdx := idx + 1
		session.Data["_step_idx"] = strconv.Itoa(nextIdx)
		h.saveSession(session)
		h.sendRequestStep(chatID, session, nextIdx, session.UserID)
		return
	}
	session.Data[step.Key] = val

	nextIdx := idx + 1
	session.Data["_step_idx"] = strconv.Itoa(nextIdx)
	h.saveSession(session)

	if nextIdx >= len(requestSteps) {
		h.finalizeRequest(chatID, user, session.Data)
		return
	}
	h.sendRequestStep(chatID, session, nextIdx, session.UserID)
}

func (h *BotHandler) sendRequestStep(chatID int64, session *BotSession, idx int, userID ...int64) {
	if idx >= len(requestSteps) { return }
	step := requestSteps[idx]

	switch step.Key {
	case "_customer":
		// Son 10 müşteriyi listele + "Elle yaz" seçeneği
		uid := int64(0); if len(userID) > 0 { uid = userID[0] }
		customers, _ := h.customerRepo.List(uid, false, "")
		if len(customers) == 0 {
			h.tg.SendMessage(chatID, "👤 Müşteri adını yazın:", nil)
		} else {
			h.tg.SendMessage(chatID, "👤 Müşteri seçin veya 'Elle yaz' deyin:",
				h.customerPickerKeyboard(customers[:intMin(len(customers), 10)]))
		}
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
	// Müşteri yoksa otomatik oluştur
	if fields["client_name"] != "" {
		customers, _ := h.customerRepo.List(user.ID, false, fields["client_name"])
		var custID int64
		for _, c := range customers {
			if strings.EqualFold(c.Name, fields["client_name"]) {
				custID = c.ID
				break
			}
		}
		if custID == 0 {
			newC := &model.Customer{UserID: user.ID, Name: fields["client_name"], Phone: fields["phone"]}
			if err := h.customerRepo.Create(newC); err == nil {
				custID = newC.ID
			}
		}
		if custID > 0 {
			fields["customer_id"] = strconv.FormatInt(custID, 10)
		}
	}

	req := &model.Request{
		UserID:   user.ID,
		Fields:   fields,
		IsActive: true,
		NotifyMe: true,
	}
	if err := h.requestRepo.Create(req); err != nil {
		h.clearSession(chatID)
		h.tg.SendMessage(chatID, "❌ Talep kaydedilirken hata oluştu. Lütfen tekrar deneyin.", svc.MainMenuKeyboard())
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

func (h *BotHandler) sendBotListingNotification(listing *model.Listing, owner *model.User) {
	usersWithChat, err := h.userRepo.ListWithChatIDs()
	if err != nil {
		log.Printf("[bot-notify] ListWithChatIDs: %v", err)
		return
	}
	var allUsers []svc.UserForNotify
	for _, u := range usersWithChat {
		chatID, _ := strconv.ParseInt(u.TelegramChatID, 10, 64)
		if chatID == 0 { continue }
		allUsers = append(allUsers, svc.UserForNotify{
			ID:         u.ID,
			ChatID:     chatID,
			NotifyType: "all",
		})
	}
	reqs, _ := h.requestRepo.List(repository.RequestFilter{OnlyActive: true})
	var requests []svc.RequestForMatch
	for _, req := range reqs {
		if !req.NotifyMe { continue }
		requests = append(requests, svc.RequestForMatch{
			ID:     req.ID,
			UserID: req.UserID,
			Fields: req.Fields,
		})
	}
	lm := svc.ListingForMatch{
		ID:        listing.ID,
		ListingNo: listing.ListingNo,
		UserID:    listing.UserID,
		OwnerID:   listing.UserID,
		OwnerName: owner.FullName,
		IsActive:  listing.IsActive,
		Fields:    listing.Fields,
	}
	h.notifySvc.NotifyNewListing(lm, allUsers, requests)
}

func intMin(a, b int) int { if a < b { return a }; return b }

func (h *BotHandler) customerPickerKeyboard(customers []model.Customer) *svc.TGInlineKeyboard {
	var rows [][]svc.TGInlineButton
	for _, c := range customers {
		label := c.Name
		if c.Phone != "" { label += " · " + c.Phone }
		rows = append(rows, []svc.TGInlineButton{{
			Text:         label,
			CallbackData: "rwiz_cust_" + label,
		}})
	}
	rows = append(rows,
		[]svc.TGInlineButton{{Text: "✏️ Elle yaz", CallbackData: "rwiz_cust_elle"}},
		[]svc.TGInlineButton{{Text: "❌ İptal", CallbackData: "wizard_cancel"}},
	)
	return &svc.TGInlineKeyboard{InlineKeyboard: rows}
}

func (h *BotHandler) sendMainMenu(chatID int64, intro string) {
	h.tg.SendMessage(chatID, intro, svc.MainMenuKeyboard())
}

func (h *BotHandler) districtKeyboardWithAll(prefix string) *svc.TGInlineKeyboard {
	districts := append([]string{"Tümü"}, h.cfg.Districts...)
	return svc.DistrictKeyboard(districts, prefix)
}

func (h *BotHandler) sendMyTasks(chatID int64, user *model.User) {
	tasks, err := h.taskRepo.List(model.TaskFilter{UserID: user.ID})
	if err != nil || len(tasks) == 0 {
		h.tg.SendMessage(chatID, "📭 Atanmış göreviniz bulunmuyor.", nil)
		return
	}

	h.tg.SendMessage(chatID, fmt.Sprintf("✅ <b>Görevlerim</b> (%d adet):", len(tasks)), nil)

	for i, t := range tasks {
		if i >= 10 { break }
		statusEmoji := map[string]string{
			"bekliyor": "⏳", "devam_ediyor": "🔄", "tamamlandi": "✅", "iptal": "❌",
		}
		priEmoji := map[string]string{
			"dusuk": "🟢", "normal": "🔵", "yuksek": "🟠", "acil": "🔴",
		}
		em := statusEmoji[t.Status]
		if em == "" { em = "📋" }
		pr := priEmoji[t.Priority]
		if pr == "" { pr = "🔵" }

		due := ""
		if t.DueDate != nil {
			due = "\n📅 " + t.DueDate.Format("02.01.2006")
		}
		h.tg.SendMessage(chatID,
			fmt.Sprintf("%s %s <b>%s</b>%s\n<i>%s</i>",
				em, pr, t.Title, due, t.Description), nil)
	}
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
	h.tg.SendMessage(chatID, "─────────────\nAna menüye dönmek için /menu yazın.", nil)
}
