from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    # Kafka
    kafka_brokers: str = "kafka:29092"
    kafka_chunking_topic: str = "jobs.chunking"
    kafka_cancel_topic: str = "jobs.cancel"
    kafka_status_topic: str = "jobs.status"
    kafka_group_id: str = "knowledge-service"

    # OpenAI
    openai_api_key: str
    openai_chat_model: str = "gpt-4o-mini"

    # PostgreSQL + pgvector
    db_dsn: str = "postgresql://rag:rag@rag-db:5432/rag"

    # MinIO
    minio_endpoint: str = "minio:9000"
    minio_access_key: str = "minioadmin"
    minio_secret_key: str = "minioadmin123"
    minio_bucket: str = "sources"
    minio_secure: bool = False

    # Keycloak (JWT verification)
    keycloak_url: str = "http://keycloak:8082"
    keycloak_realm: str = "Clients"

    # gRPC
    grpc_port: int = 50054

    # Embedding
    embedding_model: str = "text-embedding-3-small"
    embedding_dimensions: int = 1536

    # Chunking
    parent_chunk_max_chars: int = 6000   # ~1500 tokens
    child_chunk_max_tokens: int = 500

    # File size threshold for in-memory vs disk download (bytes)
    max_in_memory_bytes: int = 500 * 1024 * 1024  # 500 MB

    # Processing
    max_retries: int = 3


settings = Settings()
