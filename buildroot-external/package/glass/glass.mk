################################################################################
#
# glass
#
################################################################################

GLASS_VERSION       = $(call qstrip,$(BR2_PACKAGE_GLASS_VERSION))
GLASS_SITE          = https://github.com/glasslabs/looking-glass/releases/download/$(GLASS_VERSION)
GLASS_SOURCE        = glass_$(GLASS_VERSION:v%=%)_linux_arm64_wayland.tar.gz
GLASS_LICENSE       = MIT

define GLASS_EXTRACT_CMDS
    $(TAR) --strip-components=0 -C $(@D) -xf $(GLASS_DL_DIR)/$(GLASS_SOURCE)
endef

define GLASS_INSTALL_TARGET_CMDS
    $(INSTALL) -D -m 0755 $(@D)/glass $(TARGET_DIR)/usr/lib/glass/glass
endef

$(eval $(generic-package))
