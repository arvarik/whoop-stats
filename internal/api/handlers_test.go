package api

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name     string
		limit    string
		expected int32
	}{
		{"Empty limit", "", 50},
		{"Valid limit", "10", 10},
		{"Valid limit max", "200", 200},
		{"Limit too high", "201", 200},
		{"Limit too low", "0", 50},
		{"Negative limit", "-5", 50},
		{"Invalid limit string", "abc", 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?limit="+tt.limit, nil)
			if tt.limit == "" {
				req = httptest.NewRequest("GET", "/", nil)
			}
			got := parseLimit(req)
			if got != tt.expected {
				t.Errorf("parseLimit() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseCursor(t *testing.T) {
	now := time.Now().Truncate(time.Nanosecond)
	validTimeStr := now.Format(time.RFC3339Nano)

	tests := []struct {
		name    string
		cursor  string
		wantErr bool
	}{
		{"Valid cursor", validTimeStr, false},
		{"Empty cursor", "", false},
		{"Invalid cursor", "not-a-date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?cursor="+tt.cursor, nil)
			if tt.cursor == "" {
				req = httptest.NewRequest("GET", "/", nil)
			}
			got, err := parseCursor(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCursor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.cursor == "" {
					// For empty cursor, it should be close to now
					if time.Since(got.Time) > time.Second {
						t.Errorf("parseCursor() returned time too far in the past: %v", got.Time)
					}
				} else {
					// Use Equal for time comparison to handle monotonic clock etc
					expected, _ := time.Parse(time.RFC3339Nano, tt.cursor)
					if !got.Time.Equal(expected) {
						t.Errorf("parseCursor() = %v, want %v", got.Time, expected)
					}
				}
				if !got.Valid {
					t.Errorf("parseCursor() returned invalid pgtype.Timestamptz")
				}
			}
		})
	}
}
