# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the product service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o product ./cmd/product

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/product .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./product"]