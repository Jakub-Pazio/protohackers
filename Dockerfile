FROM golang:alpine3.22 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./cmd/pestcontrol

FROM alpine:3.22.2
COPY --from=builder /out/app /app
ENTRYPOINT ["/app"]
