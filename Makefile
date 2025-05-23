include .env
export

build:
	go build -o main .

run: build free
	nohup ./main > output.log 2>&1 &

up:
	docker compose -f docker-compose.yaml up -d --build

free:
	@./scripts/free_port.sh 8182
