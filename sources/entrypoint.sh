#!/bin/sh
set -e

echo "Running migrations..."
./migrate up

echo "Starting service..."
exec ./service