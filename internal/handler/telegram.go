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
	interestRepo *repository.InterestRepository
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
	interestRepo *repository.InterestRepository,
) *BotHandler {
	return &BotHandler{cfg: cfg, tg: tg, imageSvc: imageSvc, notifySvc: notifySvc, db: db,
		userRepo: userRepo, listingRepo: listingRepo, requestRepo: requestRepo,
		taskRepo: taskRepo, customerRepo: customerRepo, interestRepo: interestRepo}
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

	// Yeni işlem başlangıcında ayraç şeridi (wizard ortasında değilse)
	if s := h.getSession(chatID); s == nil || s.Step == "" || s.Step == "idle" {
		h.tg.SendSeparator(chatID)
	}

	// Fotoğraf geldi mi?
	if len(msg.Photo) > 0 {
		session := h.getSession(chatID)
		if session != nil && session.Step == "listing_wizard" {
			h.handleListingPhoto(msg, user, session)
			return
		}
		h.tg.SendMessage(chatID, "📸 Fotoğraf alındı ama aktif bir ilan girişi yok. Önce '🏠 İlan Ekle / İlanlar' → Yeni İlan seçin.", nil)
		return
	}

	// Rehberden kişi paylaşıldı mı? → ne yapılacağını sor
	if msg.Contact != nil {
		h.handleSharedContact(chatID, user, msg.Contact)
		return
	}

	// Reply keyboard ana buton metni mi? Öyleyse yarım kalan session'ı iptal et.
	// (Aksi halde takılı bir session buton metnini yutar.)
	mainButtons := map[string]bool{
		"🏠 İlan Ekle / İlanlar": true, "🎯 Talep Ekle / Talepler": true,
		"👥 Müşteri Ekle / Müşteriler": true, "✅ Görev Ekle / Görevler": true,
		"📞 İlgi Ekle / İlgiler": true, "⚙️ Ayarlar / Bildirimler": true,
	}
	if mainButtons[text] {
		h.clearSession(chatID)
	} else {
		// Aktif session var mı? (sadece ana buton DEĞİLSE session'a yönlendir)
		session := h.getSession(chatID)
		if session != nil && session.Step != "idle" {
			h.handleSessionStep(msg, user, session, session.Data)
			return
		}
	}

	// Reply keyboard buton metinleri → ilgili işleme yönlendir
	switch text {
	case "🏠 İlan Ekle / İlanlar":
		h.sendPairMenu(chatID, "🏠 <b>İlan</b>\nNe yapmak istersiniz?",
			"➕ Yeni İlan", "menu_add_listing", "📋 İlanlar", "menu_mine")
		return
	case "🎯 Talep Ekle / Talepler":
		h.sendPairMenu(chatID, "🎯 <b>Talep</b>\nNe yapmak istersiniz?",
			"➕ Talep Ekle", "menu_add_request", "📂 Talepler", "menu_my_requests")
		return
	case "👥 Müşteri Ekle / Müşteriler":
		h.sendPairMenu(chatID, "👥 <b>Müşteri</b>\nNe yapmak istersiniz?",
			"➕ Müşteri Ekle", "cust_add", "📋 Müşterilerim", "cust_list")
		return
	case "✅ Görev Ekle / Görevler":
		h.sendPairMenu(chatID, "✅ <b>Görev</b>\nNe yapmak istersiniz?",
			"➕ Görev Ekle", "task_add_soon", "📋 Görevler", "menu_tasks")
		return
	case "📞 İlgi Ekle / İlgiler":
		h.sendPairMenu(chatID, "📞 <b>İlan İlgileri</b>\nNe yapmak istersiniz?",
			"➕ İlgi Ekle", "int_add", "📋 İlgilerim", "int_list")
		return
	case "⚙️ Ayarlar / Bildirimler":
		h.sendPairMenu(chatID, "⚙️ <b>Ayarlar</b>\nNe yapmak istersiniz?",
			"⚙️ Ayarlar", "settings_soon", "🔔 Bildirimler", "menu_notify")
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

	// Yeni işlem başlangıcında ayraç şeridi (wizard ortasında değilse)
	if s := h.getSession(chatID); s == nil || s.Step == "" || s.Step == "idle" {
		h.tg.SendSeparator(chatID)
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
		h.startMineFilter(chatID, user)

	// ── İlanlar drill-down filtresi ───────────────────────────
	case strings.HasPrefix(data, "mf_scope_"):
		scope := strings.TrimPrefix(data, "mf_scope_") // all | mine
		h.mineFilterSet(chatID, user, "scope", scope)
		h.tg.SendMessage(chatID, "🏷️ <b>İlan tipi</b> seçin:", svc.ListingTypeKeyboard("mf_lt"))

	case strings.HasPrefix(data, "mf_lt_"):
		val := strings.TrimPrefix(data, "mf_lt_")
		h.mineFilterSet(chatID, user, "listing_type", val)
		h.tg.SendMessage(chatID, "🏘️ <b>Mülk tipi</b> seçin:", svc.PropertyTypeKeyboard("mf_pt"))

	case strings.HasPrefix(data, "mf_pt_"):
		val := strings.TrimPrefix(data, "mf_pt_")
		h.mineFilterSet(chatID, user, "property_type", val)
		switch val {
		case "Daire":
			opts := append([]string{"Hepsi"}, h.cfg.RoomOptions...)
			h.tg.SendMessage(chatID, "🛏️ <b>Oda sayısı</b> seçin:", svc.OptionsKeyboard(opts, "mf_rm"))
		case "Arsa":
			opts := append([]string{"Hepsi"}, h.cfg.ZoningOptions...)
			h.tg.SendMessage(chatID, "📋 <b>İmar durumu</b> seçin:", svc.OptionsKeyboard(opts, "mf_zn"))
		case "Villa", "Ticari":
			opts := append([]string{"Hepsi"}, h.cfg.Districts...)
			h.tg.SendMessage(chatID, "📍 <b>İlçe</b> seçin:", svc.OptionsKeyboard(opts, "mf_ds"))
		default:
			h.sendMineFilterResults(chatID, user)
		}

	case strings.HasPrefix(data, "mf_rm_"):
		val := strings.TrimPrefix(data, "mf_rm_")
		if val != "Hepsi" { h.mineFilterSet(chatID, user, "rooms", val) }
		h.sendMineFilterResults(chatID, user)

	case strings.HasPrefix(data, "mf_zn_"):
		val := strings.TrimPrefix(data, "mf_zn_")
		if val != "Hepsi" { h.mineFilterSet(chatID, user, "zoning", val) }
		opts := append([]string{"Hepsi"}, h.cfg.Districts...)
		h.tg.SendMessage(chatID, "📍 <b>İlçe</b> seçin:", svc.OptionsKeyboard(opts, "mf_ds"))

	case strings.HasPrefix(data, "mf_ds_"):
		val := strings.TrimPrefix(data, "mf_ds_")
		if val != "Hepsi" { h.mineFilterSet(chatID, user, "district", val) }
		sess := h.getSession(chatID)
		pt := ""
		if sess != nil && sess.Data != nil { pt = sess.Data["property_type"] }
		// Ticari → ilçe yeterli; ilçe seçilmediyse mahalle sorma
		if pt == "Ticari" || val == "Hepsi" {
			h.sendMineFilterResults(chatID, user)
		} else {
			hoods := append([]string{"Hepsi"}, h.cfg.NeighborhoodsFor(val)...)
			h.tg.SendMessage(chatID, "🏠 <b>Mahalle</b> seçin:", svc.OptionsKeyboard(hoods, "mf_nb"))
		}

	case strings.HasPrefix(data, "mf_nb_"):
		val := strings.TrimPrefix(data, "mf_nb_")
		if val != "Hepsi" { h.mineFilterSet(chatID, user, "neighborhood", val) }
		h.sendMineFilterResults(chatID, user)

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
			// Bildirim sorusu — finalize öncesi
			kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
				{{Text: "🔔 Evet, herkese bildir", CallbackData: "wiz_notify_yes"}},
				{{Text: "🔕 Hayır, bildirme", CallbackData: "wiz_notify_no"}},
			}}
			h.tg.SendMessage(chatID, "Bu ilanı tüm danışmanlara bildireyim mi?\n\n<i>Hayır deseniz de eşleşen aktif talep sahiplerine bilgi verilir.</i>", kb)
		} else {
			h.tg.SendMessage(chatID, "⚠️ Aktif ilan girişi bulunamadı.", nil)
		}

	case data == "wiz_notify_yes":
		session := h.getSession(chatID)
		if session != nil && session.Step == "listing_wizard" {
			h.finalizeListing(chatID, user, session.Data, true)
		} else {
			h.tg.SendMessage(chatID, "⚠️ Aktif ilan girişi bulunamadı.", nil)
		}

	case data == "wiz_notify_no":
		session := h.getSession(chatID)
		if session != nil && session.Step == "listing_wizard" {
			h.finalizeListing(chatID, user, session.Data, false)
		} else {
			h.tg.SendMessage(chatID, "⚠️ Aktif ilan girişi bulunamadı.", nil)
		}

	case data == "wizard_cancel":
		h.clearSession(chatID)
		h.tg.SendMessage(chatID, "❌ İşlem iptal edildi.", svc.MainMenuKeyboard())

	case data == "menu_tasks":
		h.sendMyTasks(chatID, user)

	// ── Müşteriler ────────────────────────────────────────────
	case data == "cust_add":
		h.setSession(chatID, user.ID, "cust_add", map[string]string{"_cstep": "name"})
		h.tg.SendMessage(chatID, "➕ <b>Yeni Müşteri</b>\nMüşteri adını yazın:", nil)

	case data == "cust_list":
		h.sendCustomerList(chatID, user, "")

	case data == "cust_search":
		h.setSession(chatID, user.ID, "cust_search", map[string]string{})
		h.tg.SendMessage(chatID, "🔍 <b>Müşteri Ara</b>\nAd veya telefonun ilk hanelerini yazın:", nil)

	case strings.HasPrefix(data, "cust_del_"):
		idStr := strings.TrimPrefix(data, "cust_del_")
		cid, _ := strconv.ParseInt(idStr, 10, 64)
		cu, _ := h.customerRepo.GetByID(cid)
		if cu == nil || cu.UserID != user.ID {
			h.tg.SendMessage(chatID, "⚠️ Müşteri bulunamadı veya yetkiniz yok.", nil)
			break
		}
		kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
			{{Text: "🗑 Evet, sil", CallbackData: fmt.Sprintf("cust_delyes_%d", cid)}},
			{{Text: "↩️ Vazgeç", CallbackData: "cust_delno"}},
		}}
		h.tg.SendMessage(chatID, fmt.Sprintf("⚠️ <b>%s</b> silinsin mi?", cu.Name), kb)

	case strings.HasPrefix(data, "cust_delyes_"):
		idStr := strings.TrimPrefix(data, "cust_delyes_")
		cid, _ := strconv.ParseInt(idStr, 10, 64)
		cu, _ := h.customerRepo.GetByID(cid)
		if cu == nil || cu.UserID != user.ID {
			h.tg.SendMessage(chatID, "⚠️ Müşteri bulunamadı veya yetkiniz yok.", nil)
			break
		}
		if err := h.customerRepo.Delete(cid); err != nil {
			h.tg.SendMessage(chatID, "❌ Silinemedi: "+err.Error(), nil)
		} else {
			h.tg.SendMessage(chatID, fmt.Sprintf("✅ <b>%s</b> silindi.", cu.Name), nil)
		}

	case data == "cust_delno":
		h.tg.SendMessage(chatID, "↩️ İşlem iptal edildi.", nil)

	// ── Paylaşılan kişi (rehberden) ───────────────────────────
	case data == "sc_cancel":
		h.clearSession(chatID)
		h.tg.SendMessage(chatID, "↩️ İptal edildi.", nil)

	case data == "sc_customer":
		s := h.getSession(chatID)
		name, phone, email := "", "", ""
		if s != nil && s.Data != nil {
			name = s.Data["shared_name"]; phone = s.Data["shared_phone"]; email = s.Data["shared_email"]
		}
		h.clearSession(chatID)
		if name == "" { name = phone }
		if dup, _ := h.customerRepo.FindDuplicate(user.ID, name, phone); dup != nil {
			h.tg.SendMessage(chatID, fmt.Sprintf("ℹ️ <b>%s</b> zaten kayıtlı.", dup.Name), nil)
			break
		}
		newC := &model.Customer{UserID: user.ID, Name: name, Phone: phone, Email: email}
		if err := h.customerRepo.Create(newC); err != nil {
			h.tg.SendMessage(chatID, "❌ Müşteri eklenemedi: "+err.Error(), nil)
		} else {
			h.tg.SendMessage(chatID, fmt.Sprintf("✅ <b>%s</b> müşteri olarak kaydedildi.", name), nil)
		}

	case data == "sc_request":
		s := h.getSession(chatID)
		name, phone := "", ""
		if s != nil && s.Data != nil { name = s.Data["shared_name"]; phone = s.Data["shared_phone"] }
		// Talep wizard'ı başlat, müşteri adımını atla (önceden dolu)
		d := map[string]string{"_step_idx": "1", "client_name": name, "phone": phone}
		h.setSession(chatID, user.ID, "request_wizard", d)
		h.tg.SendMessage(chatID, fmt.Sprintf("🎯 <b>Yeni Talep</b> — Müşteri: <b>%s</b>\nℹ️ İptal için /iptal", name), nil)
		h.sendRequestStep(chatID, &BotSession{ChatID: chatID, UserID: user.ID, Step: "request_wizard", Data: d}, 1, user.ID)

	// ── İlan İlgileri (lead/teklif) ────────────────────────────
	case data == "int_add":
		h.startInterestWizard(chatID, user)
	case data == "int_list":
		h.sendInterestList(chatID, user, false)
	case data == "int_today":
		h.sendInterestList(chatID, user, true)
	case strings.HasPrefix(data, "iwz_lst_"):
		h.interestWizardListing(chatID, user, strings.TrimPrefix(data, "iwz_lst_"))
	case strings.HasPrefix(data, "iwz_tip_"):
		h.interestWizardTip(chatID, user, strings.TrimPrefix(data, "iwz_tip_"))
	case strings.HasPrefix(data, "int_adv_"):
		h.interestAdvance(chatID, user, strings.TrimPrefix(data, "int_adv_"))
	case strings.HasPrefix(data, "int_won_"):
		h.interestSetWon(chatID, user, strings.TrimPrefix(data, "int_won_"))
	case strings.HasPrefix(data, "int_lost_"):
		h.interestSetOutcome(chatID, user, strings.TrimPrefix(data, "int_lost_"), "kaybedildi")
	case strings.HasPrefix(data, "int_cust_"):
		h.interestToCustomer(chatID, user, strings.TrimPrefix(data, "int_cust_"))
	case strings.HasPrefix(data, "int_pkg_"):
		h.interestTaskPackage(chatID, user, strings.TrimPrefix(data, "int_pkg_"))

	case data == "sc_listing":
		s := h.getSession(chatID)
		name, phone := "", ""
		if s != nil && s.Data != nil { name = s.Data["shared_name"]; phone = s.Data["shared_phone"] }
		contact := name
		if phone != "" { contact = name + " " + phone }
		// İlan wizard'ı başlat, contact önceden dolu
		d := map[string]string{"_step_idx": "0", "contact": contact}
		h.setSession(chatID, user.ID, "listing_wizard", d)
		h.tg.SendMessage(chatID, fmt.Sprintf("🏠 <b>Yeni Portföy</b> — Malik: <b>%s</b>\nℹ️ İptal için /iptal", name), nil)
		h.showListingStepPrompt(chatID, &BotSession{ChatID: chatID, UserID: user.ID, Step: "listing_wizard", Data: d}, 0)

	case data == "sc_interest":
		s := h.getSession(chatID)
		name, phone := "", ""
		if s != nil && s.Data != nil { name = s.Data["shared_name"]; phone = s.Data["shared_phone"] }
		// İlgi wizard'ı başlat, alıcı önceden dolu → ilan seçiminden devam
		d := map[string]string{"_iwz": "listing", "buyer_name": name, "buyer_phone": phone}
		h.setSession(chatID, user.ID, "interest_wizard", d)
		h.tg.SendMessage(chatID, fmt.Sprintf("📞 <b>Yeni İlgi</b> — Alıcı: <b>%s</b>", name), nil)
		h.sendInterestListingPicker(chatID, user)

	case data == "sc_task":
		h.clearSession(chatID)
		h.tg.SendMessage(chatID, "🚧 <b>Görev Ekle</b> yakında eklenecek.", nil)

	// ── Bildirimler ───────────────────────────────────────────
	case data == "task_add_soon":
		h.tg.SendMessage(chatID, "🚧 <b>Görev Ekle</b> yakında eklenecek.\nŞimdilik görevleri web arayüzünden ekleyebilirsiniz.", nil)
	case data == "settings_soon":
		h.tg.SendMessage(chatID, "🚧 <b>Ayarlar</b> yakında eklenecek.", nil)

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
					h.askListingNotify(chatID)
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
	h.sendListingsFiltered(chatID, user, f)
}

