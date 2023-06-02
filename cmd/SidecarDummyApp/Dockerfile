FROM ghcr.io/giobart/active-internal-queue/active-sidecar-queue:v1.0.1
LABEL org.opencontainers.image.source https://github.com/giobart/Active-Internal-Queue

ADD ./bin/ home/

CMD ["/bin/bash","-c","/home/app & /home/SidecarQueue -entry true -exit true -p 5000"]