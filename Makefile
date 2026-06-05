BOARDS       := rpi4 rpi5
BR2_EXTERNAL := $(CURDIR)/buildroot-external
BUILDROOT    := $(CURDIR)/buildroot
AGENT_DIST   := $(CURDIR)/agent/dist
GLASS_DIST   := $(CURDIR)/glass/dist

# Glass version — keep in sync with BR2_PACKAGE_GLASS_VERSION in the defconfigs.
GLASS_VERSION ?= v2.0.3
# Glass variant — the Gio window-system backend compiled into the binary.
# Choices: wayland (default, no X11 dep), x11, framebuffer, full.
# Must match the variant suffix used in the looking-glass GitHub release archive.
GLASS_VARIANT ?= wayland

# GlassOS image version — passed through to the RAUC bundle manifest and image
# file names.  Override on the command line (e.g. GLASSOS_VERSION=1.0.0) or let
# it fall back to the default defined in buildroot-external/meta.
GLASSOS_VERSION ?=

# Cache directories (override on the command line or via env to enable ccache).
BR2_DL_DIR   ?= $(BUILDROOT)/dl
BR2_CCACHE_DIR ?=

DOCKER_IMAGE := glassos-builder
DOCKER_RUN   := docker run --rm \
    -v $(CURDIR):/build \
    -v $(BR2_DL_DIR):/cache/dl \
    -w /build \
    $(DOCKER_IMAGE)

# Phony declarations up front.
.PHONY: $(addprefix build-,$(BOARDS))
.PHONY: $(addprefix menuconfig-,$(BOARDS))
.PHONY: $(addprefix linux-menuconfig-,$(BOARDS))
.PHONY: $(addprefix savedefconfig-,$(BOARDS))
.PHONY: $(addprefix clean-,$(BOARDS))
.PHONY: $(addprefix uboot-rebuild-,$(BOARDS))
.PHONY: $(addprefix docker-run-,$(BOARDS))
.PHONY: $(addprefix docker-uboot-rebuild-,$(BOARDS))
.PHONY: build-agent download-glass clean-all docker-build flash test-agent help

# Cross-compile a static glass-agent binary for linux/arm64.
build-agent:
	@echo "==> Building glass-agent (linux/arm64, static)"
	@mkdir -p $(AGENT_DIST)
	@cd agent && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		go build \
		-ldflags="-s -w -extldflags=-static" \
		-tags "osusergo netgo" \
		-trimpath \
		-o $(AGENT_DIST)/glass-agent \
		.
	@echo "==> Done"

# Download the glass binary for linux/arm64 from GitHub Releases and unzip it.
download-glass:
	@echo "==> Downloading glass $(GLASS_VERSION) (linux/arm64, $(GLASS_VARIANT))"
	@mkdir -p $(GLASS_DIST)
	@curl -fsSL \
		"https://github.com/glasslabs/looking-glass/releases/download/$(GLASS_VERSION)/glass-$(GLASS_VERSION)-linux-arm64-$(GLASS_VARIANT).zip" \
		-o "$(GLASS_DIST)/glass.zip"
	@unzip -o "$(GLASS_DIST)/glass.zip" -d "$(GLASS_DIST)"
	@rm -f "$(GLASS_DIST)/glass.zip"
	@chmod +x "$(GLASS_DIST)/glass"
	@echo "==> Done"

# build-rpi4 / build-rpi5 — depend on build-agent and download-glass so both
# binaries are always current before Buildroot runs.
$(addprefix build-,$(BOARDS)): build-agent download-glass
$(addprefix build-,$(BOARDS)): build-%:
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BR2_EXTERNAL) \
		BR2_DL_DIR=$(BR2_DL_DIR) \
		glassos_$*_defconfig
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BR2_EXTERNAL) \
		BR2_DL_DIR=$(BR2_DL_DIR) \
		$(if $(BR2_CCACHE_DIR),BR2_CCACHE=y BR2_CCACHE_DIR=$(BR2_CCACHE_DIR),) \
		$(if $(GLASSOS_VERSION),GLASSOS_VERSION=$(GLASSOS_VERSION),) \
		-j$(shell nproc)

