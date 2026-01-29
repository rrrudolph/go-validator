# Build and run go-validator tests
FROM golang:1.25-alpine AS test-runner

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

CMD ["go", "test", "-v", "./..."]
