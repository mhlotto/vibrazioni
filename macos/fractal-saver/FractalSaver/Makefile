PROJECT := FractalSaver
SCHEME := FractalSaver
CONFIGURATION ?= Release
DERIVED_DATA := $(CURDIR)/build/DerivedData
PRODUCT := $(DERIVED_DATA)/Build/Products/$(CONFIGURATION)/$(PROJECT).saver
INSTALL_DIR ?= $(HOME)/Library/Screen Savers

.PHONY: build clean install

build:
	xcodebuild -project $(PROJECT).xcodeproj -scheme $(SCHEME) -configuration $(CONFIGURATION) -derivedDataPath "$(DERIVED_DATA)" build
	@echo "Built: $(PRODUCT)"

clean:
	xcodebuild -project $(PROJECT).xcodeproj -scheme $(SCHEME) -configuration $(CONFIGURATION) -derivedDataPath "$(DERIVED_DATA)" clean
	rm -rf "$(DERIVED_DATA)"
	@echo "Cleaned: $(DERIVED_DATA)"

install: build
	mkdir -p "$(INSTALL_DIR)"
	rm -rf "$(INSTALL_DIR)/$(PROJECT).saver"
	cp -R "$(PRODUCT)" "$(INSTALL_DIR)/$(PROJECT).saver"
	@echo "Installed: $(INSTALL_DIR)/$(PROJECT).saver"
