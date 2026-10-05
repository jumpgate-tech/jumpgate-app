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
# The --local case: alice is a non-root sudoer who pairs the box for her own
# uid; bob is a second non-root uid the agent's peer gate must refuse.
RUN useradd -m -s /bin/bash alice \
    && echo 'alice ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/alice \
    && chmod 0440 /etc/sudoers.d/alice \
    && useradd -m -s /bin/bash bob
STOPSIGNAL SIGRTMIN+3
# systemd marks / rshared at boot only on a real host; inside a container it
# leaves Docker's rprivate root alone. Credential setup (LoadCredential=) mounts
# the per-unit credentials directory from a child mount namespace, so with a
# private /run the unit sees an empty $CREDENTIALS_DIRECTORY and the agent
# cannot read its key. Do what systemd does on a host, then become PID 1.
CMD ["/bin/sh", "-c", "mount --make-rshared / && exec /lib/systemd/systemd"]