// sendListingsFiltered — verilen filtreyle ilanları listeler
func (h *BotHandler) sendListingsFiltered(chatID int64, user *model.User, f repository.ListFilter) {
	listings, err := h.listingRepo.List(f)
	if err != nil || len(listings) == 0 {
		h.tg.SendMessage(chatID, "📭 Bu kriterlere uygun ilan bulunamadı.", nil)
		return
	}

	// Başlık — seçili filtreleri özetle
	var parts []string
	if f.ListingType != "" { parts = append(parts, f.ListingType) }
	if f.PropertyType != "" { parts = append(parts, f.PropertyType) }
	if f.Rooms != "" { parts = append(parts, f.Rooms) }
	if f.Zoning != "" { parts = append(parts, f.Zoning) }
	if f.District != "" { parts = append(parts, f.District) }
	if f.Neighborhood != "" { parts = append(parts, f.Neighborhood) }
	baslik := "Tüm İlanlar"
	if f.OnlyMine { baslik = "İlanlarım" }
	title := "📋 <b>" + baslik
	if len(parts) > 0 { title += " — " + strings.Join(parts, " · ") }
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

// handleSharedContact — rehberden paylaşılan kişi geldiğinde ne yapılacağını sorar
func (h *BotHandler) handleSharedContact(chatID int64, user *model.User, ct *svc.TGContact) {
	name := strings.TrimSpace(ct.FirstName + " " + ct.LastName)
	phone := ct.PhoneNumber
	// vCard'dan email ayıkla (varsa)
	email := extractVCardEmail(ct.VCard)

	// Paylaşılan kişiyi session'a yaz (seçim sonrası kullanılacak)
	data := map[string]string{
		"shared_name":  name,
		"shared_phone": phone,
		"shared_email": email,
	}
	h.setSession(chatID, user.ID, "shared_contact", data)

	info := "📇 <b>Paylaşılan Kişi</b>\n👤 " + name
	if phone != "" { info += "\n📞 " + phone }
	if email != "" { info += "\n✉️ " + email }
	info += "\n\nNe yapmak istersiniz?"

	kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
		{{Text: "🏠 İlan Ekle", CallbackData: "sc_listing"}, {Text: "🎯 Talep Ekle", CallbackData: "sc_request"}},
		{{Text: "📞 İlgi Ekle", CallbackData: "sc_interest"}, {Text: "✅ Görev Ekle", CallbackData: "sc_task"}},
		{{Text: "👤 Müşteri Olarak Kaydet", CallbackData: "sc_customer"}},
		{{Text: "❌ İptal", CallbackData: "sc_cancel"}},
	}}
	h.tg.SendMessage(chatID, info, kb)
}