# menuconfig-rpi4 / menuconfig-rpi5
$(addprefix menuconfig-,$(BOARDS)): menuconfig-%:
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BR2_EXTERNAL) \
		menuconfig

# linux-menuconfig-rpi4 / linux-menuconfig-rpi5
$(addprefix linux-menuconfig-,$(BOARDS)): linux-menuconfig-%:
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BR2_EXTERNAL) \
		linux-menuconfig

# savedefconfig-rpi4 / savedefconfig-rpi5
$(addprefix savedefconfig-,$(BOARDS)): savedefconfig-%:
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BR2_EXTERNAL) \
		BR2_DEFCONFIG=$(BR2_EXTERNAL)/configs/glassos_$*_defconfig \
		savedefconfig

# docker-build — build the glassos-builder Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE) .

# docker-run-rpi4 / docker-run-rpi5 — run the full build inside the glassos-builder
# container.  A named volume per board is used for the output directory so that
# host-compiled Buildroot host-tools never bleed into the container (which would
# cause "Exec format error" on a different arch).
$(addprefix docker-run-,$(BOARDS)): docker-run-%:
	docker run --rm \
	    -v "$(CURDIR)":/build \
	    -v "glassos-output-$*":/build/buildroot/output/$* \
	    -v "$(BR2_DL_DIR)":/build/buildroot/dl \
	    -w /build \
	    $(DOCKER_IMAGE) \
	    make build-$* \
	        $(if $(BR2_CCACHE_DIR),BR2_CCACHE_DIR=$(BR2_CCACHE_DIR),) \
	        $(if $(GLASSOS_VERSION),GLASSOS_VERSION=$(GLASSOS_VERSION),) \
	        GLASS_VERSION=$(GLASS_VERSION) \
	        GLASS_VARIANT=$(GLASS_VARIANT)

# docker-uboot-rebuild-rpi4 / docker-uboot-rebuild-rpi5 — force U-Boot recompile
# inside the Docker container (use after uboot.config or uboot-boot.ush change).
$(addprefix docker-uboot-rebuild-,$(BOARDS)): docker-uboot-rebuild-%:
	docker run --rm \
	    -v "$(CURDIR)":/build \
	    -v "glassos-output-$*":/build/buildroot/output/$* \
	    -v "$(BR2_DL_DIR)":/build/buildroot/dl \
	    -w /build \
	    $(DOCKER_IMAGE) \
	    make uboot-rebuild-$* \
	        $(if $(BR2_CCACHE_DIR),BR2_CCACHE_DIR=$(BR2_CCACHE_DIR),) \
	        $(if $(GLASSOS_VERSION),GLASSOS_VERSION=$(GLASSOS_VERSION),)

# flash BOARD=rpi4 DEV=/dev/sdX [SSID=x PSK=y]
flash:
	@[ -n "$(BOARD)" ] || { echo "Usage: make flash BOARD=rpi4 DEV=/dev/sdX [SSID=x PSK=y]"; exit 1; }
	@[ -n "$(DEV)" ]   || { echo "Usage: make flash BOARD=rpi4 DEV=/dev/sdX [SSID=x PSK=y]"; exit 1; }
	@echo "==> Flashing $(BOARD) image to $(DEV) -- this will erase the device."
	@read -p "    Continue? [y/N] " confirm && [ "$$confirm" = "y" ]
	xz -dc $(BUILDROOT)/output/$(BOARD)/images/sdcard.img.xz | sudo dd of=$(DEV) bs=4M conv=fsync status=progress
	sync
	@if [ -n "$(SSID)" ] && [ -n "$(PSK)" ]; then \
		echo "==> Writing WiFi credentials to boot partition"; \
		MOUNT=$$(mktemp -d); \
		sudo mount $(DEV)1 $$MOUNT; \
		printf '[connection]\nid=provisioned-wifi\ntype=wifi\nautoconnect=yes\n\n[wifi]\nmode=infrastructure\nssid=$(SSID)\n\n[wifi-security]\nkey-mgmt=wpa-psk\npsk=$(PSK)\n\n[ipv4]\nmethod=auto\n\n[ipv6]\nmethod=auto\naddr-gen-mode=stable-privacy\n' \
			| sudo tee $$MOUNT/provisioned-wifi.nmconnection > /dev/null; \
		sudo umount $$MOUNT && rmdir $$MOUNT; \
		echo "==> WiFi credentials written."; \
	fi

