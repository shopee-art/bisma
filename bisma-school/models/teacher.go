package models

import (
	"bisma-school/config"
	"context"
)

// Struct tunggal untuk entitas login Guru / Admin
type Teacher struct {
	ID             int
	NIP            string
	Name           string
	Password       string
	Role           string
	ProfilePicture string 
	// 💡 Tambahkan dua field ini ke struct Teacher utama karena dipanggil di baris 126 Anda:
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
	ClassName      *string // 💡 Ubah jadi pointer agar tidak eror saat di-indirect (*t.ClassName)
	AdditionalRole *string // 💡 Ubah jadi pointer agar tidak eror saat dibandingkan dengan nil (== nil)
	ManagedClassID *int
}

// FindTeacherByNIP digunakan saat login
func FindTeacherByNIP(nip string) (Teacher, error) {
	ctx := context.Background()
	var t Teacher

	query := `SELECT id, nip, name, password, role, COALESCE(profile_picture, '') 
	          FROM teachers WHERE nip = $1`
	
	err := config.DB.QueryRow(ctx, query, nip).Scan(
		&t.ID, &t.NIP, &t.Name, &t.Password, &t.Role, &t.ProfilePicture,
	)
	return t, err
}

// GetAdditionalRoleStr adalah fungsi pembantu untuk template HTML agar aman dari error pointer
func (t TeacherList) GetAdditionalRoleStr() string {
	if t.AdditionalRole == nil {
		return ""
	}
	return *t.AdditionalRole
}

// GetClassNameStr adalah fungsi pembantu untuk template HTML
func (t TeacherList) GetClassNameStr() string {
	if t.ClassName != nil && *t.ClassName == "" {
		return ""
	}
	return *t.ClassName
}

// GetFilteredTeachers menarik data guru dengan fitur Pencarian dan Pagination
func GetFilteredTeachers(limit, offset int, search string) ([]TeacherList, int, error) {
	ctx := context.Background()
	var teachers []TeacherList
	var totalData int

	searchParam := "%" + search + "%"

	// 1. Hitung total data guru
	countQuery := "SELECT COUNT(id) FROM teachers WHERE name ILIKE $1 OR nip ILIKE $1"
	err := config.DB.QueryRow(ctx, countQuery, searchParam).Scan(&totalData)
	if err != nil {
		return nil, 0, err
	}

	// 2. Tarik data terfilter
	query := `SELECT t.id, t.nip, t.name, t.role, t.additional_role, c.class_name 
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

	for rows.Next() {
		var tl TeacherList
		err := rows.Scan(&tl.ID, &tl.NIP, &tl.Name, &tl.Role, &tl.AdditionalRole, &tl.ClassName)
		if err != nil {
			return nil, 0, err
		}
		teachers = append(teachers, tl)
	}
	return teachers, totalData, nil
}

// UpdateTeacher memperbarui data guru di database
func UpdateTeacher(id int, nip, name, role, additionalRole string, managedClassID *int) error {
	query := `UPDATE teachers SET nip = $1, name = $2, role = $3, 
			  additional_role = NULLIF($4, ''), managed_class_id = $5 WHERE id = $6`
	_, err := config.DB.Exec(context.Background(), query, nip, name, role, additionalRole, managedClassID, id)
	return err
}

// DeleteTeacher menghapus data guru
func DeleteTeacher(id int) error {
	query := "DELETE FROM teachers WHERE id = $1"
	_, err := config.DB.Exec(context.Background(), query, id)
	return err
}

// ImportTeachersBulk memasukkan banyak data guru sekaligus (Transaction)
func ImportTeachersBulk(teachers []Teacher) error {
	ctx := context.Background()
	tx, err := config.DB.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)

	query := `INSERT INTO teachers (nip, name, password, role, additional_role, managed_class_id) 
			  VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)`

	for _, t := range teachers {
		_, err := tx.Exec(ctx, query, t.NIP, t.Name, t.Password, t.Role, t.AdditionalRole, t.ManagedClassID)
		if err != nil { return err }
	}
	return tx.Commit(ctx)
}

// CreateTeacher menambah data guru baru ke database
func CreateTeacher(nip, name, password, role, additionalRole string, managedClassID *int) error {
	query := `INSERT INTO teachers (nip, name, password, role, additional_role, managed_class_id) 
			  VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)`
	
	_, err := config.DB.Exec(context.Background(), query, nip, name, password, role, additionalRole, managedClassID)
	return err
}