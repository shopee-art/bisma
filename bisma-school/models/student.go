package models

import (
	"bisma-school/config"
	"context"
)

// Struct tunggal untuk entitas login Siswa
type Student struct {
	ID             int
	NIS            string
	Name           string
	Password       string
	ClassID        int
	ProfilePicture string 
	ClassName      string 
}

// Struct untuk keperluan list tabel data master di halaman Admin
type StudentList struct {
	ID        int
	NIS       string
	Name      string
	ClassID   int
	ClassName string
}

// FindStudentByNIS digunakan saat login untuk mengambil data siswa dan kelasnya
func FindStudentByNIS(nis string) (Student, error) {
	ctx := context.Background()
	var s Student

	query := `SELECT s.id, s.nis, s.name, s.password, s.class_id, 
	                 COALESCE(s.profile_picture, ''), COALESCE(c.class_name, '') 
	          FROM students s
	          LEFT JOIN classes c ON s.class_id = c.id 
	          WHERE s.nis = $1`

	err := config.DB.QueryRow(ctx, query, nis).Scan(
		&s.ID, &s.NIS, &s.Name, &s.Password, &s.ClassID, &s.ProfilePicture, &s.ClassName,
	)
	return s, err
}

// GetFilteredStudents mendukung Pencarian, Pagination, dan Filter per Kelas
func GetFilteredStudents(limit, offset int, search string, classID int) ([]StudentList, int, error) {
	ctx := context.Background()
	var students []StudentList
	var totalData int

	searchParam := "%" + search + "%"

	// 1. Hitung total data berdasarkan filter
	countQuery := `SELECT COUNT(s.id) FROM students s 
				  WHERE (s.name ILIKE $1 OR s.nis ILIKE $1) 
				  AND ($2 = 0 OR s.class_id = $2)`
	err := config.DB.QueryRow(ctx, countQuery, searchParam, classID).Scan(&totalData)
	if err != nil {
		return nil, 0, err
	}

	// 2. Tarik data siswa sesuai limit, offset, pencarian, dan filter kelas
	query := `SELECT s.id, s.nis, s.name, c.class_name 
			  FROM students s
			  JOIN classes c ON s.class_id = c.id
			  WHERE (s.name ILIKE $1 OR s.nis ILIKE $1)
			  AND ($2 = 0 OR s.class_id = $2)
			  ORDER BY c.class_name ASC, s.name ASC
			  LIMIT $3 OFFSET $4`

	rows, err := config.DB.Query(ctx, query, searchParam, classID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var sl StudentList
		err := rows.Scan(&sl.ID, &sl.NIS, &sl.Name, &sl.ClassName)
		if err != nil {
			return nil, 0, err
		}
		students = append(students, sl)
	}
	return students, totalData, nil
}

// UpdateStudent mengubah data siswa (Tanpa mengubah password jika dikosongkan)
func UpdateStudent(id int, nis, name string, classID int) error {
	query := "UPDATE students SET nis = $1, name = $2, class_id = $3 WHERE id = $4"
	_, err := config.DB.Exec(context.Background(), query, nis, name, classID, id)
	return err
}

// DeleteStudent menghapus data siswa dari database
func DeleteStudent(id int) error {
	query := "DELETE FROM students WHERE id = $1"
	_, err := config.DB.Exec(context.Background(), query, id)
	return err
}

// ... (Fungsi CreateStudent & ImportStudentsBulk sebelumnya tetap biarkan di bawah) ...

// CreateStudent menambah data siswa baru ke database
func CreateStudent(nis, name, password string, classID int) error {
	query := "INSERT INTO students (nis, name, password, class_id) VALUES ($1, $2, $3, $4)"
	_, err := config.DB.Exec(context.Background(), query, nis, name, password, classID)
	return err
}

// ImportStudentsBulk memasukkan data siswa secara massal menggunakan database Transaction
func ImportStudentsBulk(students []Student) error {
	ctx := context.Background()
	tx, err := config.DB.Begin(ctx)
	if err != nil {
		return err
	}
	// Pastikan di-rollback jika terjadi kegagalan di tengah jalan
	defer tx.Rollback(ctx)

	query := "INSERT INTO students (nis, name, password, class_id) VALUES ($1, $2, $3, $4)"

	for _, s := range students {
		_, err := tx.Exec(ctx, query, s.NIS, s.Name, s.Password, s.ClassID)
		if err != nil {
			return err // Jika ada satu NIP/NIS duplikat, batalkan semua demi validitas data
		}
	}

	return tx.Commit(ctx)
}