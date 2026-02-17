FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY ../../go.mod ../../go.sum ./
RUN go mod download

# Copy the entire source code
COPY ../../ ./

# Build the order service
WORKDIR /app/cmd/order
RUN CGO_ENABLED=0 GOOS=linux go build -o /order

# Final image
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /
COPY --from=builder /order .
COPY --from=builder /etc/passwd /etc/passwd

EXPOSE 8083

USER nobody

CMD ["/order"]
