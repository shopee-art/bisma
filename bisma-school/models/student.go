package models

import (
	"bisma-school/config"
	"context"
	"time"

	"github.com/jackc/pgx/v5"
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

// GetFilteredStudents mendukung Pencarian, Pagination, dan Filter per Kelas dengan optimasi
func GetFilteredStudents(limit, offset int, search string, classID int) ([]StudentList, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var students []StudentList
	var totalData int

	searchParam := "%" + search + "%"

	// 1. Hitung total data dengan single query (lebih efisien)
	countQuery := `SELECT COUNT(s.id) FROM students s
				  WHERE (s.name ILIKE $1 OR s.nis ILIKE $1)
				  AND ($2 = 0 OR s.class_id = $2)`
	err := config.DB.QueryRow(ctx, countQuery, searchParam, classID).Scan(&totalData)
	if err != nil {
		return nil, 0, err
	}

	// 2. Tarik data siswa dengan pre-allocated slice (efisiensi memory)
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

	// Pre-allocate slice untuk menghindari banyak reallocation
	students = make([]StudentList, 0, limit)

	for rows.Next() {
		var sl StudentList
		err := rows.Scan(&sl.ID, &sl.NIS, &sl.Name, &sl.ClassName)
		if err != nil {
			return nil, 0, err
		}
		students = append(students, sl)
	}

	return students, totalData, rows.Err()
}

// UpdateStudent mengubah data siswa (Tanpa mengubah password jika dikosongkan)
func UpdateStudent(id int, nis, name string, classID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := "UPDATE students SET nis = $1, name = $2, class_id = $3 WHERE id = $4"
	_, err := config.DB.Exec(ctx, query, nis, name, classID, id)
	return err
}

// DeleteStudent menghapus data siswa dari database
func DeleteStudent(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := "DELETE FROM students WHERE id = $1"
	_, err := config.DB.Exec(ctx, query, id)
	return err
}

// CreateStudent menambah data siswa baru ke database
func CreateStudent(nis, name, password string, classID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := "INSERT INTO students (nis, name, password, class_id) VALUES ($1, $2, $3, $4)"
	_, err := config.DB.Exec(ctx, query, nis, name, password, classID)
	return err
}

// copyFromSource adalah helper untuk CopyFrom
type copyFromSource struct {
	rows [][]interface{}
	idx  int
}

func (c *copyFromSource) Next() bool {
	c.idx++
	return c.idx <= len(c.rows)
}

func (c *copyFromSource) Values() ([]interface{}, error) {
	return c.rows[c.idx-1], nil
}

func (c *copyFromSource) Err() error {
	return nil
}

// ImportStudentsBulk memasukkan data siswa secara massal dengan batch insert (lebih cepat dari transaction per-row)
func ImportStudentsBulk(students []Student) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := config.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Gunakan CopyFrom untuk bulk insert yang super cepat
	rows := make([][]interface{}, 0, len(students))
	for _, s := range students {
		rows = append(rows, []interface{}{s.NIS, s.Name, s.Password, s.ClassID})
	}

	// Gunakan pgx.Identifier untuk table name (bukan string biasa)
	count, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"students"},
		[]string{"nis", "name", "password", "class_id"},
		&copyFromSource{rows: rows},
	)

	if err != nil {
		return err
	}

	if int(count) != len(students) {
		return err // Tidak semua baris berhasil di-insert
	}

	return tx.Commit(ctx)
}
