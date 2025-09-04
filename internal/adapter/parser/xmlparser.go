package parser

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"news/internal/domain"
	"strings"
	"time"
)

// rssXML представляет структуру RSS-ленты в XML формате.
// Используется для декодирования XML данных в Go структуры.
type rssXML struct {
	Channel channelXML `xml:"channel"`
}

// channelXML представляет канал RSS-ленты с заголовком, ссылкой, описанием и элементами.
type channelXML struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []itemXML `xml:"item"`
}

// itemXML представляет отдельный элемент (новость) в RSS-ленте.
// Содержит заголовок, ссылку, описание и дату публикации.
type itemXML struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// XMLParser реализует парсер RSS-лент в XML формате.
// Обрабатывает различные форматы дат и обеспечивает отказоустойчивость при парсинге.
type XMLParser struct {
	log *slog.Logger
}

// NewXMLParser создает новый экземпляр XMLParser для обработки RSS-лент.
// Принимает логгер для записи событий парсинга и ошибок.
func NewXMLParser(log *slog.Logger) *XMLParser {
	return &XMLParser{
		log: log,
	}
}

// Parse преобразует XML данные RSS-ленты в доменную модель Feed.
// Обрабатывает контекст для отмены операции, парсит элементы ленты,
// конвертирует даты из различных форматов и фильтрует некорректные элементы.
// Возвращает ошибку при проблемах с декодированием XML или форматом данных.
func (p *XMLParser) Parse(ctx context.Context, reader io.Reader) (*domain.Feed, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var rss rssXML
	decoder := xml.NewDecoder(reader)
	if err := decoder.Decode(&rss); err != nil {
		p.log.Error(
			"Error decoding XML",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to decode XML: %w", err)
	}
	feed := domain.Feed{
		Title:       rss.Channel.Title,
		Link:        rss.Channel.Link,
		Description: rss.Channel.Description,
		Items:       make([]domain.Item, 0, len(rss.Channel.Items)),
	}
	for _, itemDTO := range rss.Channel.Items {
		pubDate, err := parsePubDate(itemDTO.PubDate)
		if err != nil {
			p.log.Warn(
				"could not parse item pubDate, skipping item",
				slog.String("pubDate", itemDTO.PubDate),
				slog.String("item_title", itemDTO.Title),
				slog.Any("error", err),
			)
			continue
		}
		item := domain.Item{
			Title:       itemDTO.Title,
			Link:        itemDTO.Link,
			Description: itemDTO.Description,
			PubDate:     pubDate,
		}
		feed.Items = append(feed.Items, item)
	}
	return &feed, nil
}

// parsePubDate преобразует строку даты из RSS в объект time.Time.
// Поддерживает multiple форматы дат, включая RFC1123, RFC822 и другие распространенные варианты.
// Возвращает ошибку если ни один из форматов не подходит для парсинга.
func parsePubDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"Mon, 2 Jan 2006 15:04:05 -0700",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, strings.TrimSpace(dateStr)); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not parse date in any known format: %q", dateStr)
}
