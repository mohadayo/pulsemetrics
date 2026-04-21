.PHONY: test test-python test-go test-ts lint up down build clean

test: test-python test-go test-ts
	@echo "All tests passed."

test-python:
	cd services/ingestor && pip install -q -r requirements.txt && pytest -v

test-go:
	cd services/aggregator && go test -v ./...

test-ts:
	cd services/alerter && npm install --silent && npm test

lint: lint-python lint-go lint-ts
	@echo "All linters passed."

lint-python:
	cd services/ingestor && flake8 app.py test_app.py --max-line-length=120

lint-go:
	cd services/aggregator && go vet ./...

lint-ts:
	cd services/alerter && npm install --silent && npx eslint src/ --ext .ts

build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

clean:
	docker compose down -v --rmi local
