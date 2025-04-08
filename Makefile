build:
	go build -o main .

run: build
	./main

up:
	docker compose -f docker-compose.yaml up -d --build