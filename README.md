# Newsaggregator

Newsaggregator - Сервис на Go для парсинга RSS источников, хранения и распространения новостей.

## Структура проекта

```
NEWSAGGREGATOR/
├── cmd/
│   └── news/
│       └── main.go                 # Точка входа приложения
├── internal/
│   ├── adapter/
│   │   ├── fetcher/               # Адаптеры для получения данных
│   │   └── parser/                # Адаптеры для парсинга данных
│   ├── app/
│   │   └── app.go                 # Инициализация и сборка приложения
│   ├── config/
│   │   └── config.go              # Конфигурация приложения
│   ├── domain/
│   │   └── feed.go                # Доменные модели (сущности)
│   ├── logger/
│   │   └── logger.go              # Логирование
│   ├── migrations/
│   │   └── migrations.go          # Миграции базы данных
│   ├── transport/
│   │   └── http/
│   │       ├── handler.go         # HTTP обработчики
│   │       ├── middleware.go      # HTTP middleware
│   │       └── server.go          # HTTP сервер
│   ├── usecase/
│   │   ├── feedprocessing.go      # Use case обработки фидов
│   │   ├── fetchfeed.go           # Use case получения фидов
│   │   └── newsgetter.go          # Use case получения новостей
│   ├── worker/
│   │   └── worker.go              # Фоновые workers
│   └── storage/
│       ├── interface.go           # Интерфейсы хранилища
│       └── postgres.go            # Реализация Postgres хранилища
├── web/
│   └── static/
│       └── index.html             # Статическая веб-страница
├── config.json                    # Конфигурация приложения
├── go.mod                         # Модули Go
├── go.sum                         # Зависимости Go
└── README.md                      # Документация проекта
```

## Описание основных компонентов

cmd/news - Основное приложение

internal/adapter - Адаптеры для внешних сервисов и парсеров

internal/app - Сборка зависимостей приложения

internal/config - Конфигурация и переменные окружения

internal/domain - Бизнес-сущности и модели

internal/logger - Система логирования

internal/migrations - Управление миграциями БД

internal/transport/http - HTTP слой (роутеры, middleware, handlers)

internal/usecase - Бизнес-логика приложения

internal/worker - Фоновые задачи и workers

internal/storage - Слой работы с данными (репозитории)

web/static - Статические файлы фронтенда

config.json - Основной конфигурационный файл приложения

## Разработка

### Локальный запуск для разработки
```bash
go run cmd/news/main.go
```

## Технологии

- **Go** - основной язык разработки
- **PostgreSQL** - основная база данных
- **REST API** - коммуникация между сервисами




