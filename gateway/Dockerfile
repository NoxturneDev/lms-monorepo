FROM golang:1.25-alpine AS development

RUN apk add --no-cache git

RUN go install github.com/air-verse/air@latest

WORKDIR /app

# Copy Proto (Because we import it)
COPY proto/ proto/
COPY gateway/ gateway/

WORKDIR /app/gateway

ENV GOWORK=off

# Copy air config (Create this file next!)
COPY gateway/.air.toml .

RUN go mod download

EXPOSE 3000

CMD ["air", "-c", ".air.toml"]

# ================================================================
# Stage 2: Builder
# ================================================================

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY proto/ proto/
COPY gateway/ gateway/

WORKDIR /app/gateway

ENV GOWORK=off

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gateway-server .

# ================================================================
# Stage 3: Production
# ================================================================

FROM alpine:latest AS production

WORKDIR /root/

RUN apk --no-cache add ca-certificates

COPY --from=builder /gateway-server .

EXPOSE 3000

CMD ["./gateway-server"]
