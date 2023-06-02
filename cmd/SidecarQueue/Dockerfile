FROM ubuntu:22.04
LABEL org.opencontainers.image.source https://github.com/giobart/Active-Internal-Queue

ADD bin/ home/

CMD ["/home/SidecarQueue","-entry","true","-exit","true","-p","5000"]