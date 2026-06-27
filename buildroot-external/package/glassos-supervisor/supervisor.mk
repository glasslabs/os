################################################################################
#
# supervisor
#
################################################################################

GLASSOS_SUPERVISOR_VERSION     = $(call qstrip,$(BR2_PACKAGE_GLASSOS_SUPERVISOR_VERSION))
GLASSOS_SUPERVISOR_SITE        = https://github.com/glasslabs/supervisor/releases/download/$(GLASSOS_SUPERVISOR_VERSION)
GLASSOS_SUPERVISOR_SOURCE      = supervisor_$(GLASSOS_SUPERVISOR_VERSION:v%=%)_linux_arm64.tar.gz
GLASSOS_SUPERVISOR_LICENSE     = MIT

define GLASSOS_SUPERVISOR_EXTRACT_CMDS
    $(TAR) --strip-components=0 -C $(@D) -xf $(GLASSOS_SUPERVISOR_DL_DIR)/$(GLASSOS_SUPERVISOR_SOURCE)
endef

define GLASSOS_SUPERVISOR_INSTALL_TARGET_CMDS
    $(INSTALL) -D -m 0755 $(@D)/supervisor $(TARGET_DIR)/usr/bin/supervisor
endef

$(eval $(generic-package))
