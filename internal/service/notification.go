package service

import (
	"fmt"
	"log"
	"strconv"
)

type NotificationService struct {
	tg *TelegramService
}

func NewNotificationService(tg *TelegramService) *NotificationService {
	return &NotificationService{tg: tg}
}

func (n *NotificationService) NotifyNewListing(
	listing ListingForMatch,
	allUsers []UserForNotify,
	requests []RequestForMatch,
	notifyAll bool,
) {
	price := listing.Fields["price"]
	if price == "" {
		price = listing.Fields["price_max"]
	}
	priceN, _ := strconv.ParseInt(price, 10, 64)

	ilanText := fmt.Sprintf(
		"🏠 <b>Yeni İlan!</b>\n\n"+
			"👤 %s yeni ilan ekledi\n"+
			"📋 #%d — %s\n"+
			"🏘️ %s / %s\n"+
			"📐 %s m² %s\n"+
			"💰 %s ₺",
		listing.OwnerName,
		listing.ListingNo,
		listing.Fields["title"],
		listing.Fields["district"],
		listing.Fields["neighborhood"],
		listing.Fields["area_m2"],
		listing.Fields["property_type"],
		FormatPrice(priceN),
	)

	ownerText := fmt.Sprintf(
		"✅ <b>İlanınız Eklendi!</b>\n\n"+
			"📋 #%d — %s\n"+
			"📍 %s / %s\n"+
			"💰 %s ₺",
		listing.ListingNo,
		listing.Fields["title"],
		listing.Fields["district"],
		listing.Fields["neighborhood"],
		FormatPrice(priceN),
	)

	// Eşleşmeleri hesapla
	matches := []RequestForMatch{}
	for _, req := range requests {
		score := MatchScore(req.Fields, listing.Fields)
		if score >= 60 {
			req.Score = score
			matches = append(matches, req)
		}
	}

	// ADIM 1: Genel bildirim — yalnızca "herkese bildir" seçiliyse
	for _, u := range allUsers {
		if u.ChatID == 0 {
			continue
		}
		if u.NotifyType == "none" {
			continue
		}

		// İlan sahibine her zaman "ilanınız eklendi" mesajı
		if u.ID == listing.OwnerID {
			if err := n.tg.SendNotification(u.ChatID, ownerText); err != nil {
				log.Printf("İlan sahibi bildirimi hatası: %v", err)
			}
			continue
		}

		// Herkese bildir seçili değilse genel bildirim gönderme
		if !notifyAll {
			continue
		}

		// match tipindeyse sadece eşleşme varsa gönder
		if u.NotifyType == "match" {
			hasMatch := false
			for _, m := range matches {
				if m.UserID == u.ID {
					hasMatch = true
					break
				}
			}
			if !hasMatch {
				continue
			}
		}

		log.Printf("İLAN BİLDİRİMİ: user=%d chatID=%d", u.ID, u.ChatID)
		if err := n.tg.SendNotification(u.ChatID, ilanText); err != nil {
			log.Printf("Bildirim hatası (user=%d): %v", u.ID, err)
		} else {
			log.Printf("Bildirim gönderildi: user=%d", u.ID)
		}
	}

	// ADIM 2: Eşleşen talep sahiplerine
	notifiedOwner := false
	for _, req := range matches {
		if req.UserChatID == 0 {
			continue
		}
		// Talep kriterleri özeti
		reqType := req.Fields["listing_type"]
		if reqType == "" { reqType = "-" }
		reqProp := req.Fields["property_type"]
		if reqProp == "" { reqProp = "-" }
		reqLoc := req.Fields["district"]
		if req.Fields["neighborhood"] != "" {
			reqLoc = reqLoc + " / " + req.Fields["neighborhood"]
		}
		if reqLoc == "" { reqLoc = "-" }

		// İlan tipi + mülk tipi
		ilanType := listing.Fields["listing_type"]
		if ilanType == "" { ilanType = "-" }
		ilanProp := listing.Fields["property_type"]
		if ilanProp == "" { ilanProp = "-" }

		matchText := fmt.Sprintf(
			"🎯 <b>Talebinize Uygun İlan Bulundu!</b>  (%%%d uyum)\n\n"+
				"<b>📋 TALEP</b>\n"+
				"👤 Müşteri: %s\n"+
				"🔖 %s · %s\n"+
				"📍 %s\n\n"+
				"<b>🏠 EŞLEŞEN İLAN</b>\n"+
				"%s (#%d)\n"+
				"🔖 %s · %s\n"+
				"📍 %s / %s\n"+
				"💰 %s ₺",
			req.Score,
			req.Fields["client_name"],
			reqType, reqProp,
			reqLoc,
			listing.Fields["title"],
			listing.ListingNo,
			ilanType, ilanProp,
			listing.Fields["district"],
			listing.Fields["neighborhood"],
			FormatPrice(priceN),
		)
		log.Printf("EŞLEŞME BİLDİRİMİ: chatID=%d", req.UserChatID)
		if err := n.tg.SendNotification(req.UserChatID, matchText); err != nil {
			log.Printf("Eşleşme bildirimi hatası: %v", err)
		}

		// İlan sahibine eşleşme bildirimi (bir kez)
		if !notifiedOwner && listing.OwnerChatID != 0 {
			ownerMatchText := fmt.Sprintf(
				"🎯 <b>İlanınızla Eşleşen Talep Var!</b>  (%%%d uyum)\n\n"+
					"<b>🏠 İLAN</b>\n"+
					"%s (#%d)\n"+
					"🔖 %s · %s\n\n"+
					"<b>📋 EŞLEŞEN TALEP</b>\n"+
					"👤 Müşteri: %s\n"+
					"🔖 %s · %s\n"+
					"📍 %s",
				req.Score,
				listing.Fields["title"],
				listing.ListingNo,
				ilanType, ilanProp,
				req.Fields["client_name"],
				reqType, reqProp,
				reqLoc,
			)
			n.tg.SendNotification(listing.OwnerChatID, ownerMatchText)
			notifiedOwner = true
		}
	}
}

