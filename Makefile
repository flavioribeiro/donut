run:
	docker compose stop && docker compose up origin srt app

run-dev:
	docker compose stop && docker compose down && docker compose build app && docker compose up origin srt app

run-dev-total-rebuild:
	docker compose stop && docker compose down && docker compose build && docker compose up origin srt app

clean-docker:
	docker-compose down -v --rmi all --remove-orphans &&  docker volume prune -a -f && docker system prune -a -f && docker builder prune -a -f

lint:
	docker compose stop lint && docker compose down lint && docker compose run --rm lint	

.PHONY: run lint test run-srt mac-run-local mac-test-local html-local-coverage install-ffmpeg
