FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY ../../go.mod ../../go.sum ./
RUN go mod download

# Copy the entire source code
COPY ../../ ./

# Build the eventreader service
WORKDIR /app/cmd/customer
RUN CGO_ENABLED=0 GOOS=linux go build -o /customer

# Final image
#FROM gcr.io/distroless/base-debian11
#FROM 
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /
COPY --from=builder /customer .
COPY --from=builder /etc/passwd /etc/passwd

# Note Be sure to expose ports that match your application just to ensure the container runtime allows them.
EXPOSE 8080

USER nobody

CMD ["/customer"]