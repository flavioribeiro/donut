run:
	docker compose stop && docker compose up app

run-dev:
	docker compose stop && docker compose down && docker compose build app && docker compose up app

run-dev-total-rebuild:
	docker compose stop && docker compose down && docker compose build && docker compose up app

clean-docker:
	docker-compose down -v --rmi all --remove-orphans &&  docker volume prune -a -f && docker system prune -a -f && docker builder prune -a -f

run-docker-dev:
	docker compose run --rm --service-ports dev

run-server-inside-docker:
	go run main.go -- --enable-ice-mux=true

run-srt-rtmp-streaming-alone:
	docker compose stop && docker compose down && docker compose up nginx_rtmp haivision_srt

lint:
	docker compose stop lint && docker compose down lint && docker compose run --rm lint	
