################################################################################
#
# glass-agent
#
################################################################################

GLASS_AGENT_VERSION     = 1.0.0
GLASS_AGENT_SITE        = $(BR2_EXTERNAL_GLASSOS_PATH)/../agent/dist
GLASS_AGENT_SITE_METHOD = local
GLASS_AGENT_LICENSE     = MIT

define GLASS_AGENT_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/glass-agent $(TARGET_DIR)/usr/bin/glass-agent
endef

$(eval $(generic-package))
