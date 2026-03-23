.PHONY: android android-apk android-debug-apk build clean help install-apk test

GO ?= go
BIN_DIR ?= bin
BINARY ?= pvta-tools
PKG ?= .
CGO_ENABLED ?= 0
LD_FLAGS ?= -s -w
GOFLAGS ?=
GOCACHE ?= /tmp/pvta-go-build
ANDROID_DIR ?= android
ANDROID_BRIDGE_SCRIPT ?= $(ANDROID_DIR)/scripts/build_bridge.sh
ANDROID_APP_DIR ?= $(ANDROID_DIR)
GRADLE ?= gradle
ANDROID_APK_TASK ?= assembleDebug
ANDROID_APK ?= $(ANDROID_APP_DIR)/app/build/outputs/apk/debug/app-debug.apk
ADB ?= adb
ANDROID_SDK_DIR ?= $(if $(ANDROID_HOME),$(ANDROID_HOME),$(HOME)/Library/Android/sdk)
ANDROID_LOCAL_PROPERTIES ?= $(ANDROID_APP_DIR)/local.properties

help:
	@printf '%s\n' \
		'pvta-tools make targets' \
		'' \
		'  make build' \
		'    Build bin/pvta-tools with the default settings.' \
		'' \
		'  make test' \
		'    Run go test ./... using a writable Go build cache.' \
		'' \
		'  make android' \
		'    Build android/app/libs/pvta-mobilebridge.aar via gomobile.' \
		'' \
		'  make android-apk' \
		'    Build the Android debug APK, generating the Go bridge AAR first.' \
		'' \
		'  make android-debug-apk' \
		'    Alias for make android-apk.' \
		'' \
		'  make install-apk' \
		'    Install the debug APK to a connected Android device via adb.' \
		'' \
		'  make clean' \
		'    Remove CLI build output plus Android build/cache artifacts.' \
		'' \
		'Variables:' \
		'  ANDROID_DIR  default: android' \
		'  ANDROID_APK  default: android/app/build/outputs/apk/debug/app-debug.apk' \
		'  ANDROID_APK_TASK default: assembleDebug' \
		'  ANDROID_SDK_DIR default: $$ANDROID_HOME or $$HOME/Library/Android/sdk' \
		'  ADB          default: adb' \
		'  BIN_DIR      default: bin' \
		'  BINARY       default: pvta-tools' \
		'  CGO_ENABLED  default: 0' \
		'  GOCACHE      default: /tmp/pvta-go-build' \
		'  GRADLE       default: gradle' \
		'  GOFLAGS      default: empty'

build:
	mkdir -p $(BIN_DIR)
	GOCACHE=$(GOCACHE) GOFLAGS=$(GOFLAGS) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -trimpath -ldflags "$(LD_FLAGS)" -o $(BIN_DIR)/$(BINARY) $(PKG)

test:
	GOCACHE=$(GOCACHE) GOFLAGS=$(GOFLAGS) $(GO) test ./...

android:
	GOCACHE=$(GOCACHE) GOFLAGS=$(GOFLAGS) $(ANDROID_BRIDGE_SCRIPT)

$(ANDROID_LOCAL_PROPERTIES):
	@mkdir -p $(ANDROID_APP_DIR)
	@printf 'sdk.dir=%s\n' "$(ANDROID_SDK_DIR)" > $(ANDROID_LOCAL_PROPERTIES)

android-apk: android $(ANDROID_LOCAL_PROPERTIES)
	cd $(ANDROID_APP_DIR) && $(GRADLE) $(ANDROID_APK_TASK)
	@printf '%s\n' 'APK: $(ANDROID_APK)'

android-debug-apk: android-apk

install-apk:
	@test -f $(ANDROID_APK) || (printf '%s\n' 'APK not found: $(ANDROID_APK)' 'Run make android-apk first.' >&2; exit 1)
	$(ADB) install -r $(ANDROID_APK)

clean:
	rm -rf $(BIN_DIR)
	rm -rf $(ANDROID_DIR)/.gradle $(ANDROID_DIR)/.kotlin
	rm -rf $(ANDROID_DIR)/app/build $(ANDROID_DIR)/build
	rm -f $(ANDROID_DIR)/app/libs/pvta-mobilebridge.aar $(ANDROID_DIR)/app/libs/pvta-mobilebridge-sources.jar
	rm -f $(ANDROID_LOCAL_PROPERTIES)
