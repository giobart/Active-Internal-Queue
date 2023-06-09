FROM ubuntu:22.04 AS ubuntu22
LABEL org.opencontainers.image.source https://github.com/giobart/Active-Internal-Queue

ADD ./bin/ home/

CMD ["/home/SidecarQueue","-entry","true","-exit","true","-p","5000"]


FROM nvidia/cuda:12.1.1-cudnn8-devel-ubuntu22.04 AS cuda12
LABEL org.opencontainers.image.source https://github.com/giobart/Active-Internal-Queue

ADD ./bin/ home/

CMD ["/home/SidecarQueue","-entry","true","-exit","true","-p","5000"]