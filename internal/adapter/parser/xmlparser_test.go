package parser

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXMLParser_Parse_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := NewXMLParser(logger)

	xmlData := `
	<rss>
	<channel>
	<title>Test Feed</title>
	<link>https://example.com</link>
	<description>Test Description</description>
	<item>
	<title>Item 1</title>
	<link>https://example.com/item1</link>
	<description>Item 1 Description</description>
	<pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
	</item>
	<item>
	<title>Item 2</title>
	<link>https://example.com/item2</link>
	<description>Item 2 Description</description>
	<pubDate>Tue, 03 Jan 2006 12:00:00 GMT</pubDate>
	</item>
	</channel>
	</rss>`

	ctx := context.Background()
	feed, err := parser.Parse(ctx, strings.NewReader(xmlData))

	require.NoError(t, err)
	require.NotNil(t, feed)

	assert.Equal(t, "Test Feed", feed.Title)
	assert.Equal(t, "https://example.com", feed.Link)
	assert.Equal(t, "Test Description", feed.Description)
	assert.Len(t, feed.Items, 2)

	assert.Equal(t, "Item 1", feed.Items[0].Title)
	assert.Equal(t, "https://example.com/item1", feed.Items[0].Link)
	assert.Equal(t, "Item 1 Description", feed.Items[0].Description)
	assert.WithinDuration(t, time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC), feed.Items[0].PubDate, time.Second)

	assert.Equal(t, "Item 2", feed.Items[1].Title)
	assert.Equal(t, "https://example.com/item2", feed.Items[1].Link)
	assert.Equal(t, "Item 2 Description", feed.Items[1].Description)
	assert.WithinDuration(t, time.Date(2006, 1, 3, 12, 0, 0, 0, time.UTC), feed.Items[1].PubDate, time.Second)
}
func TestXMLParser_Parse_InvalidXML(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := NewXMLParser(logger)
	invalidXML := `
	<rss>
	<channel>
	<title>Test Feed</title>
	<invalid-tag>
	</channel>
	</rss>`
	ctx := context.Background()
	feed, err := parser.Parse(ctx, strings.NewReader(invalidXML))

	assert.Error(t, err)
	assert.Nil(t, feed)
	assert.Contains(t, err.Error(), "failed to decode XML")
}
func TestXMLParser_Parse_ContextCancelled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := NewXMLParser(logger)
	xmlData := `
<rss>
<channel>
<title>Test Feed</title>
<link>https://example.com</link>
<description>Test Description</description>
</channel>
</rss>`
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	feed, err := parser.Parse(ctx, strings.NewReader(xmlData))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Nil(t, feed)
}
func TestXMLParser_Parse_EmptyFeed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := NewXMLParser(logger)
	xmlData := `
	<rss>
	<channel>
	<title>Empty Feed</title>
	<link>https://example.com</link>
	<description>Empty Description</description>
	</channel>
	</rss>`
	ctx := context.Background()
	feed, err := parser.Parse(ctx, strings.NewReader(xmlData))

	require.NoError(t, err)
	require.NotNil(t, feed)

	assert.Equal(t, "Empty Feed", feed.Title)
	assert.Equal(t, "https://example.com", feed.Link)
	assert.Equal(t, "Empty Description", feed.Description)
	assert.Empty(t, feed.Items)
}
