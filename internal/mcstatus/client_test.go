package mcstatus

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

	var requestedPath string
	var requestedQuery string
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestedPath = request.URL.Path
		requestedQuery = request.URL.Query().Get("query")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"online": true,
				"players": {"online": 3, "max": 20},
				"version": {"name_clean": "1.21.8"}
			}`)),
		}, nil
	})}
	client := NewClient(httpClient, "https://example.test/status/{address}")
	got, err := client.Fetch(context.Background(), "203.0.113.10")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if requestedPath != "/status/203.0.113.10" {
		t.Fatalf("requested path = %q", requestedPath)
	}
	if requestedQuery != "false" {
		t.Fatalf("query parameter = %q, want false", requestedQuery)
	}
	if !got.Online || got.PlayersOnline != 3 || got.PlayersMax != 20 || got.Version != "1.21.8" {
		t.Fatalf("Fetch() = %#v", got)
	}
}

func TestFetchOffline(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"online":false}`)),
		}, nil
	})}
	client := NewClient(httpClient, "https://example.test/{address}")
	got, err := client.Fetch(context.Background(), "203.0.113.10")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if got.Online {
		t.Fatal("Fetch().Online = true, want false")
	}
}
