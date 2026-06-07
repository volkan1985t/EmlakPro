package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/volkan1985t/EmlakPro/internal/model"
)

type CustomerRepository struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) List(userID int64, isAdmin bool, search string) ([]model.Customer, error) {
	args  := []interface{}{}
	where := []string{}
	i := 1

	if !isAdmin {
		where = append(where, fmt.Sprintf("c.user_id = $%d", i))
		args = append(args, userID); i++
	}
	if search != "" {
		where = append(where, fmt.Sprintf(
			"(c.name ILIKE $%d OR c.phone ILIKE $%d OR c.email ILIKE $%d)", i, i, i))
		args = append(args, "%"+search+"%"); i++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT c.id, c.user_id, c.name, c.phone, c.email, c.source,
		       COALESCE(c.notes,''), c.is_active, c.created_at, c.updated_at,
		       u.full_name as owner_name
		FROM customers c
		JOIN users u ON u.id = c.user_id
		%s
		ORDER BY c.created_at DESC`, whereClause), args...)
	if err != nil { return nil, err }
	defer rows.Close()

	var customers []model.Customer
	for rows.Next() {
		var c model.Customer
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Phone, &c.Email, &c.Source,
			&c.Notes, &c.IsActive, &c.CreatedAt, &c.UpdatedAt, &c.OwnerName); err != nil {
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, nil
}

func (r *CustomerRepository) GetByID(id int64) (*model.Customer, error) {
	var c model.Customer
	err := r.db.QueryRow(`
		SELECT c.id, c.user_id, c.name, c.phone, c.email, c.source,
		       COALESCE(c.notes,''), c.is_active, c.created_at, c.updated_at,
		       u.full_name as owner_name
		FROM customers c
		JOIN users u ON u.id = c.user_id
		WHERE c.id = $1`, id,
	).Scan(&c.ID, &c.UserID, &c.Name, &c.Phone, &c.Email, &c.Source,
		&c.Notes, &c.IsActive, &c.CreatedAt, &c.UpdatedAt, &c.OwnerName)
	if err == sql.ErrNoRows { return nil, nil }
	if err != nil { return nil, err }
	return &c, nil
}

// FindDuplicate — aynı danışmanın kayıtlarında isim VEYA telefon eşleşmesi arar.
// Bulursa eşleşen müşteriyi, bulamazsa nil döner.
func (r *CustomerRepository) FindDuplicate(userID int64, name, phone string) (*model.Customer, error) {
	name = strings.TrimSpace(name)
	phone = strings.TrimSpace(phone)
	// Telefondan rakam-dışı karakterleri arındırarak karşılaştır
	digits := func(s string) string {
		var b strings.Builder
		for _, ch := range s {
			if ch >= '0' && ch <= '9' { b.WriteRune(ch) }
		}
		return b.String()
	}
	phoneDigits := digits(phone)

	conds := []string{}
	args := []interface{}{userID}
	i := 2
	if name != "" {
		conds = append(conds, fmt.Sprintf("LOWER(TRIM(c.name)) = LOWER(TRIM($%d))", i))
		args = append(args, name); i++
	}
	if phoneDigits != "" {
		conds = append(conds, fmt.Sprintf("regexp_replace(COALESCE(c.phone,''), '[^0-9]', '', 'g') = $%d", i))
		args = append(args, phoneDigits); i++
	}
	if len(conds) == 0 {
		return nil, nil
	}
	query := fmt.Sprintf(`
		SELECT c.id, c.user_id, c.name, c.phone, c.email, COALESCE(c.notes,''), c.is_active, c.created_at, c.updated_at
		FROM customers c
		WHERE c.user_id = $1 AND (%s)
		LIMIT 1`, strings.Join(conds, " OR "))

	var cust model.Customer
	err := r.db.QueryRow(query, args...).Scan(
		&cust.ID, &cust.UserID, &cust.Name, &cust.Phone, &cust.Email,
		&cust.Notes, &cust.IsActive, &cust.CreatedAt, &cust.UpdatedAt)
	if err != nil {
		return nil, nil // bulunamadı (sql.ErrNoRows dahil) → mükerrer yok
	}
	return &cust, nil
}

func (r *CustomerRepository) Create(c *model.Customer) error {
	return r.db.QueryRow(`
		INSERT INTO customers (user_id, name, phone, email, source, notes)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at, updated_at`,
		c.UserID, c.Name, c.Phone, c.Email, c.Source, c.Notes,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *CustomerRepository) Update(c *model.Customer) error {
	_, err := r.db.Exec(`
		UPDATE customers SET name=$1, phone=$2, email=$3, source=$4, notes=$5, updated_at=NOW()
		WHERE id=$6`,
		c.Name, c.Phone, c.Email, c.Source, c.Notes, c.ID)
	return err
}

func (r *CustomerRepository) SetActive(id int64, active bool) error {
	_, err := r.db.Exec(`UPDATE customers SET is_active=$1, updated_at=NOW() WHERE id=$2`, active, id)
	return err
}

func (r *CustomerRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM customers WHERE id=$1`, id)
	return err
}

func (r *CustomerRepository) IsOwner(customerID, userID int64) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM customers WHERE id=$1 AND user_id=$2`, customerID, userID,
	).Scan(&count)
	return count > 0, err
}

func (r *CustomerRepository) LinkListing(customerID, listingID int64, note string) error {
	var existingCustomer int64
	r.db.QueryRow(`SELECT COALESCE(customer_id,0) FROM listings WHERE id=$1`, listingID).Scan(&existingCustomer)
	if existingCustomer > 0 && existingCustomer != customerID {
		return fmt.Errorf("bu ilan zaten baska bir musteriye bagli")
	}
	_, err := r.db.Exec(`UPDATE listings SET customer_id=$1, updated_at=NOW() WHERE id=$2`, customerID, listingID)
	return err
}
func (r *CustomerRepository) UnlinkListing(customerID, listingID int64) error {
	_, err := r.db.Exec(`UPDATE listings SET customer_id=NULL, updated_at=NOW() WHERE id=$1 AND customer_id=$2`, listingID, customerID)
	return err
}
func (r *CustomerRepository) GetLinkedListings(customerID int64) ([]model.Listing, error) {
	rows, err := r.db.Query(fmt.Sprintf(`
		SELECT %s
		FROM listings l
		JOIN users u ON u.id = l.user_id
		WHERE l.customer_id = $1
		ORDER BY l.created_at DESC`, listingSelectCols), customerID)
	if err != nil { return nil, err }
	defer rows.Close()
	var listings []model.Listing
	for rows.Next() {
		l, err := scanListing(rows)
		if err != nil { return nil, err }
		listings = append(listings, *l)
	}
	return listings, nil
}
