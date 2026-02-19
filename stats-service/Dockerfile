# Stage 1: Development
FROM golang:1.25-alpine AS development

# Add build tools
RUN apk add --no-cache git gcc musl-dev

RUN go install github.com/air-verse/air@latest

# 1. Setup the Monorepo Structure at /app
WORKDIR /app

# 2. Copy the Shared Proto Module
COPY proto/ proto/

# 3. Copy the Service Code
COPY stats-service/ stats-service/

# 4. Move into the service directory
WORKDIR /app/stats-service

# 5. FORCE Go to ignore 'go.work' and use 'go.mod' replace directives
ENV GOWORK=off

# 6. Install dependencies
# (We copy go.mod first for caching, but since we already copied the folder above, 
#  we can just run download. The cache hit might be slightly less optimal but it's safer for now)
RUN go mod download

# 7. Install Air for hot reload

COPY stats-service/.air.toml .
COPY stats-service/entrypoint.sh .

# Make sure it's executable (just in case)
RUN chmod +x entrypoint.sh

EXPOSE 8080

# Use the script as the command
CMD ["./entrypoint.sh"]

# Stage 2: Builder (Production)
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Replicate structure again for builder
COPY proto/ proto/
COPY stats-service/ stats-service/

WORKDIR /app/stats-service
ENV GOWORK=off

RUN go mod download

# Build
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /stats-service-server .

# Stage 3: Production
FROM alpine:latest AS production

WORKDIR /root/
RUN apk --no-cache add ca-certificates sqlite-libs

COPY --from=builder /stats-service-server .
RUN mkdir -p ./data

EXPOSE 8080

CMD ["./stats-service-server"]
