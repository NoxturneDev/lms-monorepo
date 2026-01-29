# Stage 1: Development
FROM golang:1.25-alpine AS development

RUN apk add --no-cache git gcc musl-dev

RUN go install github.com/air-verse/air@latest

# Setup Monorepo Root
WORKDIR /app

# Copy Proto and Service
COPY proto/ proto/
COPY school-service/ school-service/

WORKDIR /app/school-service

# Force standard Go Mod mode
ENV GOWORK=off

RUN go mod download

COPY student-service/.air.toml .
COPY student-service/entrypoint.sh .

# Make sure it's executable (just in case)
RUN chmod +x entrypoint.sh

EXPOSE 8080

# Use the script as the command
CMD ["./entrypoint.sh"]

# Stage 2: Builder
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY proto/ proto/
COPY school-service/ school-service/

WORKDIR /app/school-service
ENV GOWORK=off
RUN go mod download

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /school-service-server .

# Stage 3: Production
FROM alpine:latest AS production

WORKDIR /root/
RUN apk --no-cache add ca-certificates sqlite-libs

COPY --from=builder /school-service-server .
RUN mkdir -p ./data

EXPOSE 8080

CMD ["./school-service-server"]
