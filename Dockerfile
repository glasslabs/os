FROM debian:bookworm-slim

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Build tools required by Buildroot.
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

WORKDIR /build

