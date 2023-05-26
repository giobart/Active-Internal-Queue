FROM ubuntu:22.04
LABEL org.opencontainers.image.source https://github.com/giobart/Active-Internal-Queue

ADD cmd/SidecarQueue/bin .

CMD ["./bin/SidecarQueue","-entry","true","-exit","true","-p","5000",]