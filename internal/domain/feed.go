package domain

import "time"

// Item представляет отдельную новость в RSS-ленте.
type Item struct {
	Title       string
	Link        string
	Description string
	PubDate     time.Time
}

// Feed представляет полную RSS-ленту с метаданными и списком новостей.
type Feed struct {
	Title       string
	Link        string
	Description string
	Items       []Item
}
