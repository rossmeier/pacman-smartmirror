FROM golang:alpine as builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . ./
RUN go build -o pacman-smartmirror
RUN CGO_ENABLED=0 go test ./...

FROM alpine:latest  
RUN apk --no-cache add ca-certificates curl
RUN echo 'Server = http://mirrors.evowise.com/archlinux/$repo/os/$arch' > /etc/mirrorlist
RUN echo 'Server = http://mirror.archlinuxarm.org/$arch/$repo' >> /etc/mirrorlist
RUN mkdir -p /var/cache/pkg
COPY --from=builder /app/pacman-smartmirror /bin
EXPOSE 80
VOLUME ["/var/cache/pkg"]
CMD ["pacman-smartmirror",  "-l",  ":80", "-m", "/etc/mirrorlist", "-d", "/var/cache/pkg"]  
