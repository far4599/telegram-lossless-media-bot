FROM golang:1.17-alpine as builder
WORKDIR /src
ADD . .
RUN go mod download && CGO_ENABLED=0 GOOS=linux go build -a -o main ./cmd

FROM alpine
RUN apk add --update ca-certificates && rm -rf /tmp/* /var/cache/apk/*
WORKDIR /app
COPY --from=builder /src/main .
ENTRYPOINT ["/app/main"]