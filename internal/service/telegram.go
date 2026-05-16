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

type TelegramService struct {
	token    string
	enabled  bool
	userRepo userRepo
}

type userRepo interface {
	ListWithChatIDs() ([]model.User, error)
}

type tgResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
}

type tgUpdate struct {
	UpdateID int64    `json:"update_id"`
	Message  *tgMsg   `json:"message"`
}

type tgMsg struct {
	MessageID int64   `json:"message_id"`
	Chat      tgChat  `json:"chat"`
	Text      string  `json:"text"`
}

type tgChat struct {
	ID int64 `json:"id"`
}

func NewTelegramService(cfg *config.TelegramConfig, repo userRepo) *TelegramService {
	return &TelegramService{
		token:    cfg.BotToken,
		enabled:  cfg.Enabled && cfg.BotToken != "",
		userRepo: repo,
	}
}

func (s *TelegramService) send(chatID, text string) {
	if !s.enabled || chatID == "" {
		return
	}
	body, _ := json.Marshal(map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	})
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.token)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[telegram] sendMessage hatası: %v", err)
		return
	}
	resp.Body.Close()
}

func (s *TelegramService) NotifyAssigned(task *model.Task, assignees []model.TaskUser) {
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

// StartPolling starts a long-polling goroutine to handle /chatid commands from users.
// Users send /chatid to the bot, bot replies with their chat_id which admin can then set.
func (s *TelegramService) StartPolling() {
	if !s.enabled {
		return
	}
	go func() {
		var offset int64
		client := &http.Client{Timeout: 35 * time.Second}
		log.Println("[telegram] polling başlatıldı")
		for {
			updates, err := s.getUpdates(client, offset)
			if err != nil {
				log.Printf("[telegram] getUpdates hatası: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			for _, u := range updates {
				offset = u.UpdateID + 1
				if u.Message == nil {
					continue
				}
				if u.Message.Text == "/chatid" || u.Message.Text == "/start" {
					reply := fmt.Sprintf("Chat ID'niz: <code>%d</code>\n\nBu numarayı yöneticiye bildirin.", u.Message.Chat.ID)
					s.send(fmt.Sprintf("%d", u.Message.Chat.ID), reply)
				}
			}
		}
	}()
}

func (s *TelegramService) getUpdates(client *http.Client, offset int64) ([]tgUpdate, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=30&offset=%d", s.token, offset)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tgResp struct {
		OK     bool       `json:"ok"`
		Result []tgUpdate `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
		return nil, err
	}
	return tgResp.Result, nil
}

func priorityLabel(p string) string {
	switch p {
	case "dusuk":    return "Düşük"
	case "normal":   return "Normal"
	case "yuksek":   return "Yüksek"
	case "acil":     return "Acil 🔴"
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