// extractVCardEmail — vCard metninden ilk e-postayı çıkarır
func extractVCardEmail(vcard string) string {
	if vcard == "" { return "" }
	for _, line := range strings.Split(vcard, "\n") {
		u := strings.ToUpper(line)
		if strings.HasPrefix(u, "EMAIL") {
			if idx := strings.LastIndex(line, ":"); idx >= 0 && idx+1 < len(line) {
				return strings.TrimSpace(line[idx+1:])
			}
		}
	}
	return ""
}

// sendPairMenu — çift-etiketli ana butona basınca chat-içi iki seçenek gösterir
func (h *BotHandler) sendPairMenu(chatID int64, title, lbl1, cb1, lbl2, cb2 string) {
	kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
		{{Text: lbl1, CallbackData: cb1}, {Text: lbl2, CallbackData: cb2}},
	}}
	h.tg.SendMessage(chatID, title, kb)
}

// sendCustomerMenu — Müşteriler ana menüsü
func (h *BotHandler) sendCustomerMenu(chatID int64, user *model.User) {
	customers, _ := h.customerRepo.List(user.ID, false, "")
	kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
		{{Text: "➕ Yeni Müşteri", CallbackData: "cust_add"}},
		{{Text: "📋 Müşterilerim", CallbackData: "cust_list"}},
		{{Text: "🔍 Müşteri Ara", CallbackData: "cust_search"}},
	}}
	h.tg.SendMessage(chatID,
		fmt.Sprintf("👥 <b>Müşteriler</b>\nToplam %d müşteriniz var.", len(customers)), kb)
}

