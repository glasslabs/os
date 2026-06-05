################################################################################
#
# glass
#
################################################################################

GLASS_VERSION       = $(call qstrip,$(BR2_PACKAGE_GLASS_VERSION))
GLASS_SITE          = $(BR2_EXTERNAL_GLASSOS_PATH)/../glass/dist
GLASS_SITE_METHOD   = local
GLASS_LICENSE       = MIT


define GLASS_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/glass $(TARGET_DIR)/usr/lib/glass/glass
endef

$(eval $(generic-package))
