# Knowledge Service

Python-микросервис для индексации документов и ответов на вопросы по их содержимому (RAG — Retrieval-Augmented Generation). Является аналогом движка NotebookLM в составе системы sihkaromicro.

---

## Содержание

1. [Обзор](#1-обзор)
2. [Архитектура](#2-архитектура)
3. [Функционал](#3-функционал)
4. [Поддерживаемые типы источников](#4-поддерживаемые-типы-источников)
5. [gRPC API](#5-grpc-api)
6. [Kafka-интеграция](#6-kafka-интеграция)
7. [База данных](#7-база-данных)
8. [Безопасность](#8-безопасность)
9. [Конфигурация](#9-конфигурация)
10. [Зависимости](#10-зависимости)
11. [Запуск](#11-запуск)
12. [Структура проекта](#12-структура-проекта)

---

## 1. Обзор

Knowledge Service решает две задачи:

| Задача | Триггер | Описание |
|---|---|---|
| **Индексация** | Kafka `jobs.chunking` | Скачивает файл из MinIO, парсит, разбивает на чанки, генерирует эмбеддинги, сохраняет в pgvector |
| **Ответы (RAG)** | gRPC-запрос | Ищет релевантные чанки, реранкирует, генерирует ответ через OpenAI |

Сервис работает внутри Docker-сети и недоступен снаружи напрямую — вызывается другими микросервисами через gRPC.

---

## 2. Архитектура

```
                         ┌──────────────────────────────────────┐
                         │         Knowledge Service            │
                         │                                      │
  Kafka ─────────────────►  ChunkingConsumer                    │
  jobs.chunking          │    │                                 │
  jobs.cancel            │    ├─ parsers.py   (unstructured)    │
                         │    ├─ pipeline.py  (SemanticChunker) │
                         │    ├─ openai_provider.py (embeddings)│
                         │    └─ vector_store.py (asyncpg)      │
                         │                         │            │
  Kafka ◄────────────────│  StatusProducer         │            │
  jobs.status            │                         ▼            │
                         │                    PostgreSQL        │
  gRPC :50054 ───────────►  RAGServicer       + pgvector        │
  Search                 │    │                    ▲            │
  Generate               │    ├─ search.py    ─────┘            │
  GenerateStream         │    ├─ cross_encoder.py (rerank)      │
                         │    └─ rag.py       (OpenAI GPT)      │
                         └──────────────────────────────────────┘
                                   │              │
                                MinIO          Keycloak
                           (скачивание      (JWKS верификация
                             файлов)          JWT токенов)
```

Оба компонента — gRPC-сервер и Kafka consumer — работают в одном asyncio event loop параллельно.

---

## 3. Функционал

### 3.1 Индексация источника (Chunking Pipeline)

```
MinIO download
    │
    ▼
unstructured.partition          ← Document-Aware парсинг с учётом структуры
    │
    ▼
chunk_by_title                  ← Parent chunks: группировка по заголовкам секций
    │                              (~6000 символов / ~1500 токенов)
    ▼
SemanticChunker (LangChain)     ← Child chunks: семантическое разбиение внутри секции
    │                              (~200–500 токенов)
    ▼
OpenAI text-embedding-3-small   ← Батч-эмбеддинг child чанков (1536-мерные векторы)
    │
    ▼
PostgreSQL + pgvector            ← Сохранение parent_chunks + chunks в одной транзакции
    │
    ▼
Kafka jobs.status: "done"/"failed"
```

**Retry-логика:** до 3 попыток с экспоненциальным backoff (2s, 4s). Kafka offset не коммитится до успешного завершения.

**Файлы > 500 MB** стримятся во временный файл на диске; файлы меньше загружаются в память (`io.BytesIO`).

### 3.2 RAG (Retrieval-Augmented Generation)

```
Запрос пользователя
    │
    ▼
OpenAI text-embedding-3-small   ← Эмбеддинг запроса
    │
    ▼
pgvector ANN top-20             ← Поиск ближайших child чанков
    │                              WHERE owner_id = ? AND project_id = ?
    ▼
CrossEncoder rerank top-5       ← cross-encoder/ms-marco-MiniLM-L-6-v2
    │                              Реранкинг parent чанков
    ▼
OpenAI GPT (chat completion)    ← Генерация ответа с контекстом
    │
    ▼
Ответ + ссылки на источники
```

### 3.3 Удаление источника

При получении сообщения из `jobs.cancel` сервис удаляет все `parent_chunks` источника. Связанные `chunks` удаляются каскадно через `ON DELETE CASCADE`.

---

## 4. Поддерживаемые типы источников

| Тип (`file_type`) | Формат | Парсер |
|---|---|---|
| `pdf` | PDF-документы | `unstructured` + Tesseract OCR (hi_res) |
| `docx` | Microsoft Word | `unstructured` + python-docx |
| `txt` | Plain text | `unstructured` |
| `md` / `markdown` | Markdown | `unstructured` |
| `url` | Веб-страница | `httpx` fetch + `unstructured.partition_html` |

> Для типа `url` файл не скачивается из MinIO. Сервис сам скачивает страницу по `source_url` из Kafka-сообщения.

---

## 5. gRPC API

**Порт:** `50054`  
**Proto-файл:** [`proto/rag.proto`](proto/rag.proto)

### 5.1 Методы

#### `Search` — семантический поиск

Возвращает релевантные фрагменты документов без генерации ответа.

```protobuf
rpc Search(SearchRequest) returns (SearchResponse);

message SearchRequest {
  string query      = 1;
  string owner_id   = 2;  // игнорируется — берётся из JWT
  int64  project_id = 3;
  int32  top_k      = 4;  // по умолчанию 20
}

message SearchResponse {
  repeated SearchResult results = 1;
}

message SearchResult {
  string content   = 1;  // текст parent chunk
  float  score     = 2;  // rerank score (CrossEncoder)
  int64  source_id = 3;
  string metadata  = 4;  // JSON-строка
}
```

#### `Generate` — генерация ответа

Выполняет полный RAG-цикл: поиск → реранк → генерация. Возвращает ответ и список источников.

```protobuf
rpc Generate(GenerateRequest) returns (GenerateResponse);

message GenerateRequest {
  string query      = 1;
  string owner_id   = 2;  // игнорируется — берётся из JWT
  int64  project_id = 3;
}

message GenerateResponse {
  string                answer  = 1;
  repeated SearchResult sources = 2;
}
```

#### `GenerateStream` — стриминг ответа

Тот же RAG-цикл, но ответ стримится чанками по мере генерации OpenAI.

```protobuf
rpc GenerateStream(GenerateRequest) returns (stream GenerateChunk);

message GenerateChunk {
  string text = 1;   // фрагмент текста ответа
  bool   done = 2;   // true в последнем сообщении
}
```

### 5.2 Аутентификация

Все методы требуют JWT токен Keycloak в метаданных gRPC-запроса:

```
authorization: Bearer <jwt_token>
```

`owner_id` (Keycloak `sub`) извлекается из токена сервером — поле `owner_id` в proto-сообщениях игнорируется.

---

## 6. Kafka-интеграция

### 6.1 Консьюмер

**Consumer group:** `knowledge-service`  
**Auto commit:** отключён (ручной commit после успешной обработки)

| Топик | Назначение | Формат сообщения |
|---|---|---|
| `jobs.chunking` | Запуск индексации источника | см. ниже |
| `jobs.cancel` | Удаление чанков источника | `{"job_id":"...", "owner_id":"...", "source_id":123}` |

**Формат `jobs.chunking`:**
```json
{
  "job_id":    "uuid",
  "type":      "chunking",
  "owner_id":  "keycloak-user-id",
  "source_id": 123,
  "minio_path":"owner_id/source_id/filename.pdf",
  "file_type": "pdf",
  "source_url":""
}
```

> Для `file_type: "url"` поле `minio_path` пустое, `source_url` содержит URL страницы.

### 6.2 Продюсер

| Топик | Назначение | Формат сообщения |
|---|---|---|
| `jobs.status` | Результат обработки | `{"job_id":"...", "source_id":123, "status":"done\|failed", "error":""}` |

---

## 7. База данных

**PostgreSQL 16** с расширением **pgvector**.  
Подключение через `asyncpg` с пулом соединений.

### Схема

```sql
-- Секции документа (parent chunks)
CREATE TABLE parent_chunks (
    id         BIGSERIAL PRIMARY KEY,
    source_id  BIGINT NOT NULL,
    project_id BIGINT NOT NULL,
    owner_id   TEXT   NOT NULL,
    content    TEXT   NOT NULL,
    metadata   JSONB,
    position   INT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Семантические подчанки с векторами (child chunks)
CREATE TABLE chunks (
    id              BIGSERIAL PRIMARY KEY,
    parent_chunk_id BIGINT REFERENCES parent_chunks(id) ON DELETE CASCADE,
    source_id       BIGINT NOT NULL,
    project_id      BIGINT NOT NULL,
    owner_id        TEXT   NOT NULL,
    content         TEXT   NOT NULL,
    embedding       vector(1536),
    metadata        JSONB,
    position        INT,
    created_at      TIMESTAMP DEFAULT NOW()
);
```

### Индексы

| Индекс | Тип | Назначение |
|---|---|---|
| `chunks_embedding_idx` | `ivfflat (lists=100)` | ANN-поиск по косинусному расстоянию |
| `idx_chunks_owner_project` | B-Tree | Фильтр по `owner_id, project_id` |
| `idx_parent_chunks_owner_project` | B-Tree | Фильтр по `owner_id, project_id` |
| `idx_chunks_source_id` | B-Tree | Быстрое удаление по источнику |
| `idx_parent_chunks_source_id` | B-Tree | Быстрое удаление по источнику |

### Связи между чанками

```
parent_chunks (1)
    └── chunks (N)   ← ON DELETE CASCADE
```

При удалении `parent_chunk` все связанные `chunks` удаляются автоматически.

---

## 8. Безопасность

### Изоляция данных

- Все SQL-запросы к pgvector содержат обязательный фильтр `WHERE owner_id = $1 AND project_id = $2`
- `owner_id` извлекается **исключительно** из верифицированного JWT токена (поле `sub`)
- Поле `owner_id` в proto-сообщениях существует для документации, но в handlers **игнорируется**

### Верификация JWT

Сервис верифицирует токены самостоятельно без обращения к Auth Service:

1. Загружает JWKS с `{KEYCLOAK_URL}/realms/{REALM}/protocol/openid-connect/certs`
2. Верифицирует подпись RS256
3. Извлекает `sub` как `owner_id`

JWKS кэшируется в памяти на 10 минут (TTL обновляется автоматически).

---

## 9. Конфигурация

Все параметры задаются переменными окружения (или через `.env`).

| Переменная | По умолчанию | Описание |
|---|---|---|
| `OPENAI_API_KEY` | **обязательно** | Ключ OpenAI (эмбеддинги + генерация) |
| `DB_DSN` | `postgresql://rag:rag@rag-db:5432/rag` | Строка подключения к PostgreSQL |
| `KAFKA_BROKERS` | `kafka:29092` | Адрес Kafka |
| `KAFKA_CHUNKING_TOPIC` | `jobs.chunking` | Топик входящих задач |
| `KAFKA_CANCEL_TOPIC` | `jobs.cancel` | Топик отмены |
| `KAFKA_STATUS_TOPIC` | `jobs.status` | Топик статусов |
| `KAFKA_GROUP_ID` | `knowledge-service` | Consumer group |
| `MINIO_ENDPOINT` | `minio:9000` | Адрес MinIO |
| `MINIO_ACCESS_KEY` | `minioadmin` | Ключ доступа MinIO |
| `MINIO_SECRET_KEY` | `minioadmin123` | Секретный ключ MinIO |
| `MINIO_BUCKET` | `sources` | Бакет с файлами |
| `KEYCLOAK_URL` | `http://keycloak:8082` | URL Keycloak |
| `KEYCLOAK_REALM` | `Clients` | Realm Keycloak |
| `GRPC_PORT` | `50054` | Порт gRPC-сервера |
| `OPENAI_CHAT_MODEL` | `gpt-4o-mini` | Модель для генерации ответов |
| `EMBEDDING_MODEL` | `text-embedding-3-small` | Модель эмбеддингов |
| `EMBEDDING_DIMENSIONS` | `1536` | Размерность векторов |
| `PARENT_CHUNK_MAX_CHARS` | `6000` | Макс. символов в parent chunk |
| `MAX_IN_MEMORY_BYTES` | `524288000` | Порог in-memory загрузки (500 MB) |
| `MAX_RETRIES` | `3` | Попыток обработки одного сообщения |

---

## 10. Зависимости

### Внешние сервисы

| Сервис | Назначение | Обязательный |
|---|---|---|
| **OpenAI API** | Эмбеддинги (`text-embedding-3-small`) + генерация (GPT) | Да |
| **PostgreSQL + pgvector** | Хранение чанков и векторов | Да |
| **Keycloak** | Верификация JWT токенов (JWKS) | Да |
| **MinIO** | Хранение загруженных файлов | Да (кроме `url`-источников) |
| **Kafka** | Асинхронная обработка задач индексации | Да |

### Python-библиотеки (ключевые)

| Библиотека | Назначение |
|---|---|
| `grpcio` | gRPC-сервер |
| `aiokafka` | Async Kafka consumer/producer |
| `openai` | Эмбеддинги и генерация |
| `unstructured[all-docs]` | Парсинг PDF/DOCX/TXT/MD/HTML |
| `langchain-experimental` | SemanticChunker |
| `sentence-transformers` | CrossEncoder реранкер |
| `asyncpg` + `pgvector` | Async работа с PostgreSQL+pgvector |
| `minio` | Скачивание файлов |
| `python-jose` | Верификация JWT (RS256 + JWKS) |
| `httpx` | HTTP-клиент для URL-источников и JWKS |
| `pydantic-settings` | Конфигурация через env |

---

## 11. Запуск

### Docker Compose (рекомендуется)

```bash
# 1. Заполнить API ключ в .env
echo "OPENAI_API_KEY=sk-..." >> .env

# 2. Запустить сервис вместе с БД
docker compose up -d rag-db knowledge
```

### Локальный запуск

```bash
cd knowledge

# Установить зависимости
pip install -r requirements.txt

# Сгенерировать proto-стабы
bash generate_proto.sh

# Запустить
python main.py
```

### Применение миграций

Миграции применяются **автоматически** при каждом старте сервиса (`main.py` → `run_migrations()`). Файл: [`migrations/001_create_tables.sql`](migrations/001_create_tables.sql).

---

## 12. Структура проекта

```
knowledge/
├── proto/
│   └── rag.proto                  ← Определение gRPC API
├── gen/                           ← Сгенерированные pb2-файлы (не коммитить)
├── core/
│   ├── chunking/
│   │   ├── parsers.py             ← Document-aware парсинг (unstructured)
│   │   └── pipeline.py            ← Semantic chunking (SemanticChunker)
│   ├── embeddings/
│   │   ├── base.py                ← Абстракция EmbeddingProvider
│   │   └── openai_provider.py     ← OpenAI text-embedding-3-small
│   ├── reranker/
│   │   ├── base.py                ← Абстракция Reranker
│   │   └── cross_encoder.py       ← CrossEncoder (ms-marco-MiniLM-L-6-v2)
│   ├── retrieval/
│   │   ├── search.py              ← ANN поиск + реранкинг
│   │   └── rag.py                 ← RAG генерация через OpenAI GPT
│   └── storage/
│       ├── vector_store.py        ← asyncpg + pgvector (чтение/запись)
│       └── minio_client.py        ← Скачивание файлов из MinIO
├── kafka/
│   ├── consumer.py                ← Консьюмер jobs.chunking + jobs.cancel
│   └── producer.py                ← Продюсер jobs.status
├── grpc_server/                   ← (не grpc/ — конфликт с Python-пакетом)
│   ├── handlers.py                ← Реализация RAGService
│   ├── interceptor.py             ← Верификация JWT (Keycloak JWKS)
│   └── server.py                  ← Создание gRPC-сервера
├── migrations/
│   └── 001_create_tables.sql      ← Схема БД
├── config.py                      ← Pydantic Settings
├── main.py                        ← Точка входа (asyncio)
├── requirements.txt
├── Dockerfile
└── generate_proto.sh              ← Генерация pb2-файлов
```