// sendCustomerList — müşterileri sil butonlarıyla listeler
func (h *BotHandler) sendCustomerList(chatID int64, user *model.User, search string) {
	customers, _ := h.customerRepo.List(user.ID, false, search)
	if len(customers) == 0 {
		if search != "" {
			h.tg.SendMessage(chatID, "📭 Eşleşen müşteri bulunamadı.", nil)
		} else {
			h.tg.SendMessage(chatID, "📭 Henüz müşteriniz yok.", nil)
		}
		return
	}
	limit := len(customers)
	if limit > 20 { limit = 20 }
	h.tg.SendMessage(chatID, fmt.Sprintf("👥 <b>Müşterileriniz</b> (%d):", len(customers)), nil)
	for i := 0; i < limit; i++ {
		cu := customers[i]
		info := "👤 <b>" + cu.Name + "</b>"
		if cu.Phone != "" { info += "\n📞 " + cu.Phone }
		if cu.Email != "" { info += "\n✉️ " + cu.Email }
		kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
			{{Text: "🗑 Sil", CallbackData: fmt.Sprintf("cust_del_%d", cu.ID)}},
		}}
		h.tg.SendMessage(chatID, info, kb)
	}
	if len(customers) > 20 {
		h.tg.SendMessage(chatID, fmt.Sprintf("… ve %d müşteri daha. Aramayı kullanın.", len(customers)-20), nil)
	}
}

// startMineFilter — İlanlarım drill-down filtresini başlatır
func (h *BotHandler) startMineFilter(chatID int64, user *model.User) {
	h.setSession(chatID, user.ID, "mine_filter", map[string]string{})
	kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
		{{Text: "🌐 Tüm İlanlar", CallbackData: "mf_scope_all"}},
		{{Text: "👤 Benim İlanlarım", CallbackData: "mf_scope_mine"}},
		{{Text: "❌ İptal", CallbackData: "wizard_cancel"}},
	}}
	h.tg.SendMessage(chatID, "📋 <b>İlanlar</b>\nHangi ilanları görmek istersiniz?", kb)
}

// mineFilterSet — session'daki filtreye bir alan ekler
func (h *BotHandler) mineFilterSet(chatID int64, user *model.User, key, val string) {
	sess := h.getSession(chatID)
	data := map[string]string{}
	if sess != nil && sess.Data != nil {
		data = sess.Data
	}
	data[key] = val
	h.setSession(chatID, user.ID, "mine_filter", data)
}

// sendMineFilterResults — biriken filtreyle ilanları listeler
func (h *BotHandler) sendMineFilterResults(chatID int64, user *model.User) {
	sess := h.getSession(chatID)
	data := map[string]string{}
	if sess != nil && sess.Data != nil {
		data = sess.Data
	}
	onlyMine := data["scope"] != "all" // varsayılan: benim ilanlarım
	f := repository.ListFilter{
		UserID:       user.ID,
		OnlyMine:     onlyMine,
		IsAdmin:      user.Role == model.RoleAdmin,
		ListingType:  data["listing_type"],
		PropertyType: data["property_type"],
		District:     data["district"],
		Neighborhood: data["neighborhood"],
		Rooms:        data["rooms"],
		Zoning:       data["zoning"],
	}
	h.clearSession(chatID)
	h.sendListingsFiltered(chatID, user, f)
}

