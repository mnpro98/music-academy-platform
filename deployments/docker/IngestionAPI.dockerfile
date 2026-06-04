# ==========================================
# STAGE 1: Compile the Go Application
# ==========================================
FROM golang:1.26-alpine AS builder

# Set the operational directory inside the container
WORKDIR /app

# Copy dependency manifests first to leverage Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire monorepo source code context
COPY . .

# Compile the static binary
# CGO_ENABLED=0 disables dynamic C-library linking to guarantee portability
# GOOS=linux targets the deployment environment operating system
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ingestion-api ./cmd/ingestion-api

# ==========================================
# STAGE 2: Construct the Lean Deployment Image
# ==========================================
FROM alpine:3.19

# Install security certificates for external HTTPS network calls
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Retrieve only the compiled executable from the build stage container
COPY --from=builder /app/ingestion-api .

EXPOSE 8080

CMD ["./ingestion-api"]