package fetcher

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPFetcher_Fetch_Succsess(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response data"))
	}))
	defer testServer.Close()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fetcher := NewHTTPFetcher(logger)

	ctx := context.Background()
	reader, err := fetcher.Fetch(ctx, testServer.URL)

	require.NoError(t, err)
	defer reader.Close()
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "test response data", string(data))
}
func TestHTTPFetcher_Fetch_NotFound(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer testServer.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fetcher := NewHTTPFetcher(logger)

	ctx := context.Background()
	reader, err := fetcher.Fetch(ctx, testServer.URL)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 404")
	assert.Nil(t, reader)
}
func TestHTTPFetcher_InvalidURL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fetcher := NewHTTPFetcher(logger)

	ctx := context.Background()
	reader, err := fetcher.Fetch(ctx, "invalid://url")

	assert.Error(t, err)
	assert.Nil(t, reader)
}
func TestHTTPFecher_ContextCancelled(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("slow response"))
	}))
	defer testServer.Close()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fetcher := NewHTTPFetcher(logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	reader, err := fetcher.Fetch(ctx, testServer.URL)

	assert.Error(t, err)
	assert.Nil(t, reader)
}
