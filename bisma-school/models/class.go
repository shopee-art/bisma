package models

import (
	"context"
	"bisma-school/config"
)

// Class mewakili struktur tabel 'classes'
type Class struct {
	ID        int
	ClassName string
}

// GetAllClasses mengambil semua data kelas dari database
func GetAllClasses() ([]Class, error) {
	var classes []Class
	query := "SELECT id, class_name FROM classes ORDER BY class_name ASC"
	
	rows, err := config.DB.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c Class
		err := rows.Scan(&c.ID, &c.ClassName)
		if err != nil {
			return nil, err
		}
		classes = append(classes, c)
	}
	return classes, nil
}

// CreateClass menambah data kelas baru ke database
func CreateClass(className string) error {
	query := "INSERT INTO classes (class_name) VALUES ($1)"
	_, err := config.DB.Exec(context.Background(), query, className)
	return err
}

// UpdateClass memperbarui nama kelas berdasarkan ID
func UpdateClass(id int, className string) error {
	query := "UPDATE classes SET class_name = $1 WHERE id = $2"
	_, err := config.DB.Exec(context.Background(), query, className, id)
	return err
}

// DeleteClass menghapus kelas dari database
func DeleteClass(id int) error {
	query := "DELETE FROM classes WHERE id = $1"
	_, err := config.DB.Exec(context.Background(), query, id)
	return err
}