# clean-rpi4 / clean-rpi5
$(addprefix clean-,$(BOARDS)): clean-%:
	rm -rf $(BUILDROOT)/output/$*

# uboot-rebuild-rpi4 / uboot-rebuild-rpi5 — force U-Boot to be recompiled
# (needed when uboot.config or uboot-boot.ush change without a full clean).
# host-uboot-tools is also cleaned because it owns the boot.scr compilation
# step; without this, the old boot.scr survives the rebuild.
$(addprefix uboot-rebuild-,$(BOARDS)): uboot-rebuild-%:
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BR2_EXTERNAL) \
		uboot-dirclean
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BR2_EXTERNAL) \
		host-uboot-tools-dirclean
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BR2_EXTERNAL) \
		BR2_DL_DIR=$(BR2_DL_DIR) \
		$(if $(BR2_CCACHE_DIR),BR2_CCACHE=y BR2_CCACHE_DIR=$(BR2_CCACHE_DIR),) \
		$(if $(GLASSOS_VERSION),GLASSOS_VERSION=$(GLASSOS_VERSION),) \
		-j$(shell nproc)

clean-all:
	rm -rf $(BUILDROOT)/output $(GLASS_DIST)

# Run agent unit tests
test-agent:
	@echo "==> Testing agent"
	@cd agent && go test ./...
	@echo "==> Done"

help:
	@echo ""
	@echo "GlassOS build targets:"
	@echo ""
	@echo "  build-rpi4/rpi5              Build SD card image for the given board (native host)"
	@echo "  docker-run-rpi4/rpi5         Build inside Docker (avoids arch/host-tool conflicts)"
	@echo "  docker-uboot-rebuild-rpi4/5  Force U-Boot recompile inside Docker"
	@echo "  uboot-rebuild-rpi4/rpi5      Force U-Boot recompile (use after uboot.config changes)"
	@echo "  build-agent                  Cross-compile the glass-agent binary (linux/arm64)"
	@echo "  download-glass               Download the glass binary (linux/arm64) from GitHub"
	@echo "  menuconfig-rpi4/rpi5         Open Buildroot ncurses config"
	@echo "  linux-menuconfig-rpi4/rpi5   Open kernel ncurses config"
	@echo "  savedefconfig-rpi4/rpi5      Save defconfig back to configs/"
	@echo "  docker-build                 Build the glassos-builder Docker image"
	@echo "  flash BOARD=X DEV=Y          Flash image to SD card device"
	@echo "        [SSID=x PSK=y]         Optionally write WiFi credentials (.nmconnection)"
	@echo "  clean-rpi4/rpi5              Remove build output for a board"
	@echo "  clean-all                    Remove all build output and downloaded binaries"
	@echo "  test-agent                   Run agent unit tests"
	@echo ""
	@echo "  Override BR2_DL_DIR and BR2_CCACHE_DIR to enable download/build caching."
	@echo "  Override GLASS_VERSION to download a different glass release (default: $(GLASS_VERSION))."
	@echo "  Override GLASS_VARIANT to select the Gio backend: wayland (default), x11, full."
	@echo "  Override GLASSOS_VERSION to set the OTA bundle version (default: from buildroot-external/meta)."
	@echo "  First build: ~90 min. Subsequent builds: ~5-10 min (with cache)."
	@echo ""
