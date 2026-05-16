package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/volkan1985t/EmlakPro/internal/model"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) List(f model.TaskFilter) ([]model.Task, error) {
	where := []string{"t.parent_id IS NULL"}
	args := []interface{}{}
	i := 1

	if f.Status != "" {
		where = append(where, fmt.Sprintf("t.status=$%d", i))
		args = append(args, f.Status); i++
	}
	if f.Priority != "" {
		where = append(where, fmt.Sprintf("t.priority=$%d", i))
		args = append(args, f.Priority); i++
	}
	if f.UserID > 0 {
		where = append(where, fmt.Sprintf("(t.created_by=$%d OR EXISTS(SELECT 1 FROM task_assignees ta WHERE ta.task_id=t.id AND ta.user_id=$%d))", i, i+1))
		args = append(args, f.UserID, f.UserID); i += 2
	}

	q := fmt.Sprintf(`
		SELECT t.id, t.parent_id, t.title, t.description, t.status, t.priority,
		       t.due_date, t.created_by, u.full_name, t.created_at, t.updated_at
		FROM tasks t JOIN users u ON u.id=t.created_by
		WHERE %s
		ORDER BY t.priority DESC, t.created_at DESC`, strings.Join(where, " AND "))

	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *t)
	}
	return tasks, nil
}

