package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/volkan1985t/EmlakPro/internal/middleware"
	"github.com/volkan1985t/EmlakPro/internal/model"
	"github.com/volkan1985t/EmlakPro/internal/repository"
	"github.com/volkan1985t/EmlakPro/internal/service"
)

type TaskHandler struct {
	repo     *repository.TaskRepository
	imageSvc *service.ImageService
	tg       TelegramNotifier
}

// TelegramNotifier allows nil injection without import cycle
type TelegramNotifier interface {
	NotifyAssigned(task *model.Task, assignees []model.TaskUser)
	NotifyStatusChanged(task *model.Task, oldStatus string)
}

func NewTaskHandler(repo *repository.TaskRepository, imageSvc *service.ImageService, tg TelegramNotifier) *TaskHandler {
	return &TaskHandler{repo: repo, imageSvc: imageSvc, tg: tg}
}

// GET /api/tasks
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	isAdmin := middleware.IsAdmin(r.Context())

	f := model.TaskFilter{
		Status:   r.URL.Query().Get("status"),
		Priority: r.URL.Query().Get("priority"),
	}
	if !isAdmin {
		f.UserID = userID
	}

	tasks, err := h.repo.List(f)
	if err != nil {
		jsonErr(w, "Görevler yüklenemedi", http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		tasks = []model.Task{}
	}
	for i := range tasks {
		assignees, _ := h.repo.GetAssignees(tasks[i].ID)
		if assignees != nil {
			tasks[i].Assignees = assignees
		}
		subtasks, _ := h.repo.GetSubtasks(tasks[i].ID)
		if subtasks != nil {
			tasks[i].Subtasks = subtasks
		}
	}
	jsonOK(w, tasks)
}

// GET /api/tasks/{id}
func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonErr(w, "Geçersiz ID", http.StatusBadRequest)
		return
	}
	task, err := h.repo.GetByID(id)
	if err != nil || task == nil {
		jsonErr(w, "Görev bulunamadı", http.StatusNotFound)
		return
	}
	task.Assignees, _ = h.repo.GetAssignees(id)
	task.Comments, _ = h.repo.GetComments(id)
	task.Images, _   = h.repo.GetImages(id)
	task.Subtasks, _ = h.repo.GetSubtasks(id)
	h.convertImagePaths(task)
	jsonOK(w, task)
}

func (h *TaskHandler) convertImagePaths(task *model.Task) {
	for i := range task.Images {
		task.Images[i].Path = h.imageSvc.PathToURL(task.Images[i].Path)
	}
}

// POST /api/tasks
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	var req model.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}
	if req.Title == "" {
		jsonErr(w, "Başlık zorunludur", http.StatusBadRequest)
		return
	}
	task, err := h.repo.Create(&req, userID)
	if err != nil {
		jsonErr(w, "Görev oluşturulamadı", http.StatusInternalServerError)
		return
	}
	if len(req.Assignees) > 0 {
		h.repo.SetAssignees(task.ID, req.Assignees)
	}
	task.Assignees, _ = h.repo.GetAssignees(task.ID)
	if h.tg != nil && len(task.Assignees) > 0 {
		h.tg.NotifyAssigned(task, task.Assignees)
	}
	jsonOK(w, task)
}

// PUT /api/tasks/{id}
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonErr(w, "Geçersiz ID", http.StatusBadRequest)
		return
	}
	var req model.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}

	existing, _ := h.repo.GetByID(id)
	if existing == nil {
		jsonErr(w, "Görev bulunamadı", http.StatusNotFound)
		return
	}

	oldStatus := existing.Status
	if err := h.repo.Update(id, &req); err != nil {
		jsonErr(w, "Görev güncellenemedi", http.StatusInternalServerError)
		return
	}
	if req.Assignees != nil {
		h.repo.SetAssignees(id, req.Assignees)
	}

	task, _ := h.repo.GetByID(id)
	task.Assignees, _ = h.repo.GetAssignees(id)

	if h.tg != nil && oldStatus != req.Status {
		h.tg.NotifyStatusChanged(task, oldStatus)
	}
	jsonOK(w, task)
}

// PATCH /api/tasks/{id}/status
func (h *TaskHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonErr(w, "Geçersiz ID", http.StatusBadRequest)
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	existing, _ := h.repo.GetByID(id)
	if existing == nil {
		jsonErr(w, "Görev bulunamadı", http.StatusNotFound)
		return
	}
	oldStatus := existing.Status
	if err := h.repo.UpdateStatus(id, body.Status); err != nil {
		jsonErr(w, "Durum güncellenemedi", http.StatusInternalServerError)
		return
	}
	task, _ := h.repo.GetByID(id)
	task.Assignees, _ = h.repo.GetAssignees(id)
	if h.tg != nil {
		h.tg.NotifyStatusChanged(task, oldStatus)
	}
	jsonOK(w, task)
}

// DELETE /api/tasks/{id}
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	isAdmin   := middleware.IsAdmin(r.Context())
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonErr(w, "Geçersiz ID", http.StatusBadRequest)
		return
	}
	task, _ := h.repo.GetByID(id)
	if task == nil {
		jsonErr(w, "Görev bulunamadı", http.StatusNotFound)
		return
	}
	if !isAdmin && task.CreatedBy != userID {
		jsonErr(w, "Yetki yok", http.StatusForbidden)
		return
	}
	h.repo.Delete(id)
	jsonOK(w, map[string]bool{"deleted": true})
}

// ── Comments ──────────────────────────────────────────────────

// POST /api/tasks/{id}/comments
func (h *TaskHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	userID, _ := middleware.GetUserID(r.Context())
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Body == "" {
		jsonErr(w, "Yorum boş olamaz", http.StatusBadRequest)
		return
	}
	c, err := h.repo.AddComment(id, userID, body.Body)
	if err != nil {
		jsonErr(w, "Yorum eklenemedi", http.StatusInternalServerError)
		return
	}
	jsonOK(w, c)
}

// DELETE /api/tasks/{id}/comments/{cid}
func (h *TaskHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	cid, _ := strconv.ParseInt(chi.URLParam(r, "cid"), 10, 64)
	userID, _ := middleware.GetUserID(r.Context())
	isAdmin := middleware.IsAdmin(r.Context())
	h.repo.DeleteComment(cid, userID, isAdmin)
	jsonOK(w, map[string]bool{"deleted": true})
}

// ── Images ────────────────────────────────────────────────────

// POST /api/tasks/{id}/images
func (h *TaskHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	userID, _ := middleware.GetUserID(r.Context())

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		jsonErr(w, "Form parse hatası", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		jsonErr(w, "Dosya okunamadı", http.StatusBadRequest)
		return
	}
	defer file.Close()

	result, err := h.imageSvc.SaveGallery(file, header.Filename, "tasks", 0)
	if err != nil {
		jsonErr(w, "Resim kaydedilemedi", http.StatusInternalServerError)
		return
	}
	img, err := h.repo.AddImage(id, userID, result.Path)
	if err != nil {
		jsonErr(w, "Resim kaydedilemedi", http.StatusInternalServerError)
		return
	}
	jsonOK(w, img)
}

// DELETE /api/tasks/{id}/images/{imgID}
func (h *TaskHandler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	imgID, _ := strconv.ParseInt(chi.URLParam(r, "imgID"), 10, 64)
	path, err := h.repo.DeleteImage(imgID)
	if err == nil && path != "" {
		h.imageSvc.DeleteFile(path)
	}
	jsonOK(w, map[string]bool{"deleted": true})
}
