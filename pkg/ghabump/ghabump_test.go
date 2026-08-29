package ghabump

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type stubTransport struct {
	tag string
}

func (s stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	loc := fmt.Sprintf("https://github.com/owner/repo/releases/tag/%s", s.tag)
	return &http.Response{
		StatusCode: http.StatusFound,
		Status:     "302 Found",
		Proto:      "HTTP/1.1",
		Header:     http.Header{"Location": []string{loc}},
		Body:       http.NoBody,
	}, nil
}

func stubClient(latestTag string) *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: stubTransport{tag: latestTag},
	}
}

func Test_SuggestUpgrade(t *testing.T) {
	tests := []struct {
		name     string
		uses     string
		latest   string
		expected string
	}{
		{
			name:     "floating major tag bumps on major version",
			uses:     "actions/checkout@v3",
			latest:   "v4.2.2",
			expected: "v4",
		},
		{
			name:     "floating major tag stays put on minor release",
			uses:     "actions/checkout@v4",
			latest:   "v4.3.0",
			expected: "",
		},
		{
			name:     "exact pin with v prefix stays put within major",
			uses:     "actions/checkout@v1.2.3",
			latest:   "v1.9.0",
			expected: "",
		},
		{
			name:     "exact pin with v prefix bumps to floating major",
			uses:     "actions/checkout@v1.2.3",
			latest:   "v2.0.0",
			expected: "v2",
		},
		{
			name:     "v-less exact pin bumps to latest full tag",
			uses:     "alexellis/upload-assets@0.4.1",
			latest:   "0.5.0",
			expected: "0.5.0",
		},
		{
			name:     "v-less exact pin bumps on patch release",
			uses:     "alexellis/upload-assets@0.5.0",
			latest:   "0.5.1",
			expected: "0.5.1",
		},
		{
			name:     "v-less exact pin already at latest",
			uses:     "alexellis/upload-assets@0.5.0",
			latest:   "0.5.0",
			expected: "",
		},
		{
			name:     "branch pin master is skipped",
			uses:     "actions/checkout@master",
			latest:   "v4.2.2",
			expected: "",
		},
		{
			name:     "branch pin main is skipped",
			uses:     "actions/checkout@main",
			latest:   "v4.2.2",
			expected: "",
		},
		{
			name:     "SHA pin is skipped",
			uses:     "actions/checkout@a1234567890abcdef1234567890abcdef1234567",
			latest:   "v4.2.2",
			expected: "",
		},
		{
			name:     "local composite action is skipped",
			uses:     "./.github/actions/test@v1",
			latest:   "v2.0.0",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := suggestUpgrade(stubClient(tc.latest), tc.uses)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Fatalf("expected %q got %q", tc.expected, got)
			}
		})
	}
}

func Test_ApplyReplacements(t *testing.T) {
	workflow := `name: build
on: push
jobs:
  build:
    steps:
      - uses: alexellis/upload-assets@0.4.1
      - uses: actions/checkout@v3
`

	replacements := map[string]string{
		"alexellis/upload-assets@0.4.1": "0.5.0",
		"actions/checkout@v3":           "v4",
	}

	updated := ApplyReplacements([]byte(workflow), replacements)

	if !strings.Contains(updated, "alexellis/upload-assets@0.5.0") {
		t.Fatalf("expected upload-assets to be bumped, got:\n%s", updated)
	}
	if !strings.Contains(updated, "actions/checkout@v4") {
		t.Fatalf("expected checkout to be bumped, got:\n%s", updated)
	}
}
