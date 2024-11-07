FROM golang:1.22 as build
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o cdpshell

FROM debian:bookworm-slim

# Redis CLI and misc os stuff
RUN apt update && \
    apt install -y curl strace wget unzip redis-tools less && \
    apt clean && \
    rm -rf /var/lib/apt/lists/*


COPY --from=build /go/cdpshell /usr/bin/cdpshell

RUN useradd -ms /bin/bash cdpshell
WORKDIR /home/cdpshell

ENTRYPOINT [ "/usr/bin/cdpshell", "-home", "/home/cdpshell" ]
