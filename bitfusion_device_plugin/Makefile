# Image URL to use all building/pushing image targets;
# Use your own docker registry and image name for dev/test by overridding the
# IMAGE_REPO, DEVICE_IMAGE_NAME, WEBHOOK_IMAGE_NAME, PKG_IMAGE_NAME and IMAGE_TAG environment variable.
IMAGE_REPO ?= docker.io/bitfusiondeviceplugin
DEVICE_IMAGE_NAME ?= bitfusion-device-plugin
WEBHOOK_IMAGE_NAME ?= bitfusion-webhook
PKG_IMAGE_NAME ?= bitfusion-client
IMAGE_TAG  ?= 0.1



PWD := $(shell pwd)
BASE_DIR := $(shell basename $(PWD))

IMAGE_TAG ?= $(shell date +v%Y%m%d)-$(shell git describe --match=$(git rev-parse --short=8 HEAD) --tags --always --dirty)

DEVICE_PLUGIN_ORIGINAL_NAME := phaedobf/device-plugin:v0.1
BF_CLIENT_ORIGINAL_NAME := phaedobf/bitfusion-client-ubuntu1804_2.5.0-10_amd64:v0.1
WEBHOOK_ORIGINAL_NAME := phaedobf/sidecar-injector:v0.1

DEVICE_PLUGIN_YAML_FILE := device-plugin/deployment/device_plugin.yml
CONFIGMAP_YAML_FILE := webhook/deployment/bitfusion_injector_webhook_configmap.yaml
WEBHOOK_YAML_FILE := webhook/deployment/bitfusion_injector.yaml

DEVICE_PLUGIN_YAML_TO_DEPLOY := device-plugin/deployment/deploy_device_plugin.yml
CONFIGMAP_YAML_TO_DEPLOY := webhook/deployment/deploy_bitfusion_injector_webhook_configmap.yaml
WEBHOOK_YAML_TO_DEPLOY := webhook/deployment/deploy_bitfusion_injector.yaml

LOCAL_OS := $(shell uname)
ifeq ($(LOCAL_OS),Linux)
    TARGET_OS ?= linux
    XARGS_FLAGS="-r"
else ifeq ($(LOCAL_OS),Darwin)
    TARGET_OS ?= darwin
    XARGS_FLAGS=
else
    $(error "This system's OS $(LOCAL_OS) isn't recognized/supported")
endif


############################################################
# image section
############################################################

image: build-image push-image

build-image: 
	@echo "Building the docker image: $(IMAGE_REPO)/$(DEVICE_IMAGE_NAME):$(IMAGE_TAG)..."
	@docker build -t $(IMAGE_REPO)/$(DEVICE_IMAGE_NAME):$(IMAGE_TAG) -f device-plugin/build/Dockerfile .
	@echo "Building the docker image: $(IMAGE_REPO)/$(WEBHOOK_IMAGE_NAME):$(IMAGE_TAG)..."
	@docker build -t $(IMAGE_REPO)/$(WEBHOOK_IMAGE_NAME):$(IMAGE_TAG) -f webhook/build/Dockerfile .
	@echo "Building the docker image: $(IMAGE_REPO)/$(PKG_IMAGE_NAME):$(IMAGE_TAG)..."
	@docker build -t $(IMAGE_REPO)/$(PKG_IMAGE_NAME):$(IMAGE_TAG) -f webhook/build/Dockerfile-bitfusion-client-ubuntu1804_2.5.0-10_amd64 .

push-image: build-image
	@echo "Pushing the docker image for $(IMAGE_REPO)/$(DEVICE_IMAGE_NAME):$(IMAGE_TAG) ..."
	@docker push $(IMAGE_REPO)/$(DEVICE_IMAGE_NAME):$(IMAGE_TAG)
	@echo "Pushing the docker image for $(IMAGE_REPO)/$(WEBHOOK_IMAGE_NAME):$(IMAGE_TAG) ..."
	@docker push $(IMAGE_REPO)/$(WEBHOOK_IMAGE_NAME):$(IMAGE_TAG)
	@echo "Pushing the docker image for $(IMAGE_REPO)/$(PKG_IMAGE_NAME):$(IMAGE_TAG) ..."
	@docker push $(IMAGE_REPO)/$(PKG_IMAGE_NAME):$(IMAGE_TAG)

############################################################
# deploy section
############################################################

deploy: update create 

update:
	@echo "Create file $(DEVICE_PLUGIN_YAML_TO_DEPLOY) by $(DEVICE_PLUGIN_YAML_FILE): rewrite $(DEVICE_PLUGIN_ORIGINAL_NAME) to $(IMAGE_REPO)/$(DEVICE_IMAGE_NAME):$(IMAGE_TAG) ..."
	@cat $(DEVICE_PLUGIN_YAML_FILE) | sed -e "s|$(DEVICE_PLUGIN_ORIGINAL_NAME)|$(IMAGE_REPO)/$(DEVICE_IMAGE_NAME):$(IMAGE_TAG)|g" > $(DEVICE_PLUGIN_YAML_TO_DEPLOY)
	@echo "Create file $(CONFIGMAP_YAML_TO_DEPLOY) by $(CONFIGMAP_YAML_FILE): rewrite $(BF_CLIENT_ORIGINAL_NAME) to $(IMAGE_REPO)/$(PKG_IMAGE_NAME):$(IMAGE_TAG) ..."
	@cat $(CONFIGMAP_YAML_FILE) | sed -e "s|$(BF_CLIENT_ORIGINAL_NAME)|$(IMAGE_REPO)/$(PKG_IMAGE_NAME):$(IMAGE_TAG)|g" > $(CONFIGMAP_YAML_TO_DEPLOY)
	@echo "Create file $(WEBHOOK_YAML_TO_DEPLOY) by $(WEBHOOK_YAML_FILE): rewrite $(WEBHOOK_ORIGINAL_NAME) to $(IMAGE_REPO)/$(WEBHOOK_IMAGE_NAME):$(IMAGE_TAG) ..."
	@cat $(WEBHOOK_YAML_FILE) | sed -e "s|$(WEBHOOK_ORIGINAL_NAME)|$(IMAGE_REPO)/$(WEBHOOK_IMAGE_NAME):$(IMAGE_TAG)|g" > $(WEBHOOK_YAML_TO_DEPLOY)

create: update
	@echo "Create device-plugin ..."
	@kubectl apply -f $(DEVICE_PLUGIN_YAML_TO_DEPLOY)
	@echo "Create webhook ..."
	@bash webhook/deploy.sh
