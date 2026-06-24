package config

import (
	"strconv"
)

// ParseInt adalah helper untuk parsing string ke int dengan error handling yang aman
func ParseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

// ParseIntPtr adalah helper untuk parsing string ke *int dengan aman
func ParseIntPtr(s string) *int {
	if s == "" {
		return nil
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &val
}

// TrimAndValidate adalah helper untuk validasi input string
func TrimAndValidate(input string, maxLen int) (string, bool) {
	if input == "" {
		return "", false
	}
	if len(input) > maxLen {
		return "", false
	}
	return input, true
}
