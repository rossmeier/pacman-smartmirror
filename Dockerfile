FROM golang:alpine as builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . ./
RUN go build -o pacman-smartmirror
RUN sed -i s/data/\\/var\\/cache\\/pkg/g config.yml
RUN sed -i s/41234/80/g config.yml
RUN CGO_ENABLED=0 go test ./...

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl
RUN mkdir -p /var/cache/pkg
COPY --from=builder /app/pacman-smartmirror /bin
COPY --from=builder /app/config.yml /etc
EXPOSE 80
VOLUME ["/var/cache/pkg"]
CMD ["pacman-smartmirror", "-c", "/etc/config.yml"]
