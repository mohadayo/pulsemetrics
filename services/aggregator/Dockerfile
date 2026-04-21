FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o aggregator .

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/aggregator .
EXPOSE 8002
CMD ["./aggregator"]
