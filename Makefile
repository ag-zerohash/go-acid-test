.PHONY: run
run:
	docker compose up --build

.PHONY: drop
drop:
	docker compose down
