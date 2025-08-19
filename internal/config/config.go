package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Logger   LoggerConfig   `json:"logger"`
	App      AppConfig      `json:"app"`
	Database DatabaseConfig `json:"database"`
}
type ServerConfig struct {
	Address string `json:"address"`
}
type LoggerConfig struct {
	Level string `json:"level"`
}
type FeedURL struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}
type AppConfig struct {
	DefaultNewsLimit   int       `json:"default_news_limit"`
	FeedURLs           []FeedURL `json:"feed_urls"`
	ProcessingInterval string    `json:"processing_interval"`
}
type DatabaseConfig struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
		c.SSLMode)
}
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
