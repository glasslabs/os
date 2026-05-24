################################################################################
#
# glass-agent
#
################################################################################

GLASS_AGENT_VERSION = 1.0.0
GLASS_AGENT_SITE    = $(BR2_EXTERNAL_GLASSOS_PATH)/agent
GLASS_AGENT_SITE_METHOD = local
GLASS_AGENT_LICENSE = MIT

GLASS_AGENT_BUILD_TARGETS = .
GLASS_AGENT_INSTALL_BINS  = glass-agent

$(eval $(golang-package))

