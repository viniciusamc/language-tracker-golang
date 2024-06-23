dev:
	docker compose -f ./docker-compose-test.yaml kill
	docker compose -f ./docker-compose-dev.yaml up -d
	sleep 3
	air

kill:
	docker compose -f ./docker-compose-dev.yaml kill
	docker compose -f ./docker-compose.yaml kill

load:
	docker compose -f ./docker-compose-dev.yaml kill
	docker compose -f ./docker-compose-test.yaml kill
	docker compose -f ./docker-compose-test.yaml build
	docker compose -f ./docker-compose-test.yaml up -d

migrate:
	export POSTGRESQL_URL='postgres://postgres:123@localhost:5432/yaml?sslmode=disable'
	migrate -path=./migrations -database=$POSTGRESQL_URL up
