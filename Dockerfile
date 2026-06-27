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
        ca-certificates \
        ccache \
        cpio \
        curl \
        file \
        flex \
        genext2fs \
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
        sudo \
        texinfo \
        unzip \
        wget \
        xz-utils \
    && rm -rf /var/lib/apt/lists/*

COPY scripts/entry.sh /usr/sbin/entry.sh

# ccache directory — overrideable, matches the mount point used by enter-* targets.
ENV BR2_CCACHE_DIR=/cache/cc

WORKDIR /build

CMD ["bash"]
