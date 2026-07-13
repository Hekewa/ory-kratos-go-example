FROM golang:1.26-alpine AS builder

WORKDIR /app
# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o main .

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/main .
COPY assets assets

ENTRYPOINT ["/root/main"]