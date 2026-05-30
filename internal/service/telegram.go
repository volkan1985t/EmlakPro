package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/volkan1985t/EmlakPro/internal/config"
	"github.com/volkan1985t/EmlakPro/internal/model"
)

// ── Exported Telegram types ───────────────────────────────────

type TGUpdate struct {
	UpdateID      int64       `json:"update_id"`
	Message       *TGMessage  `json:"message"`
	CallbackQuery *TGCallback `json:"callback_query"`
}

type TGPhotoSize struct {
	FileID   string `json:"file_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FileSize int    `json:"file_size"`
}

type TGMessage struct {
	MessageID int64          `json:"message_id"`
	Chat      TGChat         `json:"chat"`
	Text      string         `json:"text"`
	Photo     []TGPhotoSize  `json:"photo"`  // en büyük boyut son eleman
}

type TGChat struct {
	ID int64 `json:"id"`
}

type TGCallback struct {
	ID      string     `json:"id"`
	Message *TGMessage `json:"message"`
	Data    string     `json:"data"`
}

type TGInlineKeyboard struct {
	InlineKeyboard [][]TGInlineButton `json:"inline_keyboard"`
}

type TGInlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

// ── TelegramService ───────────────────────────────────────────

type TelegramService struct {
	token         string
	enabled       bool
	userRepo      userRepo
	updateHandler func(TGUpdate)
}

type userRepo interface {
	ListWithChatIDs() ([]model.User, error)
}

type tgResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
}

func NewTelegramService(cfg *config.TelegramConfig, repo userRepo) *TelegramService {
	return &TelegramService{
		token:    cfg.BotToken,
		enabled:  cfg.Enabled && cfg.BotToken != "",
		userRepo: repo,
	}
}

// SetUpdateHandler registers the function called for every incoming update.
func (s *TelegramService) SetUpdateHandler(fn func(TGUpdate)) {
	s.updateHandler = fn
}

// ── Send methods ──────────────────────────────────────────────

// GetFileURL — Telegram file_id'den indirilebilir URL döndürür
func (s *TelegramService) GetFileURL(fileID string) (string, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", s.token, fileID)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.OK || result.Result.FilePath == "" {
		return "", fmt.Errorf("dosya bulunamadı")
	}
	return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", s.token, result.Result.FilePath), nil
}

// DownloadFile — URL'den dosya içeriğini indirir
func (s *TelegramService) DownloadFile(fileURL string) ([]byte, error) {
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *TelegramService) sendRaw(chatID int64, text string, replyMarkup interface{}) {
	if !s.enabled {
		return
	}
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	if replyMarkup != nil {
		payload["reply_markup"] = replyMarkup
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.token)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[telegram] sendMessage hatası: %v", err)
		return
	}
	resp.Body.Close()
}

// send is used internally when chatID is already a string (legacy helper).
func (s *TelegramService) send(chatIDStr, text string) {
	if !s.enabled || chatIDStr == "" {
		return
	}
	var n int64
	fmt.Sscanf(chatIDStr, "%d", &n)
	if n == 0 {
		return
	}
	s.sendRaw(n, text, nil)
}

// SendNotification sends a plain HTML message; satisfies NotificationService dependency.
func (s *TelegramService) SendNotification(chatID int64, text string) error {
	s.sendRaw(chatID, text, nil)
	return nil
}

// SendMessage sends a message with an optional inline keyboard.
func (s *TelegramService) SendMessage(chatID int64, text string, kb interface{}) {
	s.sendRaw(chatID, text, kb)
}

// AnswerCallback — callback query'ye cevap ver (spinner'ı temizler, zorunlu)
func (s *TelegramService) AnswerCallback(callbackID string) {
	if !s.enabled { return }
	payload := map[string]interface{}{"callback_query_id": callbackID}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", s.token)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil { log.Printf("[telegram] answerCallback hatası: %v", err); return }
	resp.Body.Close()
}

// RemoveKeyboard — gönderilen mesajdaki inline keyboard'u kaldır (async)
func (s *TelegramService) RemoveKeyboard(chatID int64, messageID int64) {
	if !s.enabled { return }
	go func() {
		payload := map[string]interface{}{
			"chat_id":      chatID,
			"message_id":   messageID,
			"reply_markup": map[string]interface{}{"inline_keyboard": []interface{}{}},
		}
		body, _ := json.Marshal(payload)
		url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageReplyMarkup", s.token)
		resp, err := http.Post(url, "application/json", bytes.NewReader(body))
		if err != nil { log.Printf("[telegram] removeKeyboard hatası: %v", err); return }
		resp.Body.Close()
	}()
}

// ── Task notifications ────────────────────────────────────────

func (s *TelegramService) NotifyAssigned(task *model.Task, assignees []model.TaskUser) {
	log.Printf("[notify] NotifyAssigned called, enabled=%v assignees=%d", s.enabled, len(assignees))
	if !s.enabled {
		return
	}
	users, err := s.userRepo.ListWithChatIDs()
	if err != nil {
	log.Printf("[notify] ListWithChatIDs users=%d err=%v", len(users), err)
		return
	}
	chatByID := map[int64]string{}
	for _, u := range users {
		chatByID[u.ID] = u.TelegramChatID
		log.Printf("[notify] user=%d chatID=%s", u.ID, u.TelegramChatID)
		log.Printf("[notify] user=%d chatID=%s", u.ID, u.TelegramChatID)
	}
	msg := fmt.Sprintf("📋 <b>Yeni Görev Atandı</b>\n\n<b>%s</b>\n\nPriorite: %s\nDurum: %s",
		task.Title, priorityLabel(task.Priority), statusLabel(task.Status))
	if task.DueDate != nil {
		msg += fmt.Sprintf("\nBitiş: %s", task.DueDate.Format("02.01.2006"))
	}
	for _, a := range assignees {
		if chatID, ok := chatByID[a.ID]; ok {
			s.send(chatID, msg)
		}
	}
}

func (s *TelegramService) NotifyStatusChanged(task *model.Task, oldStatus string) {
	if !s.enabled {
		return
	}
	users, err := s.userRepo.ListWithChatIDs()
	if err != nil {
		return
	}
	chatByID := map[int64]string{}
	for _, u := range users {
		chatByID[u.ID] = u.TelegramChatID
	}
	msg := fmt.Sprintf("🔄 <b>Görev Durumu Değişti</b>\n\n<b>%s</b>\n\n%s → %s",
		task.Title, statusLabel(oldStatus), statusLabel(task.Status))
	for _, a := range task.Assignees {
		if chatID, ok := chatByID[a.ID]; ok {
			s.send(chatID, msg)
		}
	}
}

// ── Polling ───────────────────────────────────────────────────

func (s *TelegramService) StartPolling() {
	if !s.enabled {
		return
	}
	go func() {
		var offset int64
		client := &http.Client{Timeout: 35 * time.Second}
		log.Println("[telegram] polling başlatıldı")
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[telegram] update handler panic: %v", r)
					}
				}()
				updates, err := s.getUpdates(client, offset)
				if err != nil {
					log.Printf("[telegram] getUpdates hatası: %v", err)
					time.Sleep(5 * time.Second)
					return
				}
				for _, u := range updates {
					offset = u.UpdateID + 1
					if s.updateHandler != nil {
						s.updateHandler(u)
					} else if u.Message != nil &&
						(u.Message.Text == "/chatid" || u.Message.Text == "/start") {
						reply := fmt.Sprintf("Chat ID'niz: <code>%d</code>\n\nBu numarayı yöneticiye bildirin.", u.Message.Chat.ID)
						s.sendRaw(u.Message.Chat.ID, reply, nil)
					}
				}
			}()
		}
	}()
}

func (s *TelegramService) getUpdates(client *http.Client, offset int64) ([]TGUpdate, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=30&offset=%d", s.token, offset)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tgResp struct {
		OK     bool       `json:"ok"`
		Result []TGUpdate `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
		return nil, err
	}
	return tgResp.Result, nil
}

