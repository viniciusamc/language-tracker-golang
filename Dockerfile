FROM golang:latest

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o llt-go ./cmd/*.go

CMD ["./llt-go"]
# CMD ["air", "-c", ".air.toml"]
