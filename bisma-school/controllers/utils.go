package controllers

import (
	"strconv"
	"strings"
)

// ParseIntFromString adalah helper untuk parsing string ke int dengan aman
func ParseIntFromString(s string) (int, error) {
	s = strings.TrimSpace(s)
	return strconv.Atoi(s)
}

// ValidateNonEmpty memeriksa apakah string tidak kosong setelah trim
func ValidateNonEmpty(s string) bool {
	return strings.TrimSpace(s) != ""
}

// ValidateMinLength memeriksa apakah string memiliki panjang minimum
func ValidateMinLength(s string, minLen int) bool {
	return len(strings.TrimSpace(s)) >= minLen
}

// ValidateMaxLength memeriksa apakah string tidak melebihi panjang maksimum
func ValidateMaxLength(s string, maxLen int) bool {
	return len(s) <= maxLen
}