// showListingStepPrompt — ilan wizard'\''ında verilen adımı doğru klavyeyle gösterir
func (h *BotHandler) showListingStepPrompt(chatID int64, session *BotSession, idx int) {
	propType := session.Data["property_type"]
	steps := h.listingSteps(propType)
	if idx >= len(steps) {
		h.askListingNotify(chatID)
		return
	}
	step := steps[idx]
	var kb interface{}
	if step.Keyboard != nil {
		if step.Key == "neighborhood" {
			hoods := h.cfg.NeighborhoodsFor(session.Data["district"])
			kb = svc.NeighborhoodKeyboard(hoods, "wiz_hood")
		} else if step.Key == "_photos" {
			kb = step.Keyboard()
			h.tg.SendMessage(chatID, step.Prompt, kb)
			return
		} else {
			kb = step.Keyboard()
		}
	}
	if kb != nil {
		h.tg.SendMessage(chatID, step.Prompt, kb)
		return
	}
	// Müşteri/iletişim adımı → Yeni/Mevcut seçimi
	if step.Key == "contact" {
		// Contact önceden doldurulmuşsa (rehberden paylaşım) bu adımı atla
		if strings.TrimSpace(session.Data["contact"]) != "" {
			h.advanceListingPastContact(chatID, session)
			return
		}
		h.showListingCustomerChoice(chatID, session)
		return
	}
	skipKeys := map[string]bool{"contact": true, "neighborhood": true, "area_m2": true, "description": true}
	if skipKeys[step.Key] {
		h.tg.SendMessage(chatID, step.Prompt, svc.SkipKeyboard(step.Key))
	} else {
		combined := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{{{Text: "❌ İptal", CallbackData: "wizard_cancel"}}}}
		h.tg.SendMessage(chatID, step.Prompt, combined)
	}
}

// showListingCustomerChoice — Yeni/Mevcut müşteri seçimi
func (h *BotHandler) showListingCustomerChoice(chatID int64, session *BotSession) {
	session.Data["_lcust_search"] = ""
	h.saveSession(session)
	kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
		{{Text: "➕ Yeni Müşteri", CallbackData: "lwiz_cust_new"}},
		{{Text: "🔍 Mevcut Müşteriler", CallbackData: "lwiz_cust_pick"}},
		{{Text: "⏭️ Atla", CallbackData: "lwiz_cust_skip"}},
		{{Text: "❌ İptal", CallbackData: "wizard_cancel"}},
	}}
	h.tg.SendMessage(chatID, "👤 <b>Müşteri / Malik</b>\nYeni müşteri mi ekleyeceksiniz, mevcut müşterilerden mi seçeceksiniz?", kb)
}

// advanceListingPastContact — contact adımından sonrakine geçer
func (h *BotHandler) advanceListingPastContact(chatID int64, session *BotSession) {
	propType := session.Data["property_type"]
	steps := h.listingSteps(propType)
	ci := -1
	for i, s := range steps {
		if s.Key == "contact" { ci = i; break }
	}
	nextIdx := ci + 1
	session.Data["_step_idx"] = strconv.Itoa(nextIdx)
	session.Data["_lcust_search"] = ""
	h.saveSession(session)
	h.showListingStepPrompt(chatID, session, nextIdx)
}

// listingCustomerPicker — mevcut müşteri buton listesi (ilan wizard)
func (h *BotHandler) listingCustomerPicker(customers []model.Customer) *svc.TGInlineKeyboard {
	var rows [][]svc.TGInlineButton
	for _, c := range customers {
		label := c.Name
		if c.Phone != "" { label += " · " + c.Phone }
		rows = append(rows, []svc.TGInlineButton{{Text: label, CallbackData: "lwiz_cust_" + label}})
	}
	rows = append(rows, []svc.TGInlineButton{{Text: "❌ İptal", CallbackData: "wizard_cancel"}})
	return &svc.TGInlineKeyboard{InlineKeyboard: rows}
}

// listingCustomerSearchKb — arama sonuçları + yeni ekle (ilan wizard)
func (h *BotHandler) listingCustomerSearchKb(customers []model.Customer, typed string) *svc.TGInlineKeyboard {
	var rows [][]svc.TGInlineButton
	for _, c := range customers {
		label := c.Name
		if c.Phone != "" { label += " · " + c.Phone }
		rows = append(rows, []svc.TGInlineButton{{Text: label, CallbackData: "lwiz_cust_" + label}})
	}
	rows = append(rows,
		[]svc.TGInlineButton{{Text: "➕ Yeni: " + typed, CallbackData: "lwiz_custnew"}},
		[]svc.TGInlineButton{{Text: "❌ İptal", CallbackData: "wizard_cancel"}},
	)
	return &svc.TGInlineKeyboard{InlineKeyboard: rows}
}

// parseContact — "Ahmet Yılmaz 0532 123 45 67" → ad + telefon
func parseContact(s string) (name, phone string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	firstDigit := -1
	for i, ch := range s {
		if ch >= '0' && ch <= '9' {
			firstDigit = i
			break
		}
	}
	if firstDigit == -1 {
		return s, "" // sadece isim
	}
	name = strings.TrimSpace(s[:firstDigit])
	var d strings.Builder
	for _, ch := range s[firstDigit:] {
		if ch >= '0' && ch <= '9' {
			d.WriteRune(ch)
		}
	}
	phone = d.String()
	if name == "" {
		name = s
	}
	return name, phone
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
	switch session.Step {
	case "listing_wizard":
		h.listingWizardTextStep(msg, user, session)
	case "request_wizard":
		h.requestWizardTextStep(msg, user, session)
	case "cust_add":
		h.customerAddTextStep(msg, user, session)
	case "interest_wizard":
		h.interestWizardTextStep(msg, user, session)
	case "cust_search":
		h.clearSession(msg.Chat.ID)
		h.sendCustomerList(msg.Chat.ID, user, strings.TrimSpace(msg.Text))
	default:
		// Bilinmeyen/eski step → temizle ve ana menüyü göster
		h.clearSession(msg.Chat.ID)
		h.sendMainMenu(msg.Chat.ID, "Ana Menü:")
	}
}

