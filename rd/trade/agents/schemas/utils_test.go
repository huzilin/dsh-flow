package schemas

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnit_JoinLines(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{"Empty", []string{}, ""},
		{"Single part", []string{"hello"}, "hello"},
		{"Multiple parts", []string{"hello", "world", "test"}, "hello\nworld\ntest"},
		{"With empty parts", []string{"hello", "", "world"}, "hello\n\nworld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JoinLines(tt.parts...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnit_Float64ToString(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{"Zero", 0, "0"},
		{"Positive", 123.456, "123.456"},
		{"Negative", -123.456, "-123.456"},
		{"Integer", 42, "42"},
		{"Large number", 1e6, "1000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Float64ToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnit_FormatFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{"Zero", 0, "0.0"},
		{"Positive", 123.456, "123.5"},
		{"Negative", -123.456, "-123.5"},
		{"Integer", 42, "42.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFloat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnit_Capitalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Lowercase", "hello", "Hello"},
		{"Uppercase", "HELLO", "HELLO"},
		{"Empty", "", ""},
		{"Single char", "a", "A"},
		{"Already capitalized", "Hello", "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Capitalize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
