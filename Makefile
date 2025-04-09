include .env
export

build:
	go build -o main .

run: build
	nohup ./main > output.log 2>&1 &

up:
	docker compose -f docker-compose.yaml up -d --build
