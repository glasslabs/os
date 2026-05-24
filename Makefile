BOARDS       := rpi4 rpi5
BR2_EXTERNAL := $(CURDIR)/buildroot-external
BUILDROOT    := $(CURDIR)/buildroot
AGENT_DIST   := $(CURDIR)/agent/dist
GLASS_DIST   := $(CURDIR)/glass/dist

# Glass version — keep in sync with BR2_PACKAGE_GLASS_VERSION in the defconfigs.
GLASS_VERSION ?= v2.0.0

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
	@echo "==> Downloading glass $(GLASS_VERSION) (linux/arm64)"
	@mkdir -p $(GLASS_DIST)
	@curl -fsSL \
		"https://github.com/glasslabs/looking-glass/releases/download/$(GLASS_VERSION)/glass-$(GLASS_VERSION)-linux-arm64.zip" \
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

# flash BOARD=rpi4 DEV=/dev/sdX [SSID=x PSK=y]
flash:
	@[ -n "$(BOARD)" ] || { echo "Usage: make flash BOARD=rpi4 DEV=/dev/sdX [SSID=x PSK=y]"; exit 1; }
	@[ -n "$(DEV)" ]   || { echo "Usage: make flash BOARD=rpi4 DEV=/dev/sdX [SSID=x PSK=y]"; exit 1; }
	@echo "==> Flashing $(BOARD) image to $(DEV) -- this will erase the device."
	@read -p "    Continue? [y/N] " confirm && [ "$$confirm" = "y" ]
	sudo dd if=$(BUILDROOT)/output/$(BOARD)/images/sdcard.img of=$(DEV) bs=4M conv=fsync status=progress
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
	@echo "  build-rpi4/rpi5              Build SD card image for the given board"
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
	@echo "  First build: ~90 min. Subsequent builds: ~5-10 min (with cache)."
	@echo ""