// ── Keyboard helpers ──────────────────────────────────────────

var defaultPropertyTypes = []string{"Daire", "Villa", "Arsa", "Ticari", "Diğer"}
var defaultListingTypes  = []string{"Satılık", "Kiralık"}

func MainMenuKeyboard() *TGInlineKeyboard {
	return &TGInlineKeyboard{InlineKeyboard: [][]TGInlineButton{
		{{Text: "📋 İlanlar", CallbackData: "menu_list"}, {Text: "🏠 İlanlarım", CallbackData: "menu_mine"}},
		{{Text: "➕ İlan Gir", CallbackData: "menu_add_listing"}, {Text: "🎯 Talep Gir", CallbackData: "menu_add_request"}},
		{{Text: "📂 Taleplerim", CallbackData: "menu_my_requests"}, {Text: "✅ Görevler", CallbackData: "menu_tasks"}},
		{{Text: "🔔 Bildirimler", CallbackData: "menu_notify"}},
	}}
}

func PropertyTypeKeyboard(prefix string) *TGInlineKeyboard {
	return OptionsKeyboard(defaultPropertyTypes, prefix)
}

func ListingTypeKeyboard(prefix string) *TGInlineKeyboard {
	return OptionsKeyboard(defaultListingTypes, prefix)
}

