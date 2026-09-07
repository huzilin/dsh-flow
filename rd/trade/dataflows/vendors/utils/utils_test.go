package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnit_NormalizeSymbol(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"AAPL", "AAPL"},
		{"aapl", "AAPL"},
		{"  msft  ", "MSFT"},
		{"XAUUSD", "GC=F"},
		{"XAUUSD+", "GC=F"},
		{"GOLD", "GC=F"},
		{"XAGUSD", "SI=F"},
		{"SILVER", "SI=F"},
		{"BTC", "BTC-USD"},
		{"BTC-USD", "BTC-USD"},
		{"ETH", "ETH-USD"},
		{"ETH-USD", "ETH-USD"},
		{"TSLA", "TSLA"},
		{"", ""},
	}

	for _, tc := range testCases {
		result := NormalizeSymbol(tc.input)
		assert.Equal(t, tc.expected, result)
	}
}

func TestUnit_FormatDate(t *testing.T) {
	testTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	result := FormatDate(testTime)
	assert.Equal(t, "2024-03-15", result)
}

func TestUnit_FormatDateTime(t *testing.T) {
	testTime := time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC)
	result := FormatDateTime(testTime)
	assert.Equal(t, "2024-03-15 10:30:45", result)
}

func TestUnit_ParseDate(t *testing.T) {
	testCases := []struct {
		input    string
		valid    bool
		expected time.Time
	}{
		{"2024-03-15", true, time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)},
		{"2024-12-31", true, time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)},
		{"2024-01-01", true, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"2024/03/15", false, time.Time{}},
		{"2024-13-01", false, time.Time{}},
		{"2024-02-30", false, time.Time{}},
	}

	for _, tc := range testCases {
		result, err := ParseDate(tc.input)
		if tc.valid {
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestUnit_ValidateSymbol(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"AAPL", true},
		{"MSFT", true},
		{"GOOGL", true},
		{"A", true},
		{"ABCDE", true},
		{"", false},
		{"AB", true},
		{"ABCDEF", false},
		{"123", false},
		{"A1", false},
		{"AAPL1", false},
		{"aapl", true},
	}

	for _, tc := range testCases {
		result := ValidateSymbol(tc.input)
		assert.Equal(t, tc.expected, result)
	}
}

func TestUnit_FormatLargeNumber(t *testing.T) {
	testCases := []struct {
		input    float64
		expected string
	}{
		{1500000000000.0, "1.50T"},
		{5000000000.0, "5.00B"},
		{1234567890.0, "1.23B"},
		{2000000.0, "2.00M"},
		{1500000.50, "1.50M"},
		{5000.0, "5.00K"},
		{1234.56, "1.23K"},
		{100.0, "100.00"},
		{12.34, "12.34"},
		{0.0, "0.00"},
		{0.1234, "0.12"},
	}

	for _, tc := range testCases {
		result := FormatLargeNumber(tc.input)
		assert.Equal(t, tc.expected, result)
	}
}

func TestUnit_AddDays(t *testing.T) {
	testCases := []struct {
		input    time.Time
		days     int
		expected time.Time
	}{
		{time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), 1, time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC)},
		{time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), -1, time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC)},
		{time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), 7, time.Date(2024, 3, 22, 0, 0, 0, 0, time.UTC)},
		{time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), 30, time.Date(2024, 4, 14, 0, 0, 0, 0, time.UTC)},
		{time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), 1, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), -1, time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)},
	}

	for _, tc := range testCases {
		result := AddDays(tc.input, tc.days)
		assert.Equal(t, tc.expected, result)
	}
}
