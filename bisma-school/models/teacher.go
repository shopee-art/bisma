package models

import (
	"bisma-school/config"
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Struct tunggal untuk entitas login Guru / Admin
type Teacher struct {
	ID             int
	NIP            string
	Name           string
	Password       string
	Role           string
	ProfilePicture string
	AdditionalRole *string
	ManagedClassID *int
}

// Struct untuk kebutuhan list tabel data master di halaman Admin
type TeacherList struct {
	ID             int
	NIP            string
	Name           string
	Role           string
	ProfilePicture string
	ClassName      *string
	AdditionalRole *string
	ManagedClassID *int
}

// FindTeacherByNIP digunakan saat login dengan context timeout
func FindTeacherByNIP(nip string) (Teacher, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var t Teacher

	query := `SELECT id, nip, name, password, role, COALESCE(profile_picture, '')
	          FROM teachers WHERE nip = $1`

	err := config.DB.QueryRow(ctx, query, nip).Scan(
		&t.ID, &t.NIP, &t.Name, &t.Password, &t.Role, &t.ProfilePicture,
	)
	return t, err
}

// GetAdditionalRoleStr adalah fungsi pembantu untuk template HTML
func (t TeacherList) GetAdditionalRoleStr() string {
	if t.AdditionalRole == nil {
		return ""
	}
	return *t.AdditionalRole
}

// GetClassNameStr adalah fungsi pembantu untuk template HTML
func (t TeacherList) GetClassNameStr() string {
	if t.ClassName == nil || *t.ClassName == "" {
		return ""
	}
	return *t.ClassName
}

// GetFilteredTeachers menarik data guru dengan fitur Pencarian dan Pagination
func GetFilteredTeachers(limit, offset int, search string) ([]TeacherList, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var teachers []TeacherList
	var totalData int

	searchParam := "%" + search + "%"

	// 1. Hitung total data guru
	countQuery := "SELECT COUNT(id) FROM teachers WHERE name ILIKE $1 OR nip ILIKE $1"
	err := config.DB.QueryRow(ctx, countQuery, searchParam).Scan(&totalData)
	if err != nil {
		return nil, 0, err
	}

	// 2. Tarik data terfilter dengan pre-allocated slice
	query := `SELECT t.id, t.nip, t.name, t.role, t.profile_picture, t.additional_role, c.class_name
			  FROM teachers t
			  LEFT JOIN classes c ON t.managed_class_id = c.id
			  WHERE t.name ILIKE $1 OR t.nip ILIKE $1
			  ORDER BY t.name ASC
			  LIMIT $2 OFFSET $3`

	rows, err := config.DB.Query(ctx, query, searchParam, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	teachers = make([]TeacherList, 0, limit)

	for rows.Next() {
		var tl TeacherList
		err := rows.Scan(&tl.ID, &tl.NIP, &tl.Name, &tl.Role, &tl.ProfilePicture, &tl.AdditionalRole, &tl.ClassName)
		if err != nil {
			return nil, 0, err
		}
		teachers = append(teachers, tl)
	}

	return teachers, totalData, rows.Err()
}

// UpdateTeacher memperbarui data guru di database
func UpdateTeacher(id int, nip, name, role, additionalRole string, managedClassID *int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `UPDATE teachers SET nip = $1, name = $2, role = $3,
			  additional_role = NULLIF($4, ''), managed_class_id = $5 WHERE id = $6`
	_, err := config.DB.Exec(ctx, query, nip, name, role, additionalRole, managedClassID, id)
	return err
}

// DeleteTeacher menghapus data guru
func DeleteTeacher(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := "DELETE FROM teachers WHERE id = $1"
	_, err := config.DB.Exec(ctx, query, id)
	return err
}

// copyFromSourceTeacher adalah helper untuk CopyFrom teacher
type copyFromSourceTeacher struct {
	rows [][]interface{}
	idx  int
}

func (c *copyFromSourceTeacher) Next() bool {
	c.idx++
	return c.idx <= len(c.rows)
}

func (c *copyFromSourceTeacher) Values() ([]interface{}, error) {
	return c.rows[c.idx-1], nil
}

func (c *copyFromSourceTeacher) Err() error {
	return nil
}

// ImportTeachersBulk memasukkan banyak data guru sekaligus dengan CopyFrom (batch insert cepat)
func ImportTeachersBulk(teachers []Teacher) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := config.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows := make([][]interface{}, 0, len(teachers))
	for _, t := range teachers {
		var addRole interface{} = nil
		if t.AdditionalRole != nil {
			addRole = *t.AdditionalRole
		}
		rows = append(rows, []interface{}{t.NIP, t.Name, t.Password, t.Role, addRole, t.ManagedClassID})
	}

	// Gunakan pgx.Identifier untuk table name (bukan string biasa)
	count, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"teachers"},
		[]string{"nip", "name", "password", "role", "additional_role", "managed_class_id"},
		&copyFromSourceTeacher{rows: rows},
	)

	if err != nil {
		return err
	}

	if int(count) != len(teachers) {
		return err
	}

	return tx.Commit(ctx)
}

// CreateTeacher menambah data guru baru ke database
func CreateTeacher(nip, name, password, role, additionalRole string, managedClassID *int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `INSERT INTO teachers (nip, name, password, role, additional_role, managed_class_id)
			  VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)`

	_, err := config.DB.Exec(ctx, query, nip, name, password, role, additionalRole, managedClassID)
	return err
}