func (r *TaskRepository) GetByID(id int64) (*model.Task, error) {
	row := r.db.QueryRow(`
		SELECT t.id, t.parent_id, t.title, t.description, t.status, t.priority,
		       t.due_date, t.created_by, u.full_name, t.created_at, t.updated_at
		FROM tasks t JOIN users u ON u.id=t.created_by
		WHERE t.id=$1`, id)
	t, err := scanTask(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (r *TaskRepository) GetSubtasks(parentID int64) ([]model.Task, error) {
	rows, err := r.db.Query(`
		SELECT t.id, t.parent_id, t.title, t.description, t.status, t.priority,
		       t.due_date, t.created_by, u.full_name, t.created_at, t.updated_at
		FROM tasks t JOIN users u ON u.id=t.created_by
		WHERE t.parent_id=$1
		ORDER BY t.created_at ASC`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []model.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *t)
	}
	return tasks, nil
}

func (r *TaskRepository) Create(req *model.CreateTaskRequest, createdBy int64) (*model.Task, error) {
	t := &model.Task{}
	err := r.db.QueryRow(`
		INSERT INTO tasks (parent_id, title, description, status, priority, due_date, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, parent_id, title, description, status, priority, due_date, created_by, created_at, updated_at`,
		intOrNull(req.ParentID), req.Title, req.Description,
		statusOrDefault(req.Status), priorityOrDefault(req.Priority),
		req.DueDate, createdBy,
	).Scan(&t.ID, &t.ParentID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&t.DueDate, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (r *TaskRepository) Update(id int64, req *model.UpdateTaskRequest) error {
	_, err := r.db.Exec(`
		UPDATE tasks SET title=$1, description=$2, status=$3, priority=$4, due_date=$5, updated_at=NOW()
		WHERE id=$6`,
		req.Title, req.Description, req.Status, req.Priority, req.DueDate, id)
	return err
}

func (r *TaskRepository) UpdateStatus(id int64, status string) error {
	_, err := r.db.Exec(`UPDATE tasks SET status=$1, updated_at=NOW() WHERE id=$2`, status, id)
	return err
}

func (r *TaskRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM tasks WHERE id=$1`, id)
	return err
}

// ── Assignees ─────────────────────────────────────────────────

func (r *TaskRepository) GetAssignees(taskID int64) ([]model.TaskUser, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.full_name FROM task_assignees ta
		JOIN users u ON u.id=ta.user_id
		WHERE ta.task_id=$1 ORDER BY u.full_name`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.TaskUser
	for rows.Next() {
		var tu model.TaskUser
		rows.Scan(&tu.ID, &tu.FullName)
		out = append(out, tu)
	}
	return out, nil
}

func (r *TaskRepository) SetAssignees(taskID int64, userIDs []int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM task_assignees WHERE task_id=$1`, taskID); err != nil {
		return err
	}
	for _, uid := range userIDs {
		if _, err := tx.Exec(`INSERT INTO task_assignees(task_id,user_id) VALUES($1,$2)`, taskID, uid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ── Comments ──────────────────────────────────────────────────

func (r *TaskRepository) GetComments(taskID int64) ([]model.TaskComment, error) {
	rows, err := r.db.Query(`
		SELECT c.id, c.task_id, c.user_id, u.full_name, c.body, c.created_at
		FROM task_comments c JOIN users u ON u.id=c.user_id
		WHERE c.task_id=$1 ORDER BY c.created_at ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.TaskComment
	for rows.Next() {
		var c model.TaskComment
		rows.Scan(&c.ID, &c.TaskID, &c.UserID, &c.UserName, &c.Body, &c.CreatedAt)
		out = append(out, c)
	}
	return out, nil
}

func (r *TaskRepository) AddComment(taskID, userID int64, body string) (*model.TaskComment, error) {
	c := &model.TaskComment{}
	err := r.db.QueryRow(`
		INSERT INTO task_comments(task_id,user_id,body) VALUES($1,$2,$3)
		RETURNING id,task_id,user_id,body,created_at`,
		taskID, userID, body,
	).Scan(&c.ID, &c.TaskID, &c.UserID, &c.Body, &c.CreatedAt)
	return c, err
}

func (r *TaskRepository) DeleteComment(id, userID int64, isAdmin bool) error {
	var err error
	if isAdmin {
		_, err = r.db.Exec(`DELETE FROM task_comments WHERE id=$1`, id)
	} else {
		_, err = r.db.Exec(`DELETE FROM task_comments WHERE id=$1 AND user_id=$2`, id, userID)
	}
	return err
}

// ── Images ────────────────────────────────────────────────────

func (r *TaskRepository) GetImages(taskID int64) ([]model.TaskImage, error) {
	rows, err := r.db.Query(`
		SELECT id, task_id, path, uploaded_by, created_at
		FROM task_images WHERE task_id=$1 ORDER BY created_at ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.TaskImage
	for rows.Next() {
		var img model.TaskImage
		rows.Scan(&img.ID, &img.TaskID, &img.Path, &img.UploadedBy, &img.CreatedAt)
		out = append(out, img)
	}
	return out, nil
}

func (r *TaskRepository) AddImage(taskID, userID int64, path string) (*model.TaskImage, error) {
	img := &model.TaskImage{}
	err := r.db.QueryRow(`
		INSERT INTO task_images(task_id,path,uploaded_by) VALUES($1,$2,$3)
		RETURNING id,task_id,path,uploaded_by,created_at`,
		taskID, path, userID,
	).Scan(&img.ID, &img.TaskID, &img.Path, &img.UploadedBy, &img.CreatedAt)
	return img, err
}

func (r *TaskRepository) DeleteImage(id int64) (string, error) {
	var path string
	err := r.db.QueryRow(`DELETE FROM task_images WHERE id=$1 RETURNING path`, id).Scan(&path)
	return path, err
}

// ── helpers ───────────────────────────────────────────────────

type rowScanner interface {
	Scan(...interface{}) error
}

func scanTask(row rowScanner) (*model.Task, error) {
	t := &model.Task{}
	err := row.Scan(&t.ID, &t.ParentID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&t.DueDate, &t.CreatedBy, &t.CreatorName, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func intOrNull(p *int64) interface{} {
	if p == nil { return nil }
	return *p
}

func statusOrDefault(s string) string {
	switch s {
	case "bekliyor", "devam_ediyor", "tamamlandi", "iptal":
		return s
	}
	return "bekliyor"
}

func priorityOrDefault(s string) string {
	switch s {
	case "dusuk", "normal", "yuksek", "acil":
		return s
	}
	return "normal"
}
