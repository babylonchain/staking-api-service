#!/bin/bash

# Check if the MongoDB container is already running
MONGO_CONTAINER_NAME="mongodb"
if [ $(docker ps -q -f name=^/${MONGO_CONTAINER_NAME}$) ]; then
    echo "MongoDB container already running. Skipping MongoDB startup."
else
    echo "Starting MongoDB"
    # Start MongoDB
    docker-compose up mongodb -d
fi
