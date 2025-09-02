FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum /app/
COPY ./cmd/ /app/cmd/
COPY ./internal/ /app/internal/
COPY ./api/ /app/api/
COPY ./web/ /app/web/

ENV CGO_ENABLED=0
RUN go build -o app cmd/runner/main.go

FROM scratch

COPY --from=builder /app/app /app/
ENTRYPOINT ["/app/app"]
