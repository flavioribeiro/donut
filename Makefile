run:
	docker-compose stop && docker-compose down && docker-compose build && docker-compose up

test:
	docker compose stop test && docker compose down test && docker compose run --rm test

lint:
	docker compose stop lint && docker compose down lint && docker compose run --rm lint	

.PHONY: run
