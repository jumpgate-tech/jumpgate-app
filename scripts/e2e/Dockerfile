# scripts/e2e/Dockerfile — a systemd + sshd box to pair against.
ARG BASE=debian:12
FROM ${BASE}
ENV DEBIAN_FRONTEND=noninteractive \
    container=docker
RUN apt-get update && apt-get install -y --no-install-recommends \
      systemd systemd-sysv openssh-server sudo procps ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && systemctl enable ssh \
    && mkdir -p /root/.ssh && chmod 700 /root/.ssh
STOPSIGNAL SIGRTMIN+3
CMD ["/lib/systemd/systemd"]
