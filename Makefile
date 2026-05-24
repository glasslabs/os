BOARDS       := rpi4 rpi5
BR2_EXTERNAL := $(CURDIR)/buildroot-external
BUILDROOT    := $(CURDIR)/buildroot
# Phony declarations up front.
.PHONY: $(addprefix build-,$(BOARDS))
.PHONY: $(addprefix menuconfig-,$(BOARDS))
.PHONY: $(addprefix linux-menuconfig-,$(BOARDS))
.PHONY: $(addprefix savedefconfig-,$(BOARDS))
.PHONY: $(addprefix clean-,$(BOARDS))
.PHONY: clean-all flash test-agent help
# build-rpi4 / build-rpi5 — static pattern rule avoids the GNU make 3.81
# limitation where explicit .PHONY entries shadow regular pattern rules.
$(addprefix build-,$(BOARDS)): build-%:
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BR2_EXTERNAL) \
		glassos_$*_defconfig
	$(MAKE) -C $(BUILDROOT) \
		O=$(BUILDROOT)/output/$* \
		BR2_EXTERNAL=$(BR2_EXTERNAL) \
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
# flash BOARD=rpi4 DEV=/dev/sdX [SSID=x PSK=y]
flash:
	@[ -n "$(BOARD)" ] || { echo "Usage: make flash BOARD=rpi4 DEV=/dev/sdX [SSID=x PSK=y]"; exit 1; }
	@[ -n "$(DEV)" ]   || { echo "Usage: make flash BOARD=rpi4 DEV=/dev/sdX [SSID=x PSK=y]"; exit 1; }
	@echo "==> Flashing $(BOARD) image to $(DEV) -- this will erase the device."
	@read -p "    Continue? [y/N] " confirm && [ "$$confirm" = "y" ]
	sudo dd if=$(BUILDROOT)/output/$(BOARD)/images/sdcard.img of=$(DEV) bs=4M conv=fsync status=progress
	sync
	@if [ -n "$(SSID)" ] && [ -n "$(PSK)" ]; then \
		echo "==> Writing wpa_supplicant.conf to boot partition"; \
		MOUNT=$$(mktemp -d); \
		sudo mount $(DEV)1 $$MOUNT; \
		printf 'country=GB\nctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev\nupdate_config=1\n\nnetwork={\n\tssid="$(SSID)"\n\tpsk="$(PSK)"\n}\n' \
			| sudo tee $$MOUNT/wpa_supplicant.conf > /dev/null; \
		sudo umount $$MOUNT && rmdir $$MOUNT; \
		echo "==> WiFi credentials written."; \
	fi
# clean-rpi4 /# clean-rpi4$(addprefix clean-,$(BOARDS)): clean-%:
	rm -rf $(BUILDROOT)/output/$*
clean-all:
	rm -rf $(BUILDROOT)/output
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
	@echo "  menuconfig-rpi4/rpi5         Open Buildroot ncurses config"
	@echo "  linux-menuconfig-rpi4/rpi5   Open kernel ncurses config"
	@echo "  savedefconfig-rpi4/rpi5      Save defconfig back to configs/"
	@echo "  flash BOARD=X DEV=Y          Flash image to SD card device"
	@echo "        [SSID=x PSK=y]         Optionally write WiFi credentials"
	@echo "  clean-rpi4/rpi5              Remove build output for a board"
	@echo "  clean-all                    Remove all build output"
	@echo "  test-agent                   Run agent unit tests"
	@echo ""
	@echo "  First build: ~90 min. Subsequent builds: ~5-10 min (with cache)."
	@echo ""
