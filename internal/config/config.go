package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"
)

// Config представляет основную конфигурацию приложения News Aggregator.
// Содержит настройки сервера, логгера, приложения и базы данных.
type Config struct {
	Server   ServerConfig   `json:"server"`
	Logger   LoggerConfig   `json:"logger"`
	App      AppConfig      `json:"app"`
	Database DatabaseConfig `json:"database"`
}

// ServerConfig содержит настройки HTTP-сервера приложения.
// Включает адрес и порт для прослушивания входящих соединений.
type ServerConfig struct {
	Address string `json:"address"`
}

// LoggerConfig содержит настройки системы логирования.
// Определяет уровень детализации логов (debug, info, warn, error).
type LoggerConfig struct {
	Level string `json:"level"`
}

// FeedURL представляет конфигурацию отдельной RSS-ленты.
// Содержит уникальное имя ленты и URL для загрузки контента.
type FeedURL struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// AppConfig содержит настройки бизнес-логики приложения.
// Включает лимиты новостей, список RSS-лент и интервалы обработки.
type AppConfig struct {
	DefaultNewsLimit   int       `json:"default_news_limit"`
	FeedURLs           []FeedURL `json:"feed_urls"`
	ProcessingInterval string    `json:"processing_interval"`
}

// DatabaseConfig содержит параметры подключения к PostgreSQL.
// Включает хост, порт, учетные данные и настройки SSL соединения.
type DatabaseConfig struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

// DSN возвращает строку подключения к PostgreSQL в формате URI.
// Формат: postgres://username:password@host:port/dbname?sslmode=mode
// Используется для установки соединения с базой данных через pgxpool.
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
		c.SSLMode)
}

// Load загружает конфигурацию из JSON-файла по указанному пути.
// Возвращает ошибку если файл не существует, недоступен для чтения
// или содержит некорректный JSON. Использует значения по умолчанию
// для незаданных полей конфигурации.
func Load(configPath string) (*Config, error) {
	cfg := New()
	fileData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}
	if err := json.Unmarshal(fileData, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from file %s: %w", configPath, err)
	}
	return cfg, nil
}

// New создает новый экземпляр Config с значениями по умолчанию.
func New() *Config {
	return &Config{
		Server: ServerConfig{
			Address: ":8080",
		},
		Logger: LoggerConfig{
			Level: "info",
		},
		App: AppConfig{
			DefaultNewsLimit:   10,
			ProcessingInterval: "3m",
			FeedURLs:           []FeedURL{},
		},
		Database: DatabaseConfig{
			Host:    "localhost",
			Port:    5432,
			SSLMode: "disable",
		},
	}
}

// Validate проверяет корректность конфигурации.
// Проверяет обязательные поля базы данных, корректность URL RSS-лент,
// валидность интервала обработки и другие критичные параметры.
// Возвращает ошибку с описанием первой найденной проблемы.
func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database host is not set")
	}
	if c.Database.Username == "" {
		return fmt.Errorf("database username is not set")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("database password is not set")
	}
	if c.App.DefaultNewsLimit <= 0 {
		return fmt.Errorf("app.default_news_limit must be a positive number")
	}
	if len(c.App.FeedURLs) == 0 {
		return fmt.Errorf("app_feed_urls must not be empty")
	}
	for _, feed := range c.App.FeedURLs {
		if _, err := url.ParseRequestURI(feed.URL); err != nil {
			return fmt.Errorf("invalid url in app.feed_urls: %s", feed.URL)
		}
		if feed.Name == "" {
			return fmt.Errorf("feed name cannot be empty for url: %s", feed.URL)
		}
	}
	if _, err := time.ParseDuration(c.App.ProcessingInterval); err != nil {
		return fmt.Errorf("invalid app.processing_interval: %w", err)
	}
	return nil
}
