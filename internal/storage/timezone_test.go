package storage

import (
	"testing"
	"time"
)

func TestParseTimezoneOffset(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantValid      bool
		wantMicrosecs  int64
	}{
		{
			name:          "UTC (Z)",
			input:         "Z",
			wantValid:     true,
			wantMicrosecs: 0,
		},
		{
			name:          "empty string",
			input:         "",
			wantValid:     true,
			wantMicrosecs: 0,
		},
		{
			name:          "negative offset -0500",
			input:         "-0500",
			wantValid:     true,
			wantMicrosecs: -5 * time.Hour.Microseconds(),
		},
		{
			name:          "positive offset +0200",
			input:         "+0200",
			wantValid:     true,
			wantMicrosecs: 2 * time.Hour.Microseconds(),
		},
		{
			name:          "colon format +02:00",
			input:         "+02:00",
			wantValid:     true,
			wantMicrosecs: 2 * time.Hour.Microseconds(),
		},
		{
			name:          "colon format -05:30",
			input:         "-05:30",
			wantValid:     true,
			wantMicrosecs: -(5*time.Hour + 30*time.Minute).Microseconds(),
		},
		{
			name:          "positive with minutes +0530",
			input:         "+0530",
			wantValid:     true,
			wantMicrosecs: (5*time.Hour + 30*time.Minute).Microseconds(),
		},
		{
			name:      "malformed - too short",
			input:     "+05",
			wantValid: false,
		},
		{
			name:      "malformed - no sign",
			input:     "0500",
			wantValid: false,
		},
		{
			name:      "malformed - letters",
			input:     "+ab:cd",
			wantValid: false,
		},
		{
			name:      "malformed - single char",
			input:     "X",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTimezoneOffset(tt.input)

			if result.Valid != tt.wantValid {
				t.Errorf("ParseTimezoneOffset(%q).Valid = %v, want %v", tt.input, result.Valid, tt.wantValid)
			}

			if tt.wantValid && result.Microseconds != tt.wantMicrosecs {
				t.Errorf("ParseTimezoneOffset(%q).Microseconds = %d, want %d", tt.input, result.Microseconds, tt.wantMicrosecs)
			}
		})
	}
}
