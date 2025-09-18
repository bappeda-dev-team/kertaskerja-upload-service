APP_NAME=kertaskerja-upload-service
include .env
export $(shell sed 's/=.*//' .env)

.PHONY: all build run clean

# DEFAULT TARGET
all: build

build: $(APP_NAME)

$(APP_NAME): *.go
	@echo ">>> Building $(APP_NAME)..."
	@go build -o $(APP_NAME) .
	@echo ">>> SUCCESS: binary created"

run: build
	@echo ">>> Running $(APP_NAME)..."
	./$(APP_NAME)

clean:
	@echo "CLEANING UP"
	rm -f $(APP_NAME)
