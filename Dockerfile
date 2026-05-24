FROM debian:bookworm-slim

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Build tools required by Buildroot (installed first so wget is available for the Go download below).
RUN apt-get update && apt-get install -y --no-install-recommends \
        automake \
        bash \
        bc \
        binutils \
        bison \
        build-essential \
        bzip2 \
        cpio \
        file \
        flex \
        genimage \
        gettext \
        git \
        help2man \
        libncurses-dev \
        libssl-dev \
        make \
        openssh-client \
        patch \
        perl \
        python3 \
        python3-setuptools \
        rsync \
        texinfo \
        unzip \
        wget \
    && rm -rf /var/lib/apt/lists/*

# Install Go for cross-compiling the glass-agent binary.
ARG GO_VERSION=1.26.3
RUN set -eux; \
    ARCH=$(dpkg --print-architecture); \
    case "$ARCH" in \
        amd64) GO_ARCH=amd64 ;; \
        arm64) GO_ARCH=arm64 ;; \
        *) echo "Unsupported build arch: $ARCH" >&2; exit 1 ;; \
    esac; \
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" \
        -O /tmp/go.tar.gz; \
    tar -C /usr/local -xzf /tmp/go.tar.gz; \
    rm /tmp/go.tar.gz
ENV PATH="${PATH}:/usr/local/go/bin"

WORKDIR /build
