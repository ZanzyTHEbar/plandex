# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get

# Main package name
MAIN_PACKAGE = main

# Output binary name
BINARY_NAME = plandex

# Check the PLANDEX_ENVIRONMENT environment variable, reassign the BINARY_NAME if necessary
ifeq ($(PLANDEX_ENV),development)
BINARY_NAME = plandex-dev
endif

WEBHOOK_SERVER_NAME=webhook
WEBHOOK_PORT=9797
WEBHOOK_BASE_URL=http://localhost:$(WEBHOOK_PORT)
WEBHOOK_URL=$(WEBHOOK_BASE_URL)/webhook
WEBHOOK_HEALTH_URL=$(WEBHOOK_BASE_URL)/health

# create a dev cmd that runs a shell script
dev:
	@cd app/scripts && ./dev.sh

# Build target
build:
	@$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PACKAGE)

# Clean target
clean:
	@$(GOCLEAN)
	@rm -f $(BINARY_NAME)

# Test target
test: render
	@$(GOTEST) -v ./...

#### Evals and Providers ####


#! No cache is used to ensure that the latest changes are reflected in the eval
# TODO: Implement eval all

# Target to start the webhook server
start-webhook:
	@echo "Starting Go webhook server on port $(WEBHOOK_PORT) ..."
	@$(GOCMD) run app/scripts/cmd/webhook-test/webhook.go & \
	WEBHOOK_PID=$$! && \
	echo "Webhook server started with PID $$WEBHOOK_PID" && \
	echo "Process Name: $(WEBHOOK_SERVER_NAME), Port: $(WEBHOOK_PORT)"
	@sleep 2

# Target to stop the webhook server
stop-webhook:
	@echo "Stopping Go webhook server on port $(WEBHOOK_PORT)..."
	@PID_TO_KILL=$$(lsof -i :$(WEBHOOK_PORT) -t) && \
	if [ ! -z "$$PID_TO_KILL" ]; then \
		echo "Found process with PID $$PID_TO_KILL to kill"; \
		kill -9 $$PID_TO_KILL && \
		echo "Webhook server with PID $$PID_TO_KILL stopped"; \
	else \
		echo "No process found on port $(WEBHOOK_PORT)"; \
	fi
	@sleep 2

# Health check target to verify if the server is running
health-check-webhook:
	@echo "Checking health of the webhook server..."
	@HTTP_STATUS=$$(curl -s -o /dev/null -w "%{http_code}" $(WEBHOOK_HEALTH_URL)); \
	if [ "$$HTTP_STATUS" -eq 200 ]; then \
		echo "Webhook server is healthy and running."; \
	else \
		echo "Webhook server is not running or unhealthy (status: $$HTTP_STATUS)."; \
	fi

eval: start-webhook
	@sleep 2
	@$(MAKE) health-check-webhook
	@cd test/evals/promptfoo-poc/$(filter-out $@,$(MAKECMDGOALS)) && promptfoo eval --no-cache
	@sleep 10
	$(MAKE) stop-webhook
	@$(MAKE) health-check-webhook

view-eval:
	@cd test/evals/promptfoo-poc/$(filter-out $@,$(MAKECMDGOALS)) && promptfoo view

gen-eval:
	@$(GOCMD) run app/scripts/cmd/gen/gen.go test/evals/promptfoo-poc/$(filter-out $@,$(MAKECMDGOALS))

gen-provider:
	@$(GOCMD) run app/scripts/cmd/provider/gen_provider.go

#### End Evals and Providers ####

# Get dependencies
deps:
	$(GOGET) -v ./...

# Default target
default: build

# Usage
help:
	@echo "Usage:"
	@echo "  make dev - to run the development scripts"
	@echo "  make eval <directory_name> - to run the promptfoo eval command on a specific directory"
	@echo "  make view-eval - to view the promptfoo eval output"
	@echo "  make gen-eval <directory_name> - to create a new promptfoo eval directory structure"
	@echo "  make gen-provider - to create a new promptfoo provider file from the promptfoo diretory structure"
	@echo "  make clean - to remove generated files and directories"
	@echo "  make help - to display this help message"

devTests:
	@$(GOCMD) run app/scripts/cmd/dev/dev.go $(filter-out $@,$(MAKECMDGOALS))

# Prevents make from interpreting the arguments as targets
%:
	@:

.PHONY: all render build clean test deps