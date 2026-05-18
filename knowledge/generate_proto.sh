#!/bin/bash
set -e

mkdir -p gen

python -m grpc_tools.protoc \
    -I proto \
    --python_out=gen \
    --grpc_python_out=gen \
    proto/rag.proto

# Fix absolute imports in generated files (grpc_tools generates broken imports)
sed -i '' 's/^import rag_pb2/from gen import rag_pb2/' gen/rag_pb2_grpc.py 2>/dev/null || \
sed -i 's/^import rag_pb2/from gen import rag_pb2/' gen/rag_pb2_grpc.py

touch gen/__init__.py
echo "Proto generated successfully"
