CLIENT_PATH := gitlab.alipay-inc.com/sigma/clientset
CLIENT_APIS := $(CLIENT_PATH)/pkg/apis
PATHS := $(shell find pkg/apis/ -type d -mindepth 2 -maxdepth 2 2>/dev/null)
SPACE := 
SPACE += 
SEP := ,
PKGS :=  $(subst $(SPACE),$(SEP),$(shell echo $(PATHS) | sed 's|pkg/apis/*||g'))
FULL_PKGS :=  $(subst $(SPACE),$(SEP),$(shell echo $(PATHS) | sed 's|pkg/apis/*|$(CLIENT_APIS)/|g'))
BINS := client-gen conversion-gen deepcopy-gen defaulter-gen informer-gen lister-gen register-gen
BINS_DIR := cmd/tmpbin



all: gen

clean:
	@rm -rf informers kubernetes listers
	@find ./pkg/apis -maxdepth 3 -mindepth 3 -name 'zz_generated.*.go' -exec rm -f {} \;
	@rm -rf $(BINS_DIR)

build:
	@echo "Building generators"
	@for bin in $(BINS); do \
		go build -o $(BINS_DIR)/$$bin ./cmd/$$bin; \
		done

test:
	@go test ./...

gen: clean build
	@mkdir -p expansions/{listers,kubernetes,informers}
	@cp -r expansions/* ./
	@echo APIs: $(PKGS)
	@$(BINS_DIR)/conversion-gen -i $(FULL_PKGS) -O "zz_generated.conversion"
	@echo "Generated conversion"
	@$(BINS_DIR)/defaulter-gen -i $(FULL_PKGS) -O "zz_generated.defaults"
	@echo "Generated defaulter"
	@$(BINS_DIR)/deepcopy-gen -i $(FULL_PKGS) -O "zz_generated.deepcopy"
	@echo "Generated register"
	@$(BINS_DIR)/register-gen -i $(FULL_PKGS) -O "zz_generated.register"
	@echo "Generated deepcopy"
	@$(BINS_DIR)/client-gen -n kubernetes --clientset-path $(CLIENT_PATH) --input-base $(CLIENT_APIS) --input $(PKGS)
	@echo "Generated clients"
	@$(BINS_DIR)/lister-gen -p $(CLIENT_PATH)/listers -i $(FULL_PKGS)
	@echo "Generated listers"
	@$(BINS_DIR)/informer-gen --single-directory -p $(CLIENT_PATH)/informers --versioned-clientset-package \
	       $(CLIENT_PATH)/kubernetes --listers-package $(CLIENT_PATH)/listers -i $(FULL_PKGS)
	@echo "Generated informers"
	@rm -rf $(BINS_DIR)
 
