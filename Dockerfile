FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /xan-pythia ./cmd/server/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /xan-pythia /xan-pythia
COPY --from=builder /app/db/migrations /db/migrations
COPY --from=builder /app/web /web
WORKDIR /
EXPOSE 8080
CMD ["/xan-pythia"]
