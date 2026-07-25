package publicip

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestFetch(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ip":"203.0.113.10"}`)),
		}, nil
	})}
	client := NewClient(httpClient, "https://example.test/ip")
	got, err := client.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if got != "203.0.113.10" {
		t.Fatalf("Fetch() = %q, want %q", got, "203.0.113.10")
	}
}

func TestFetchRejectsInvalidIP(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ip":"not-an-ip"}`)),
		}, nil
	})}
	client := NewClient(httpClient, "https://example.test/ip")
	if _, err := client.Fetch(context.Background()); err == nil {
		t.Fatal("Fetch() error = nil, want an error")
	}
}
