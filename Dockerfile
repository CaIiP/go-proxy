FROM golang:1.19-alpine AS builder

# Set working directory
WORKDIR /app

# Copy source code
COPY server.go .

# Build the Go application
RUN go build -o server server.go

# Use a minimal alpine image for the final container
FROM alpine:3.17

# Install necessary packages
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/server /usr/local/bin/

# Expose the port
EXPOSE 8000

# Set the entry point
ENTRYPOINT ["/usr/local/bin/server"]