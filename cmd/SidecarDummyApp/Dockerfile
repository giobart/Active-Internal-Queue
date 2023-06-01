FROM FROM ghcr.io/giobart/active-internal-queue/active-sidecar-queue:v1.0.1

ADD bin .

CMD ["/bin/bash","-c","./bin/app & ./bin/SidecarQueue -entry true -exit true -p 5000"]