run:
	docker compose stop && docker compose down && docker compose build && docker compose up origin srt app

test:
	# in case you need to re-build it
	# docker compose stop test && docker compose down test && docker compose build test && docker compose run --rm test
	docker compose stop test && docker compose down test && docker compose run --rm test

mac-test-local:
	./scripts/mac_local_run_test.sh

html-local-coverage:
	go tool cover -html=coverage.out

lint:
	docker compose stop lint && docker compose down lint && docker compose run --rm lint	

.PHONY: run
