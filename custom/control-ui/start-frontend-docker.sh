#!/bin/bash

# Build and run the frontend in production mode using Docker

cd "$(dirname "$0")"

echo "Building Docker images..."
docker-compose -f docker-compose.frontend.yaml build

echo "Starting containers..."
docker-compose -f docker-compose.frontend.yaml up -d

echo ""
echo "Services started successfully!"
echo "- Frontend: http://localhost:3000"
echo "- API: http://localhost:8080"
echo ""
echo "To view logs: docker-compose -f docker-compose.frontend.yaml logs -f"
echo "To stop: docker-compose -f docker-compose.frontend.yaml down"
