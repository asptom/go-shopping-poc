FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY ../../go.mod ../../go.sum ./
RUN go mod download

# Copy the entire source code
COPY ../../ ./

# Build the eventreader service
WORKDIR /app/cmd/eventreader
RUN CGO_ENABLED=0 GOOS=linux go build -o /eventreader

# Final image
#FROM gcr.io/distroless/base-debian11
FROM scratch
WORKDIR /
COPY --from=builder /eventreader .
COPY --from=builder /app/.env.local .
COPY --from=builder /etc/passwd /etc/passwd

USER nobody

CMD ["/eventreader"]