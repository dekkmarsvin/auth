FROM golang:1.24.5-alpine3.22 AS builder
WORKDIR /api
COPY go.mod go.sum .
RUN go mod download
COPY . .
RUN go build -o app

FROM alpine:3.22.1
COPY --from=builder /api/app /api/app
WORKDIR /api
EXPOSE 8080
CMD ["./app"]
