################################################################################
#
# glass
#
################################################################################

GLASS_VERSION = $(call qstrip,$(BR2_PACKAGE_GLASS_VERSION))
GLASS_SITE    = https://github.com/glasslabs/looking-glass/releases/download/$(GLASS_VERSION)
GLASS_SOURCE  = glass-$(GLASS_VERSION)-linux-arm64.zip
GLASS_LICENSE = MIT

define GLASS_EXTRACT_CMDS
	unzip -o $(DL_DIR)/$(GLASS_SOURCE) -d $(@D)
endef

define GLASS_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/glass $(TARGET_DIR)/usr/bin/glass
endef

$(eval $(generic-package))

