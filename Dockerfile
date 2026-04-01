# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY src/ .
RUN go build -o server .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /app
COPY --from=builder /app/server .
COPY src/templates/ src/static/ ./

EXPOSE 8080

CMD ["./server"]
