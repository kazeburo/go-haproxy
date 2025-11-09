package haproxy

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseCSV(t *testing.T) {
	tests := []struct {
		name      string
		csvData   string
		expectErr bool
		expectLen int
	}{
		{
			name: "Valid CSV",
			csvData: `# pxname,svname,stot,rate,bck,status,scur
example,FRONTEND,1836,2,0,OPEN,1
example-backend,198.51.100.196:443,53,0,1,UP,0`,
			expectErr: false,
			expectLen: 2,
		},
		{
			name: "Invalid CSV",
			csvData: `# pxname,svname,stot,rate,bck,status,scur
example,FRONTEND,1836,2,0,OPEN`,
			expectErr: true,
			expectLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &http.Client{
				Transport: roundTripFunc(func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(tt.csvData)),
					}
				}),
			}
			stats, err := Status(HTTPClient(mockClient))
			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}
			if len(stats) != tt.expectLen {
				t.Fatalf("expected length: %d, got: %d", tt.expectLen, len(stats))
			}
		})
	}
}

func TestStatus(t *testing.T) {
	mockData := `# pxname,svname,stot,rate,bck,status,scur
example,FRONTEND,1836,2,0,OPEN,1`
	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(mockData)),
			}
		}),
	}
	stats, err := Status(HTTPClient(mockClient))
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if len(stats) != 1 || stats[0].Pxname != "example" {
		t.Fatalf("unexpected Status result: %+v", stats)
	}
}

func TestMapToStats_TypeField(t *testing.T) {
	cases := []struct {
		inputType string
		expected  string
	}{
		{"0", "FRONTEND"},
		{"1", "BACKEND"},
		{"2", "SERVER"},
		{"3", "LISTENER"},
		{"unknown", "UNKNOWN"},
	}

	for _, c := range cases {
		input := map[string]string{
			"pxname": "example",
			"svname": "test",
			"type":   c.inputType,
		}
		result := mapToStats(input)
		if result.Type != c.expected {
			t.Errorf("expected Type %q, got %q", c.expected, result.Type)
		}
	}
}

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}
