FROM ghcr.io/giobart/active-internal-queue/active-sidecar-queue:latest
LABEL org.opencontainers.image.source https://github.com/giobart/Active-Internal-Queue

ADD ./bin/ home/

CMD ["/bin/bash","-c","/home/app & /home/SidecarQueue -entry=true -exit=true -p=50001"]