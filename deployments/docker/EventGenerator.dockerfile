# ==========================================
# STAGE 1: Compile the Go Application
# ==========================================
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency manifests for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the monorepo source code
COPY . .

# Compile the specialized mock generation binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o event-generator ./cmd/event-generator

# ==========================================
# STAGE 2: Construct the Lean Deployment Image
# ==========================================
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Retrieve only the executable
COPY --from=builder /app/event-generator .

CMD ["./event-generator"]