func DistrictKeyboard(districts []string, prefix string) *TGInlineKeyboard {
	return OptionsKeyboard(districts, prefix)
}

func NeighborhoodKeyboard(hoods []string, prefix string) *TGInlineKeyboard {
	return OptionsKeyboard(hoods, prefix)
}

// OptionsKeyboard builds an inline keyboard from a list, 2 buttons per row.
func OptionsKeyboard(opts []string, prefix string) *TGInlineKeyboard {
	var rows [][]TGInlineButton
	var row []TGInlineButton
	for i, opt := range opts {
		row = append(row, TGInlineButton{
			Text:         opt,
			CallbackData: prefix + "_" + opt,
		})
		if len(row) == 2 || i == len(opts)-1 {
			rows = append(rows, row)
			row = nil
		}
	}
	rows = append(rows, []TGInlineButton{{Text: "❌ İptal", CallbackData: "wizard_cancel"}})
	return &TGInlineKeyboard{InlineKeyboard: rows}
}

// SkipKeyboard — metin girişi adımlarında "Geç" butonu gösterir
func SkipKeyboard(stepKey string) *TGInlineKeyboard {
	return &TGInlineKeyboard{InlineKeyboard: [][]TGInlineButton{
		{{Text: "Geç ➡️", CallbackData: "wiz_skip_" + stepKey}},
		{{Text: "❌ İptal", CallbackData: "wizard_cancel"}},
	}}
}

func YesNoKeyboard(yesData, noData string) *TGInlineKeyboard {
	return &TGInlineKeyboard{InlineKeyboard: [][]TGInlineButton{
		{{Text: "✅ Evet", CallbackData: yesData}, {Text: "❌ Hayır", CallbackData: noData}},
	}}
}

// ── Label helpers ─────────────────────────────────────────────

func priorityLabel(p string) string {
	switch p {
	case "dusuk":  return "Düşük"
	case "normal": return "Normal"
	case "yuksek": return "Yüksek"
	case "acil":   return "Acil 🔴"
	}
	return p
}

func statusLabel(s string) string {
	switch s {
	case "bekliyor":     return "Bekliyor"
	case "devam_ediyor": return "Devam Ediyor"
	case "tamamlandi":   return "Tamamlandı ✅"
	case "iptal":        return "İptal ❌"
	}
	return s
}
