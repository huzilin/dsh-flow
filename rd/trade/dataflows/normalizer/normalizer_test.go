package normalizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnit_NormalizeAshareSymbol(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    NormalizedSymbol
		wantErr bool
	}{
		{
			name:  "A-share with exchange suffix",
			input: "000001.SZ",
			want: NormalizedSymbol{
				Original: "000001.SZ",
				Standard: "000001.SZ",
				Plain:    "000001",
				Market:   MarketAshare,
				Exchange: "SZ",
			},
			wantErr: false,
		},
		{
			name:  "A-share with exchange prefix",
			input: "SH600000",
			want: NormalizedSymbol{
				Original: "SH600000",
				Standard: "600000.SH",
				Plain:    "600000",
				Market:   MarketAshare,
				Exchange: "SH",
			},
			wantErr: false,
		},
		{
			name:  "A-share plain code 000xxx",
			input: "000001",
			want: NormalizedSymbol{
				Original: "000001",
				Standard: "000001.SZ",
				Plain:    "000001",
				Market:   MarketAshare,
				Exchange: "SZ",
			},
			wantErr: false,
		},
		{
			name:  "A-share plain code 600xxx",
			input: "600000",
			want: NormalizedSymbol{
				Original: "600000",
				Standard: "600000.SH",
				Plain:    "600000",
				Market:   MarketAshare,
				Exchange: "SH",
			},
			wantErr: false,
		},
		{
			name:  "A-share plain code 300xxx",
			input: "300001",
			want: NormalizedSymbol{
				Original: "300001",
				Standard: "300001.SZ",
				Plain:    "300001",
				Market:   MarketAshare,
				Exchange: "SZ",
			},
			wantErr: false,
		},
		{
			name:    "A-share invalid code",
			input:   "12345",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeAshareSymbol(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnit_NormalizeHKSymbol(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    NormalizedSymbol
		wantErr bool
	}{
		{
			name:  "HK-share with exchange suffix",
			input: "00700.HK",
			want: NormalizedSymbol{
				Original: "00700.HK",
				Standard: "00700.HK",
				Plain:    "00700",
				Market:   MarketHKShare,
				Exchange: "HK",
			},
			wantErr: false,
		},
		{
			name:  "HK-share with exchange prefix 5-digit",
			input: "HK00700",
			want: NormalizedSymbol{
				Original: "HK00700",
				Standard: "00700.HK",
				Plain:    "00700",
				Market:   MarketHKShare,
				Exchange: "HK",
			},
			wantErr: false,
		},
		{
			name:  "HK-share 5-digit code",
			input: "00700",
			want: NormalizedSymbol{
				Original: "00700",
				Standard: "00700.HK",
				Plain:    "00700",
				Market:   MarketHKShare,
				Exchange: "HK",
			},
			wantErr: false,
		},
		{
			name:  "HK-share 4-digit code",
			input: "0700",
			want: NormalizedSymbol{
				Original: "0700",
				Standard: "00700.HK",
				Plain:    "00700",
				Market:   MarketHKShare,
				Exchange: "HK",
			},
			wantErr: false,
		},
		{
			name:    "HK-share 3-digit code unsupported",
			input:   "700",
			wantErr: true,
		},
		{
			name:    "HK-share invalid code",
			input:   "123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeHKSymbol(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnit_DefaultNormalizer_Normalize(t *testing.T) {
	norm := NewDefaultNormalizer()

	tests := []struct {
		name       string
		input      string
		wantMarket MarketType
		wantErr    bool
	}{
		{"A-share SZ", "000001", MarketAshare, false},
		{"A-share SH", "600000", MarketAshare, false},
		{"HK-share", "00700", MarketHKShare, false},
		{"US-share", "AAPL", MarketUSShare, false},
		{"empty", "", MarketUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := norm.Normalize(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantMarket, got.Market)
		})
	}
}

func TestUnit_DefaultNormalizer_DetectMarket(t *testing.T) {
	norm := NewDefaultNormalizer()

	tests := []struct {
		name  string
		input string
		want  MarketType
	}{
		{"A-share SZ", "000001", MarketAshare},
		{"A-share SH", "600000", MarketAshare},
		{"A-share BJ", "830000", MarketAshare},
		{"HK-share", "00700", MarketHKShare},
		{"US-share", "AAPL", MarketUSShare},
		{"unknown", "INVALID", MarketUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := norm.DetectMarket(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnit_PadHKCode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"5-digit", "00700", "00700"},
		{"4-digit", "0700", "00700"},
		{"3-digit", "700", "00700"},
		{"2-digit", "70", "00070"},
		{"1-digit", "7", "00007"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padHKCode(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
