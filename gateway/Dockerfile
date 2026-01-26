FROM golang:1.25-alpine AS development

RUN apk add --no-cache git

WORKDIR /app

# Copy Proto (Because we import it)
COPY proto/ proto/
COPY gateway/ gateway/

WORKDIR /app/gateway

RUN go install github.com/air-verse/air@latest
ENV GOWORK=off

# Copy air config (Create this file next!)
COPY gateway/.air.toml .

RUN go mod download

EXPOSE 3000

CMD ["air", "-c", ".air.toml"]
