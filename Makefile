BUILDDIR:=$(CURDIR)

BUILDROOT=$(BUILDDIR)/buildroot
BUILDROOT_EXTERNAL=$(BUILDDIR)/buildroot-external
DEFCONFIG_DIR = $(BUILDROOT_EXTERNAL)/configs

TARGETS := $(notdir $(patsubst %_defconfig,%,$(wildcard $(DEFCONFIG_DIR)/*_defconfig)))

DOCKER_IMAGE := glassos-builder

# Supervisor version — keep in sync with BR2_PACKAGE_SUPERVISOR_VERSION in the defconfigs.
GLASSOS_SUPERVISOR_VERSION ?= v0.1.1

# Glass version — keep in sync with BR2_PACKAGE_GLASS_VERSION in the defconfigs.
GLASS_VERSION ?= v2.0.5

# GlassOS image version — passed through to the RAUC bundle manifest and image
# file names.  Override on the command line (e.g. GLASSOS_VERSION=1.0.0) or let
# it fall back to the default defined in buildroot-external/meta.
GLASSOS_VERSION ?=

# Phony declarations up front.
.PHONY: $(addprefix build-,$(BOARDS))
.PHONY: $(addprefix menuconfig-,$(BOARDS))
.PHONY: $(addprefix linux-menuconfig-,$(BOARDS))
.PHONY: $(addprefix savedefconfig-,$(BOARDS))
.PHONY: $(addprefix clean-,$(BOARDS))
.PHONY: $(addprefix uboot-rebuild-,$(BOARDS))
.PHONY: clean-all docker-build enter help

$(addprefix build-,$(TARGETS)):
$(addprefix build-,$(TARGETS)): build-%:
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BUILDROOT_EXTERNAL) \
		BR2_DL_DIR=$(BUILDROOT)/dl \
		$*_defconfig
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BUILDROOT_EXTERNAL) \
		BR2_DL_DIR=$(BUILDROOT)/dl \
		$(if $(BR2_CCACHE_DIR),BR2_CCACHE=y BR2_CCACHE_DIR=$(BR2_CCACHE_DIR),) \
		$(if $(GLASSOS_VERSION),GLASSOS_VERSION=$(GLASSOS_VERSION),) \
		-j$(shell nproc)

$(addprefix menuconfig-,$(TARGETS)): menuconfig-%:
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BUILDROOT_EXTERNAL) \
		menuconfig

$(addprefix linux-menuconfig-,$(TARGETS)): linux-menuconfig-%:
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BUILDROOT_EXTERNAL) \
		linux-menuconfig

$(addprefix savedefconfig-,$(TARGETS)): savedefconfig-%:
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BUILDROOT_EXTERNAL) \
		BR2_DEFCONFIG=$(BUILDROOT_EXTERNAL)/configs/$*_defconfig \
		savedefconfig

docker-build:
	docker build -t $(DOCKER_IMAGE) .

enter:
	docker run --rm -it \
	    -v "$(CURDIR)":/build \
	    -v "glassos-output":/build/buildroot/output \
	    -v "glassos-ccache":/cache/cc \
	    -w /build \
	    $(DOCKER_IMAGE)

$(addprefix uboot-rebuild-,$(BOARDS)): uboot-rebuild-%:
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BUILDROOT_EXTERNAL) \
		uboot-dirclean
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BUILDROOT_EXTERNAL) \
		host-uboot-tools-dirclean
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BUILDROOT_EXTERNAL) \
		BR2_DL_DIR=$(BUILDROOT)/dl \
		$(if $(BR2_CCACHE_DIR),BR2_CCACHE=y BR2_CCACHE_DIR=$(BR2_CCACHE_DIR),) \
		$(if $(GLASSOS_VERSION),GLASSOS_VERSION=$(GLASSOS_VERSION),) \
		-j$(shell nproc)

$(addprefix clean-,$(TARGETS)): clean-%:
	rm -rf $(BUILDROOT)/output/$*

clean-all:
	rm -rf $(BUILDROOT)/output $(GLASS_DIST)

help:
	@echo ""
	@echo "GlassOS build targets:"
	@echo ""
	@echo "  build-<target>               Build SD card image for the given board (native host / inside shell)"
	@echo "  menuconfig-<target>          Open Buildroot ncurses config"
	@echo "  linux-menuconfig-<target>    Open kernel ncurses config"
	@echo "  savedefconfig-<target>       Save defconfig back to configs/"
	@echo "  uboot-rebuild-<target>       Force U-Boot recompile (use after uboot.config changes)"
	@echo "  docker-build                 Build the glassos-builder Docker image"
	@echo "  enter                        Drop into a build shell inside Docker (local development)"
	@echo "  clean-<target>               Remove build output for a board"
	@echo "  clean-all                    Remove all build output"
	@echo ""
	@echo "  Supported targets: $(TARGETS)"
	@echo ""
	@echo "  Override GLASSOS_SUPERVISOR_VERSION to use a different supervisor release (default: $(GLASSOS_SUPERVISOR_VERSION))."
	@echo "  Override GLASS_VERSION to use a different glass release (default: $(GLASS_VERSION))."
	@echo "  Override GLASSOS_VERSION to set the OTA bundle version (default: from buildroot-external/meta)."
	@echo ""