// customerAddTextStep — Müşteriler menüsünden yeni müşteri ekleme (isim → telefon)
func (h *BotHandler) customerAddTextStep(msg *svc.TGMessage, user *model.User, session *BotSession) {
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)
	step := session.Data["_cstep"]
	if step == "name" {
		if text == "" || text == "-" {
			h.tg.SendMessage(chatID, "👤 Müşteri adını yazın:", nil)
			return
		}
		session.Data["_cname"] = text
		session.Data["_cstep"] = "phone"
		h.saveSession(session)
		h.tg.SendMessage(chatID, "📞 Telefon numarasını yazın (yoksa - yazın):", nil)
		return
	}
	if step == "phone" {
		phone := text
		if phone == "-" { phone = "" }
		name := session.Data["_cname"]
		h.clearSession(chatID)
		// Mükerrer kontrolü
		if dup, _ := h.customerRepo.FindDuplicate(user.ID, name, phone); dup != nil {
			h.tg.SendMessage(chatID, fmt.Sprintf("ℹ️ <b>%s</b> zaten kayıtlı. Yeni kayıt oluşturulmadı.", dup.Name), nil)
			return
		}
		newC := &model.Customer{UserID: user.ID, Name: name, Phone: phone}
		if err := h.customerRepo.Create(newC); err != nil {
			h.tg.SendMessage(chatID, "❌ Müşteri eklenemedi: "+err.Error(), nil)
		} else {
			h.tg.SendMessage(chatID, fmt.Sprintf("✅ <b>%s</b> eklendi.", name), nil)
		}
		return
	}
	h.clearSession(chatID)
	h.sendMainMenu(chatID, "Ana Menü:")
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
		h.askListingNotify(chatID)
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

	// Müşteri/iletişim adımı
	if currentStep.Key == "contact" {
		text := strings.TrimSpace(msg.Text)
		// Yeni müşteri: önce isim
		if session.Data["_lcust_new"] == "name" {
			if text == "" || text == "-" {
				h.tg.SendMessage(chatID, "👤 Müşteri adını yazın:", nil)
				return
			}
			session.Data["_lcust_name"] = text
			session.Data["_lcust_new"] = "phone"
			h.saveSession(session)
			h.tg.SendMessage(chatID, "📞 Telefon numarasını yazın (yoksa - yazın):", nil)
			return
		}
		// Yeni müşteri: sonra telefon → contact'a "Ad Tel" yaz ve ilerle
		if session.Data["_lcust_new"] == "phone" {
			phone := text
			if phone == "-" { phone = "" }
			name := session.Data["_lcust_name"]
			// Mükerrer kontrolü — varsa mevcut müşteriye bağla
			if dup, _ := h.customerRepo.FindDuplicate(session.UserID, name, phone); dup != nil {
				name = dup.Name
				phone = dup.Phone
				h.tg.SendMessage(chatID, fmt.Sprintf("ℹ️ <b>%s</b> zaten kayıtlı, mevcut müşteriye bağlandı.", dup.Name), nil)
			}
			contact := name
			if phone != "" { contact = name + " " + phone }
			session.Data["contact"] = contact
			session.Data["_lcust_new"] = ""
			h.advanceListingPastContact(chatID, session)
			return
		}
		// Arama modu: yazılanı ara, sonuçları buton göster
		if session.Data["_lcust_search"] == "1" {
			if text == "" || text == "-" {
				h.advanceListingPastContact(chatID, session)
				return
			}
			results, _ := h.customerRepo.List(session.UserID, false, text)
			session.Data["_lcust_typed"] = text
			h.saveSession(session)
			if len(results) > 0 {
				if len(results) > 8 { results = results[:8] }
				h.tg.SendMessage(chatID,
					fmt.Sprintf("🔍 \"%s\" için %d sonuç. Seçin veya yeni ekleyin:", text, len(results)),
					h.listingCustomerSearchKb(results, text))
			} else {
				kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
					{{Text: "➕ Yeni: " + text, CallbackData: "lwiz_custnew"}},
					{{Text: "❌ İptal", CallbackData: "wizard_cancel"}},
				}}
				h.tg.SendMessage(chatID,
					fmt.Sprintf("🔍 \"%s\" için eşleşme yok. Yeni ekleyebilir veya tekrar arayabilirsiniz.", text), kb)
			}
			return
		}
		// Yeni müşteri yazımı (serbest "ad telefon")
		session.Data["contact"] = text
		h.advanceListingPastContact(chatID, session)
		return
	}

	val := strings.TrimSpace(msg.Text)
	if val == "-" { val = "" }
	session.Data[currentStep.Key] = val

	nextIdx := idx + 1
	session.Data["_step_idx"] = strconv.Itoa(nextIdx)
	h.saveSession(session)

	if nextIdx >= len(steps) {
		h.askListingNotify(chatID)
		return
	}

	h.showListingStepPrompt(chatID, session, nextIdx)
}

