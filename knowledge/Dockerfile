FROM python:3.11-slim

# System deps required by unstructured / python-magic / tesseract
RUN apt-get update && apt-get install -y --no-install-recommends \
    libmagic1 \
    poppler-utils \
    tesseract-ocr \
    libreoffice \
    pandoc \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install Python dependencies first (Docker layer cache)
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy source
COPY . .

# Generate protobuf stubs
RUN mkdir -p gen && \
    python -m grpc_tools.protoc \
        -I proto \
        --python_out=gen \
        --grpc_python_out=gen \
        proto/rag.proto && \
    sed -i 's/^import rag_pb2/from gen import rag_pb2/' gen/rag_pb2_grpc.py && \
    touch gen/__init__.py

# Pre-download CrossEncoder model so the container starts faster
RUN python -c "from sentence_transformers import CrossEncoder; CrossEncoder('cross-encoder/ms-marco-MiniLM-L-6-v2')"

EXPOSE 50054

CMD ["python", "main.py"]
