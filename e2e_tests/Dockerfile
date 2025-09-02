FROM golang:1.25 AS test
WORKDIR /app

COPY ./internal/ /app/internal/
COPY ./e2e_tests/ /app/e2e_tests/
COPY go.mod go.sum /app/
RUN go mod download

COPY . .

CMD ["go", "test", "./e2e_tests/...", "-v"]