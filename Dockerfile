FROM golang:1.26-alpine AS builder

# Set the working directory inside the builder container
WORKDIR /app

# Copy your source code into the builder container
COPY . .

# Build the binary and name it 'main'
RUN go build -o main .

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/main .
COPY assets assets

ENTRYPOINT ["/root/main"]