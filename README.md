# Go Price Tracker

Микросервисная система и Telegram-бот на Go для мониторинга и отслеживания цен на товары в интернет-магазинах. Сервис уведомляет пользователей при снижении стоимости товаров до целевого уровня и организует фоновый опрос цен через очередь задач на базе Apache Kafka.

---

##  Основные возможности

- **Telegram-бот:**
  - `/add <url> <цена>` — моментальное получение текущей цены через парсер и сохранение подписки.
  - `/list` — просмотр всех активных подписок с кликабельными ссылками, текущими и целевыми ценами.
  - Ограничение конкурентных запросов (Worker Pool с семафором) и безопасная HTML-разметка.
- **HTML-скрапер (`goquery`):**
  - Поддержка различных магазинов по доменам (`dns-shop.ru`, `citilink.ru`, `scrapeme.live` и др.).
  - Очистка и нормализация строк с ценами в точный тип `decimal.Decimal` (без ошибок округления `float64`).
- **Фоновый планировщик (`scheduler`):**
  - Поиск устаревших записей в PostgreSQL (`last_checked_at IS NULL OR last_checked_at < NOW() - interval`).
  - Батчевая выгрузка и публикация задач в топик Kafka (`scrape_jobs`) с партиционированием по домену.
- **Хранилище данных (PostgreSQL 17):**
  - Автоматическое применение миграций при старте (`golang-migrate`).
  - Быстрый поиск и отсутствие полных сканов таблиц благодаря индексам.
  - Идемпотентность запросов (`ON CONFLICT`).

---

## Архитектура системы

```mermaid
flowchart TD
    User([Пользователь]) <-->|Команды /add, /list| Bot[Telegram Bot Service]
    
    subgraph Storage Layer
        DB[(PostgreSQL 17)]
    end

    subgraph Message Queue
        Kafka[Apache Kafka]
        KafkaUI[Kafka UI :8085]
        KafkaUI -.-> Kafka
    end

    Bot <-->|Upsert / Select| DB
    Bot -->|Синхронный парсинг при добавлении| WebSites[Интернет-магазины]

    Scheduler[Scheduler Service] -->|Поиск устаревших цен| DB
    Scheduler -->|Публикация scrape_jobs| Kafka
```

---

## Стек технологий

* **Язык:** Go 1.26.1+
* **База данных:** PostgreSQL 17 (`sqlx`, драйвер `pgx/v5`)
* **Очередь сообщений:** Apache Kafka (KRaft режим) + `segmentio/kafka-go`
* **Скрапинг:** `PuerkitoBio/goquery`
* **Финансы / Вычисления:** `shopspring/decimal`
* **Конфигурация:** `cleanenv` + `.env`
* **Управление задачами:** Taskfile (`task`)
* **Контейнеризация:** Docker & Docker Compose (Multi-stage сборка на Alpine)

---

## 📁 Структура проекта

```text
.
├── cmd/
│   ├── bot/                 # Точка входа для Telegram-бота
│   └── scheduler/           # Точка входа для планировщика задач
├── internal/
│   ├── bot/                 # Логика Telegram-бота и обработка команд
│   ├── config/              # Загрузка и валидация конфигурации (.env / ENV)
│   ├── kafka/               # Продюсер и работа с Kafka
│   ├── models/              # DTO, события и сущности БД
│   ├── scheduler/           # Логика периодической выборки товаров
│   ├── scraper/             # Модуль парсинга цен с HTML-страниц
│   ├── service/             # Сервисный слой бизнес-логики (TrackerService)
│   └── storage/             # Репозиторий PostgreSQL и запуск миграций
├── migrations/              # SQL-миграции (Up / Down)
├── Dockerfile               # Multi-stage образ для Telegram-бота
├── Dockerfile.scheduler     # Multi-stage образ для планировщика
├── docker-compose.yaml      # Декларация всех сервисов (Postgres, Bot, Scheduler, Kafka, Kafka UI)
├── Taskfile.yml             # Удобные команды автоматизации (Task runner)
└── README.md
```

---

## ⚙️ Переменные окружения (`.env`)

Создайте файл `.env` в корневой директории:

```env
# Окружение (local / production)
APP_ENV=local
LOG_LEVEL=debug

# Telegram
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrSTUvwxYZ

# База данных PostgreSQL
POSTGRES_USER=tracker_user
POSTGRES_PASSWORD=tracker_password
POSTGRES_DB=price_tracker
DATABASE_URL=postgres://tracker_user:tracker_password@postgres:5432/price_tracker?sslmode=disable

# Apache Kafka
KAFKA_BROKERS=kafka:9092
```

---

## 🚦 Быстрый старт

### Вариант 1: Запуск через Docker Compose (Рекомендуется)

1. Убедитесь, что установлены **Docker** и **Docker Compose**.
2. Заполните `.env` (укажите ваш токен Telegram-бота).
3. Запустите стек:

```bash
docker compose up -d --build
```

Или с использованием **Taskfile**:
```bash
task up:build
```

### Доступные веб-интерфейсы:
* **Kafka UI:** [http://localhost:8085](http://localhost:8085) — мониторинг топиков, сообщений и консьюмеров.

---

### Вариант 2: Локальная разработка

1. Запустите инфраструктуру (Postgres и Kafka):
   ```bash
   docker compose up -d postgres kafka kafka-ui
   ```
2. В `.env` укажите локальные адреса:
   ```env
   DATABASE_URL=postgres://tracker_user:tracker_password@localhost:5432/price_tracker?sslmode=disable
   KAFKA_BROKERS=localhost:9092
   ```
3. Запустите бота:
   ```bash
   go run ./cmd/bot
   ```
4. Запустите планировщик:
   ```bash
   go run ./cmd/scheduler
   ```

---

## 📋 Команды Taskfile

Если у вас установлен [Task](https://taskfile.dev/):

| Команда | Описание |
|---|---|
| `task up` | Запустить сервисы в фоне |
| `task up:build` | Пересобрать и перезапустить все сервисы |
| `task down` | Остановить контейнеры |
| `task down:v` | Остановить контейнеры и очистить Volume БД |
| `task logs` | Просмотр логов всех сервисов в реальном времени |
| `task logs:bot` | Просмотр логов бота |
| `task db:psql` | Открыть консоль `psql` внутри контейнера Postgres |
| `task build` | Скомпилировать бинарник бота в `bin/bot` |
| `task vet` | Запустить статический анализ `go vet` |
| `task test` | Запустить тесты |
| `task tidy` | Обновить и синхронизировать `go.mod` / `go.sum` |

---

## 🤖 Использование Telegram-бота

1. Откройте чат с ботом и отправьте `/start`.
2. Добавьте товар на отслеживание:
   ```text
   /add https://citilink.ru/product/example-item-12345 49990.00
   ```
3. Посмотрите ваши текущие подписки:
   ```text
   /list
   ```

---

## 📄 Лицензия

Проект распространяется под лицензией [MIT](LICENSE).