# Kubometr

AI-консультант строительного магазина, разработанный на Go.

Kubometr — бэкенд-сервис для строительного магазина: принимает сообщения через мессенджер (Telegram, с миграцией на MAX Messenger), с помощью AI помогает подбирать материалы и консультирует по вопросам ремонта. История диалогов и пользователи хранятся в PostgreSQL.

Проект развивается как коммерческий backend с постепенным добавлением REST API, Docker и интеграции с 1С:УНФ (каталог товаров, цены, остатки, заказы).

## Возможности

- Консультации через Telegram (миграция на MAX Messenger — в процессе)
- Интеграция с AI через OpenRouter (легко заменяемый провайдер)
- История диалогов в PostgreSQL
- Ограничение количества и параллельности AI-запросов
- Ограничение длины сообщений
- Таймауты запросов к AI
- Управление состоянием пользователя
- Разделение транспортного слоя и бизнес-логики (Service Layer, Repository Pattern)

## Технологии

- Go
- Telegram Bot API / MAX Bot API (webhook)
- OpenRouter API
- PostgreSQL (pgx/v5)
- SQL-миграции
- slog
- Docker *(в разработке)*

## Архитектура

```text
cmd/
├── bot/
└── maxtest/

internal/
├── ai/
├── app/
├── config/
├── consultation/
├── database/
├── history/
├── logger/
├── max/
├── onec/
├── state/
├── telegram/
└── users/

migrations/
```

### Основные компоненты

| Пакет | Назначение |
|--------|------------|
| `telegram` | Работа с Telegram Bot API |
| `max` | Приём вебхуков MAX Messenger |
| `consultation` | Бизнес-логика консультаций |
| `ai` | Клиент AI-провайдера (OpenRouter) |
| `database` | Подключение к PostgreSQL |
| `history` | Хранение истории диалогов |
| `users` | Хранение пользователей |
| `state` | Управление состоянием пользователей |
| `config` | Загрузка конфигурации |
| `logger` | Настройка логирования |
| `app` | Сборка зависимостей и запуск приложения |

## Запуск проекта

### 1. Клонировать репозиторий

```bash
git clone https://github.com/mp3dotexe/kubometr.git
cd kubometr
```

### 2. Создать файл `.env`

Используйте `.env.example` как шаблон.

```env
BOT_TOKEN=

OPENROUTER_API_KEY=
AI_MODEL=

AI_TIMEOUT=30s
AI_RATE_LIMIT=5s
MAX_PROMPT_LENGTH=2000
MAX_CONCURRENT_AI=5

POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=
POSTGRES_PASSWORD=
POSTGRES_DB=
```

### 3. Применить миграции

На данный момент миграции применяются вручную — выполните SQL-файлы
из папки `migrations/` (например, через `psql` или клиент вроде DBeaver)
в порядке возрастания номера. Переход на `golang-migrate` — в планах.

### 4. Запустить приложение

```bash
go run ./cmd/bot
```

## Текущее состояние

### Реализовано

- Telegram Bot
- AI-консультации через OpenRouter
- Service Layer, Repository Pattern
- История диалогов в PostgreSQL (миграции, users + messages)
- Конфигурация через `.env`
- Логирование
- Управление состоянием пользователя
- Ограничение количества и параллельности AI-запросов
- Приём и валидация вебхуков MAX (HTTP-хендлер, локально протестирован)

### В разработке

- Полная интеграция MAX Messenger (регистрация бота, боевой токен)
- Интеграция с 1С:УНФ (каталог, цены, остатки, заказы)
- Docker / Docker Compose
- REST API
- Unit-тесты

## Планы развития

- Завершить миграцию на MAX Messenger
- Интеграция 1С:УНФ
- Контекстные ответы AI на основе истории
- Docker и Docker Compose
- REST API + Swagger
- Развёртывание на сервере

## Автор

Roman