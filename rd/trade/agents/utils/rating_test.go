package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnit_ParseRating(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		defaultRating string
		expected      string
	}{
		{"Rating label Buy", "Rating: Buy", "", "Buy"},
		{"Rating label Overweight", "Rating: Overweight", "", "Overweight"},
		{"Rating label Hold", "Rating: Hold", "", "Hold"},
		{"Rating label Underweight", "Rating: Underweight", "", "Underweight"},
		{"Rating label Sell", "Rating: Sell", "", "Sell"},
		{"Markdown bold rating", "**Rating**: **Buy**", "", "Buy"},
		{"Rating with dash", "rating - Sell", "", "Sell"},
		{"Rating in lowercase", "rating: buy", "", "Buy"},
		{"Rating with extra spaces", "Rating:   Overweight   ", "", "Overweight"},
		{"First rating word in text", "Some text Buy more text", "", "Buy"},
		{"Multiple ratings takes first", "Sell something Buy", "", "Sell"},
		{"Default rating when not found", "No rating here", "", "Hold"},
		{"Custom default rating", "No rating here", "Sell", "Sell"},
		{"Rating in markdown table", "| Rating | Buy |", "", "Buy"},
		{"Rating with punctuation", "Rating: Buy.", "", "Buy"},
		{"Rating with colon and asterisks", "**Rating**:*Hold*", "", "Hold"},
		{"Empty text", "", "", "Hold"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseRating(tt.text, tt.defaultRating)
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
		{"Lowercase", "buy", "Buy"},
		{"Uppercase", "BUY", "BUY"},
		{"Empty", "", ""},
		{"Single char", "b", "B"},
		{"Mixed case", "oVeRwEiGhT", "OVeRwEiGhT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := capitalize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
