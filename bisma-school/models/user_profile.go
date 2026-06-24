package models

import (
	"bisma-school/config"
	"context"
	"errors"
	"fmt"
)

type UserProfile struct {
	ID             int
	UsernameOrNIP  string
	Name           string
	BirthPlace     string
	BirthDate      string 
	Hobby          string
	ProfilePicture string
}

// GetUserProfileByID adaptif terhadap Admin & Guru yang berada di satu tabel
func GetUserProfileByID(id int, role string) (UserProfile, error) {
	ctx := context.Background()
	var prof UserProfile
	prof.ID = id

	var query string
	// Jika admin atau guru, arahkan ke tabel teachers
	if role == "admin" || role == "teacher" || role == "guru" {
		query = "SELECT nip, name, COALESCE(birth_place, ''), COALESCE(TO_CHAR(birth_date, 'YYYY-MM-DD'), ''), COALESCE(hobby, ''), COALESCE(profile_picture, '') FROM teachers WHERE id = $1"
	} else if role == "siswa" {
		query = "SELECT nis, name, COALESCE(birth_place, ''), COALESCE(TO_CHAR(birth_date, 'YYYY-MM-DD'), ''), COALESCE(hobby, ''), COALESCE(profile_picture, '') FROM students WHERE id = $1"
	} else {
		return prof, errors.New("role tidak dikenali")
	}

	err := config.DB.QueryRow(ctx, query, id).Scan(&prof.UsernameOrNIP, &prof.Name, &prof.BirthPlace, &prof.BirthDate, &prof.Hobby, &prof.ProfilePicture)
	return prof, err
}

// UpdateUserProfile mengarahkan Admin & Guru ke tabel teachers
func UpdateUserProfile(id int, role, birthPlace, birthDate, hobby string) error {
	ctx := context.Background()
	var query string

	tabel := "students"
	if role == "admin" || role == "teacher" || role == "guru" {
		tabel = "teachers"
	}

	if birthDate == "" {
		query = fmt.Sprintf("UPDATE %s SET birth_place = $1, birth_date = NULL, hobby = $2 WHERE id = $3", tabel)
		_, err := config.DB.Exec(ctx, query, birthPlace, hobby, id)
		return err
	}

	query = fmt.Sprintf("UPDATE %s SET birth_place = $1, birth_date = $2, hobby = $3 WHERE id = $4", tabel)
	_, err := config.DB.Exec(ctx, query, birthPlace, birthDate, hobby, id)
	return err
}

// UpdateProfilePicture untuk foto profil
func UpdateProfilePicture(id int, role, filePath string) error {
	tabel := "students"
	if role == "admin" || role == "teacher" || role == "guru" {
		tabel = "teachers"
	}

	query := fmt.Sprintf("UPDATE %s SET profile_picture = $1 WHERE id = $2", tabel)
	_, err := config.DB.Exec(context.Background(), query, filePath, id)
	return err
}

// UpdateUserPassword untuk ganti password
func UpdateUserPassword(id int, role, oldPwd, newPwd string) error {
	ctx := context.Background()
	tabel := "students"
	if role == "admin" || role == "teacher" || role == "guru" {
		tabel = "teachers"
	}

	var currentPwd string
	checkQuery := fmt.Sprintf("SELECT password FROM %s WHERE id = $1", tabel)
	err := config.DB.QueryRow(ctx, checkQuery, id).Scan(&currentPwd)
	if err != nil {
		return err
	}

	if currentPwd != oldPwd {
		return errors.New("password lama tidak sesuai")
	}

	updateQuery := fmt.Sprintf("UPDATE %s SET password = $1 WHERE id = $2", tabel)
	_, err = config.DB.Exec(ctx, updateQuery, newPwd, id)
	return err
}