// ─── Tipler ───────────────────────────────────────────────────

type UserForNotify struct {
	ID         int64
	ChatID     int64
	NotifyType string
}

type RequestForMatch struct {
	ID         int64
	UserID     int64
	UserChatID int64
	NotifyMe   bool
	Score      int
	Fields     map[string]string
}

type ListingForMatch struct {
	ID          int64
	ListingNo   int64
	UserID      int64
	OwnerID     int64
	OwnerName   string
	OwnerChatID int64
	IsActive    bool
	Fields      map[string]string
}

func MatchScore(talepFields, ilanFields map[string]string) int {
	if ilanFields == nil || talepFields == nil {
		return 0
	}
	tLT := talepFields["listing_type"]
	iLT := ilanFields["listing_type"]
	if tLT != "" && iLT != "" && tLT != iLT {
		return 0
	}
	tPT := talepFields["property_type"]
	iPT := ilanFields["property_type"]
	if tPT != "" && iPT != "" && tPT != iPT {
		return 0
	}
	score, total := 0, 0
	check := func(tVal, iVal string, weight int) {
		total += weight
		if tVal == "" {
			score += weight
		} else if tVal == iVal {
			score += weight
		}
	}
	check(tLT, iLT, 25)
	check(tPT, iPT, 20)
	check(talepFields["district"], ilanFields["district"], 15)
	check(talepFields["neighborhood"], ilanFields["neighborhood"], 10)
	total += 20
	budgetMax := parseInt64(talepFields["budget_max"])
	if budgetMax == 0 {
		budgetMax = parseInt64(talepFields["budget"])
	}
	budgetMin := parseInt64(talepFields["budget_min"])
	price := parseInt64(ilanFields["price_max"])
	if price == 0 {
		price = parseInt64(ilanFields["price"])
	}
	if budgetMax == 0 {
		score += 20
	} else if budgetMin > 0 && price < budgetMin {
		return 0
	} else if price <= budgetMax {
		score += 20
	} else if price <= int64(float64(budgetMax)*1.1) {
		score += 10
	}
	total += 10
	if talepFields["rooms"] == "" {
		score += 10
	} else if talepFields["rooms"] == ilanFields["rooms"] {
		score += 10
	}
	if total == 0 {
		return 0
	}
	return (score * 100) / total
}

func parseInt64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func FormatPrice(n int64) string {
	if n == 0 {
		return "—"
	}
	s := strconv.FormatInt(n, 10)
	result := []byte{}
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
