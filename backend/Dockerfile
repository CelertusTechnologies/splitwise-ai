FROM golang:1.23-alpine AS builder

WORKDIR /app
RUN apk add --no-cache ca-certificates git

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/nivra-api ./cmd/api

FROM alpine:3.21

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/nivra-api /app/nivra-api

EXPOSE 8080
ENTRYPOINT ["/app/nivra-api"]