func (h *BotHandler) listingWizardCallbackStep(chatID int64, user *model.User, session *BotSession, cbData string) {
	propType := session.Data["property_type"]
	steps    := h.listingSteps(propType)
	idx, _   := strconv.Atoi(session.Data["_step_idx"])
	if idx >= len(steps) {
		h.askListingNotify(chatID)
		return
	}
	currentStep := steps[idx]

	// _photos adımındayken sadece wiz_photos_done beklenir — finalize handleCallback'te yapılır
	if currentStep.Key == "_photos" {
		return
	}

	// Müşteri/iletişim adımı — Yeni/Mevcut/seçim
	if currentStep.Key == "contact" {
		switch {
		case cbData == "lwiz_cust_new":
			session.Data["_lcust_search"] = ""
			session.Data["_lcust_new"] = "name"
			h.saveSession(session)
			h.tg.SendMessage(chatID, "➕ <b>Yeni Müşteri</b>\nMüşteri adını yazın:", nil)
			return
		case cbData == "lwiz_cust_skip":
			session.Data["contact"] = ""
			h.advanceListingPastContact(chatID, session)
			return
		case cbData == "lwiz_cust_pick":
			customers, _ := h.customerRepo.List(session.UserID, false, "")
			if len(customers) == 0 {
				session.Data["_lcust_search"] = ""
				h.saveSession(session)
				h.tg.SendMessage(chatID, "📭 Kayıtlı müşteriniz yok. Ad ve telefon yazın:", nil)
			} else if len(customers) < 10 {
				session.Data["_lcust_search"] = ""
				h.saveSession(session)
				h.tg.SendMessage(chatID, "👤 Müşteri seçin:", h.listingCustomerPicker(customers))
			} else {
				session.Data["_lcust_search"] = "1"
				h.saveSession(session)
				h.tg.SendMessage(chatID,
					fmt.Sprintf("🔍 <b>Müşteri Ara</b>\n%d müşteriniz var. Ad veya telefonun ilk hanelerini yazın (örn: <i>ahm</i> veya <i>0532</i>).", len(customers)), nil)
			}
			return
		case cbData == "lwiz_custnew":
			session.Data["_lcust_name"] = session.Data["_lcust_typed"]
			session.Data["_lcust_search"] = ""
			session.Data["_lcust_new"] = "phone"
			h.saveSession(session)
			h.tg.SendMessage(chatID, "📞 Telefon numarasını yazın (yoksa - yazın):", nil)
			return
		case strings.HasPrefix(cbData, "lwiz_cust_"):
			label := strings.TrimPrefix(cbData, "lwiz_cust_")
			session.Data["contact"] = strings.ReplaceAll(label, " · ", " ")
			h.advanceListingPastContact(chatID, session)
			return
		}
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
		h.askListingNotify(chatID)
		return
	}
	h.showListingStepPrompt(chatID, session, nextIdx)
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

// askListingNotify — finalize öncesi "herkese bildir?" sorusu gösterir
func (h *BotHandler) askListingNotify(chatID int64) {
	kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
		{{Text: "🔔 Evet, herkese bildir", CallbackData: "wiz_notify_yes"}},
		{{Text: "🔕 Hayır, bildirme", CallbackData: "wiz_notify_no"}},
	}}
	h.tg.SendMessage(chatID, "Bu ilanı tüm danışmanlara bildireyim mi?\n\n<i>Hayır deseniz de eşleşen aktif talep sahiplerine bilgi verilir.</i>", kb)
}

func (h *BotHandler) finalizeListing(chatID int64, user *model.User, data map[string]string, notifyAll bool) {
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
		"_source":       "telegram",
	}

	// İletişim bilgisinden müşteri (malik) oluştur/bul ve ilana bağla
	var custID int64
	cName, cPhone := parseContact(data["contact"])
	if cName != "" {
		existing, _ := h.customerRepo.List(user.ID, false, cName)
		for _, cu := range existing {
			if strings.EqualFold(cu.Name, cName) {
				custID = cu.ID
				break
			}
		}
		if custID == 0 {
			newC := &model.Customer{UserID: user.ID, Name: cName, Phone: cPhone}
			if err := h.customerRepo.Create(newC); err == nil {
				custID = newC.ID
			}
		}
	}

	listing := &model.Listing{
		UserID:     user.ID,
		Fields:     fields,
		IsActive:   true,
		CustomerID: custID,
	}
	if err := h.listingRepo.Create(listing); err != nil {
		log.Printf("[BOT][HATA] ilan oluşturma: %v | user=%d fields=%v", err, user.ID, fields)
		h.clearSession(chatID)
		h.tg.SendMessage(chatID, fmt.Sprintf("❌ İlan kaydedilemedi: %v\nLütfen tekrar deneyin.", err), svc.MainMenuKeyboard())
		return
	}
	log.Printf("[BOT] ilan oluşturuldu: id=%d no=%d user=%d", listing.ID, listing.ListingNo, user.ID)

	// Bildirim gönder (notifyAll'a göre genel + her durumda eşleşme)
	if h.notifySvc != nil {
		go h.sendBotListingNotification(listing, user, notifyAll)
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
		// Yeni müşteri: önce isim
		if session.Data["_cust_new"] == "name" {
			if text == "" || text == "-" {
				h.tg.SendMessage(chatID, "👤 Müşteri adını yazın:", nil)
				return
			}
			session.Data["client_name"] = text
			session.Data["_cust_new"] = "phone"
			h.saveSession(session)
			h.tg.SendMessage(chatID, "📞 Telefon numarasını yazın (yoksa - yazın):", nil)
			return
		}
		// Yeni müşteri: sonra telefon → kaydet ve ilerle
		if session.Data["_cust_new"] == "phone" {
			phone := text
			if phone == "-" { phone = "" }
			// Mükerrer kontrolü — varsa mevcut müşteriye bağla
			if dup, _ := h.customerRepo.FindDuplicate(session.UserID, session.Data["client_name"], phone); dup != nil {
				session.Data["client_name"] = dup.Name
				session.Data["phone"] = dup.Phone
				h.tg.SendMessage(chatID, fmt.Sprintf("ℹ️ <b>%s</b> zaten kayıtlı, mevcut müşteriye bağlandı.", dup.Name), nil)
			} else {
				session.Data["phone"] = phone
			}
			session.Data["_cust_new"] = ""
			nextIdx := idx + 1
			session.Data["_step_idx"] = strconv.Itoa(nextIdx)
			h.saveSession(session)
			h.sendRequestStep(chatID, session, nextIdx, session.UserID)
			return
		}
		if text == "" || text == "-" {
			h.tg.SendMessage(chatID, "👤 Müşteri adını yazın:", nil)
			return
		}
		// Arama modu (çok müşteri varken): yazılanı ara, sonuçları buton göster
		if session.Data["_cust_search"] == "1" {
			results, _ := h.customerRepo.List(session.UserID, false, text)
			session.Data["_cust_typed"] = text
			h.saveSession(session)
			if len(results) > 0 {
				if len(results) > 8 { results = results[:8] }
				h.tg.SendMessage(chatID,
					fmt.Sprintf("🔍 \"%s\" için %d sonuç. Seçin veya yeni ekleyin:", text, len(results)),
					h.customerSearchKeyboard(results, text))
			} else {
				kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
					{{Text: "➕ Yeni: " + text, CallbackData: "rwiz_custnew"}},
					{{Text: "❌ İptal", CallbackData: "wizard_cancel"}},
				}}
				h.tg.SendMessage(chatID,
					fmt.Sprintf("🔍 \"%s\" için eşleşme yok. Yeni müşteri olarak ekleyebilir veya tekrar arayabilirsiniz.", text), kb)
			}
			return
		}
		// Yeni müşteri: "Ad Soyad 0532..." → isim + telefon ayır
		cn, cp := parseContact(text)
		session.Data["client_name"] = cn
		session.Data["phone"] = cp
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
		// Yeni müşteri ekle → ad yaz
		if cbData == "rwiz_cust_new" {
			session.Data["_cust_search"] = ""
			session.Data["_cust_new"] = "name"
			h.saveSession(session)
			h.tg.SendMessage(chatID, "➕ <b>Yeni Müşteri</b>\nMüşteri adını yazın:", nil)
			return
		}
		// Mevcut müşterilerden seç → <10 liste, ≥10 arama
		if cbData == "rwiz_cust_pick" {
			customers, _ := h.customerRepo.List(session.UserID, false, "")
			if len(customers) == 0 {
				session.Data["_cust_search"] = ""
				h.saveSession(session)
				h.tg.SendMessage(chatID, "📭 Kayıtlı müşteriniz yok. Müşteri adını yazın:", nil)
			} else if len(customers) < 10 {
				session.Data["_cust_search"] = ""
				h.saveSession(session)
				h.tg.SendMessage(chatID, "👤 Müşteri seçin:", h.customerPickerKeyboard(customers))
			} else {
				session.Data["_cust_search"] = "1"
				h.saveSession(session)
				h.tg.SendMessage(chatID,
					fmt.Sprintf("🔍 <b>Müşteri Ara</b>\n%d müşteriniz var. Adın veya telefonun ilk birkaç hanesini yazın (örn: <i>ahm</i> veya <i>0532</i>).", len(customers)),
					nil)
			}
			return
		}
		if cbData == "rwiz_cust_elle" {
			session.Data["_cust_search"] = ""
			h.saveSession(session)
			h.tg.SendMessage(chatID, "👤 Müşteri adını yazın:", nil)
			return
		}
		// Aramada bulunamadı → yazılan adla yeni müşteri, telefon sor
		if cbData == "rwiz_custnew" {
			session.Data["client_name"] = session.Data["_cust_typed"]
			session.Data["_cust_search"] = ""
			session.Data["_cust_new"] = "phone"
			h.saveSession(session)
			h.tg.SendMessage(chatID, "📞 Telefon numarasını yazın (yoksa - yazın):", nil)
			return
		}
		// "Ad Soyad · Tel" ayrıştır (listeden/aramadan seçim)
		parts := strings.SplitN(val, " · ", 2)
		session.Data["client_name"] = strings.TrimSpace(parts[0])
		if len(parts) == 2 {
			session.Data["phone"] = strings.TrimSpace(parts[1])
		}
		session.Data["_cust_search"] = ""
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
		// Önce: Yeni mi ekle, mevcuttan mı seç?
		session.Data["_cust_search"] = ""
		h.saveSession(session)
		kb := &svc.TGInlineKeyboard{InlineKeyboard: [][]svc.TGInlineButton{
			{{Text: "➕ Yeni Müşteri", CallbackData: "rwiz_cust_new"}},
			{{Text: "🔍 Mevcut Müşteriler", CallbackData: "rwiz_cust_pick"}},
			{{Text: "❌ İptal", CallbackData: "wizard_cancel"}},
		}}
		h.tg.SendMessage(chatID, "👤 <b>Müşteri</b>\nYeni müşteri mi eklemek istersiniz, mevcut müşterilerden mi seçeceksiniz?", kb)
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

