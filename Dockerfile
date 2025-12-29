FROM alpine:latest
WORKDIR /home

ARG TARGETARCH
COPY ./dist/rttys-linux-${TARGETARCH} /usr/bin/rttys

ENTRYPOINT ["/usr/bin/rttys"]
