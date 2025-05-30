FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY ../../go.mod ../../go.sum ./
RUN go mod download

# Copy the entire source code
COPY ../../ ./

# Build the websocket service
WORKDIR /app/cmd/websocket
RUN CGO_ENABLED=0 GOOS=linux go build -o /websocket

# Final image
#FROM gcr.io/distroless/base-debian11
FROM scratch
WORKDIR /
COPY --from=builder /websocket .
COPY --from=builder /app/.env.development .
COPY --from=builder /etc/passwd /etc/passwd

USER nobody

EXPOSE 80

CMD ["/websocket"]