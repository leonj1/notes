IMAGE_TEST ?= notes-test

.PHONY: test start stop logs restart

test:
	docker build -f Dockerfile.test -t $(IMAGE_TEST) .
	docker run --rm $(IMAGE_TEST)

# Bring up MySQL, run Flyway migrations, then start the notes server.
start:
	docker compose up --build -d
	@echo
	@echo "notes is starting on http://localhost:8080"
	@echo "Tail logs with: make logs"

stop:
	docker compose down

restart: stop start

logs:
	docker compose logs -f
