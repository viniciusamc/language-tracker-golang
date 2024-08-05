# Language Tracker

## Run the project
### Requirements
 - Golang 1.22+
 - Docker
 - Golang-migrate
 - Air (Optional)

```bash
    docker compose -f ./docker-compose-dev.yaml up -d

    migrate -path ./migrations -database "postgres://postgres:123@localhost:5434/llt-go?sslmode=disable" up

    # You can use Air to run the project with live reloading or go run
    go run ./cmd/*.go

    # Live reloading
    air
```