func (h *BotHandler) sendBotListingNotification(listing *model.Listing, owner *model.User, notifyAll bool) {
	usersWithChat, err := h.userRepo.ListWithChatIDs()
	if err != nil {
		log.Printf("[bot-notify] ListWithChatIDs: %v", err)
		return
	}
	var allUsers []svc.UserForNotify
	chatByUserID := map[int64]int64{}
	for _, u := range usersWithChat {
		chatID, _ := strconv.ParseInt(u.TelegramChatID, 10, 64)
		if chatID == 0 { continue }
		chatByUserID[u.ID] = chatID
		allUsers = append(allUsers, svc.UserForNotify{
			ID:         u.ID,
			ChatID:     chatID,
			NotifyType: "all",
		})
	}
	// Eşleşme bildirimi: yalnızca aktif + notify_me açık talepler
	reqs, _ := h.requestRepo.List(repository.RequestFilter{OnlyActive: true})
	var requests []svc.RequestForMatch
	for _, req := range reqs {
		if !req.NotifyMe { continue }
		requests = append(requests, svc.RequestForMatch{
			ID:         req.ID,
			UserID:     req.UserID,
			UserChatID: chatByUserID[req.UserID],
			NotifyMe:   req.NotifyMe,
			Fields:     req.Fields,
		})
	}
	lm := svc.ListingForMatch{
		ID:          listing.ID,
		ListingNo:   listing.ListingNo,
		UserID:      listing.UserID,
		OwnerID:     listing.UserID,
		OwnerName:   owner.FullName,
		OwnerChatID: chatByUserID[listing.UserID],
		IsActive:    listing.IsActive,
		Fields:      listing.Fields,
	}
	h.notifySvc.NotifyNewListing(lm, allUsers, requests, notifyAll)
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

// customerSearchKeyboard — arama sonuçları + "yeni ekle" butonu
func (h *BotHandler) customerSearchKeyboard(customers []model.Customer, typedName string) *svc.TGInlineKeyboard {
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
		[]svc.TGInlineButton{{Text: "➕ Yeni: " + typedName, CallbackData: "rwiz_custnew"}},
		[]svc.TGInlineButton{{Text: "❌ İptal", CallbackData: "wizard_cancel"}},
	)
	return &svc.TGInlineKeyboard{InlineKeyboard: rows}
}

func (h *BotHandler) sendMainMenu(chatID int64, intro string) {
	// Kalıcı alt menü (reply keyboard) — ana navigasyon hep görünür
	h.tg.SendMessage(chatID, intro, svc.MainReplyKeyboard())
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
