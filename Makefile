# 顶层 Makefile：汇总各子 make 文件
MAKEFILES_DIR := make-files

# BACKEND 参数：gin（默认）| fastapi
# 例如：make BACKEND=fastapi test-smoke

include $(MAKEFILES_DIR)/common.mk
include $(MAKEFILES_DIR)/build.mk
include $(MAKEFILES_DIR)/test.mk
include $(MAKEFILES_DIR)/migrate.mk
include $(MAKEFILES_DIR)/dev.mk
include $(MAKEFILES_DIR)/docker.mk
include $(MAKEFILES_DIR)/project.mk
include $(MAKEFILES_DIR)/security.mk
include $(MAKEFILES_DIR)/docs.mk
include $(MAKEFILES_DIR)/manifest.mk
include $(MAKEFILES_DIR)/compat.mk
include $(MAKEFILES_DIR)/release.mk
include $(MAKEFILES_DIR)/ci.mk
