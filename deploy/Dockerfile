FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/claude-remote-server ./server

# -------------------------
FROM alpine:3.20

RUN apk add --no-cache \
    ca-certificates \
    git \
    openssh-server \
    bash

# Create dedicated service user
RUN addgroup -S claude-remote && \
    adduser -S -G claude-remote -h /home/claude-remote claude-remote

# Data directory for instances
RUN mkdir -p /var/lib/claude-remote/instances && \
    chown -R claude-remote:claude-remote /var/lib/claude-remote

# SSH host keys
RUN ssh-keygen -A && \
    mkdir -p /home/claude-remote/.ssh && \
    chmod 700 /home/claude-remote/.ssh && \
    chown claude-remote:claude-remote /home/claude-remote/.ssh

COPY --from=builder /bin/claude-remote-server /usr/local/bin/claude-remote-server

# SSH config: public-key auth only, no password, forced command
COPY deploy/sshd_config /etc/ssh/sshd_config

EXPOSE 22

# Note: claude CLI must be installed separately or via a layer.
# Add your public key to /home/claude-remote/.ssh/authorized_keys.
VOLUME ["/var/lib/claude-remote", "/home/claude-remote/.ssh"]

CMD ["/usr/sbin/sshd", "-D", "-e"]
