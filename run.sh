#!/bin/bash
set -e

if ! command -v docker compose &> /dev/null; then
  echo "Ошибка: docker-compose не найден"
  exit 1
fi

echo "Выберите окружение:"
echo "1) prod"
echo "2) dev"
echo "3) automated-cache-only"

read -p "Введите соответствующий ему номер (1-3): " choice

case $choice in
  1) C_FILE_PATH="build/prod/docker-compose.yml" ;;
  2) C_FILE_PATH="build/dev/docker-compose.yml" ;;
  3) C_FILE_PATH="build/automated/docker-compose-cache.yml" ;;
  *) echo "Неверный номер окружения"; exit 1 ;;
esac

echo "Выберите действие с файлом $C_FILE_PATH..."
echo "0) local go main file build"
echo "1) build (no cache)"
echo "2) build (with cache)"
echo "3) up"
echo "4) down"
echo "5) unpack env by environment"
echo "6) build main file only"
echo "7) WARNING! Docker full clear"

read -p "Введите соответствующий номер операции над файлом (0-7): " command

case $command in
  0) COMMAND="local_go_build" ;;
  1) COMMAND="build_no_cache" ;;
  2) COMMAND="build_with_cache" ;;
  3) COMMAND="up" ;;
  4) COMMAND="down" ;;
  5) COMMAND="unpack_env_by_environment" ;;
  6) COMMAND="build_main_only" ;;
  7) COMMAND="docker_clear" ;;
  *) echo "Неверный номер операции"; exit 1 ;;
esac

BASE_COMPOSE="docker-compose.yml"
# Переменная для фиксации контекста в корне проекта
P_DIR="--project-directory ."

if [ "$C_FILE_PATH" = "build/automated/docker-compose-cache.yml" ]; then
    IS_AUTOMATED_TEST_OPERATION=true
else
    IS_AUTOMATED_TEST_OPERATION=false
fi

if [ "$COMMAND" = "build_no_cache" ]; then
  if [ "$C_FILE_PATH" = "build/automated/docker-compose-cache.yml" ]; then
    echo "Операция запрещена"
    exit 1
  else
    docker compose $P_DIR -f "$BASE_COMPOSE" -f "$C_FILE_PATH" build --no-cache --progress=plain
  fi

elif [ "$COMMAND" = "build_with_cache" ]; then
  if [ "$C_FILE_PATH" = "build/automated/docker-compose-cache.yml" ]; then
    COMPOSE_DOCKER_CLI_BUILD=1 DOCKER_BUILDKIT=1 docker compose $P_DIR -f docker-compose-automated.yml -f "$C_FILE_PATH" build
  else
    docker compose $P_DIR -f "$BASE_COMPOSE" -f "$C_FILE_PATH" build
  fi

elif [ "$COMMAND" = "up" ]; then
  if [ "$IS_AUTOMATED_TEST_OPERATION" = false ]; then
    docker compose $P_DIR -f "$BASE_COMPOSE" -f "$C_FILE_PATH" up
  else
    docker compose $P_DIR -f "docker-compose-automated.yml" -f "$C_FILE_PATH" up
  fi

elif [ "$COMMAND" = "down" ]; then
  if [ "$IS_AUTOMATED_TEST_OPERATION" = false ]; then
    docker compose $P_DIR -f "$BASE_COMPOSE" -f "$C_FILE_PATH" down
  else
    docker compose $P_DIR -f "docker-compose-automated.yml" -f "$C_FILE_PATH" down
  fi

elif [ "$COMMAND" = "unpack_env_by_environment" ]; then
  if  [ "$C_FILE_PATH" = "build/prod/docker-compose.yml" ]; then
    echo "Операция запрещена"
    exit 1
  elif [ "$C_FILE_PATH" = "build/dev/docker-compose.yml" ]; then
    cp .env.dev .env
    sudo cp .env /app/.env
  elif [ "$IS_AUTOMATED_TEST_OPERATION" = true ]; then
    cp .env.automated .env
    sudo cp .env /app/.env
  fi

elif [ "$COMMAND" = "build_main_only" ]; then
  echo "Введите имя контейнера"
  read CONTAINER_NAME
  docker cp ./ "$CONTAINER_NAME":/app/
  docker compose $P_DIR -f "$BASE_COMPOSE" -f "$C_FILE_PATH" exec api go mod init chickChirick-message
  docker compose $P_DIR -f "$BASE_COMPOSE" -f "$C_FILE_PATH" exec api go mod tidy
  docker compose $P_DIR -f "$BASE_COMPOSE" -f "$C_FILE_PATH" exec api go build -o main app/cmd/main.go

elif [ "$COMMAND" = "docker_clear" ]; then
  read -p "Удалить все контейнеры, образы, волюмы и т.д? (y/n): " answer
  if [[ "$answer" =~ ^[Yy]$ ]]; then
    containers=$(docker ps -aq)
    if [ -n "$containers" ]; then
      docker stop $containers
      docker rm $containers
    fi
    docker volume prune -f
    docker system prune -f
  else
    echo "Операция отменена"
  fi

elif [ "$COMMAND" = "local_go_build" ]; then
  echo ">>> Локальная сборка бинарника под Linux (amd64)..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags="all=-N -l" -o main ./cmd/main.go
  echo ">>> Готово! Файл 'main' создан в корне проекта."
fi
