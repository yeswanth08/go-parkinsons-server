# GEN FILE 
# using phony for exporting and importing the cmds
.PHONY: stubs_gen install_tools gen_parkinsons_stubs clean_stubs server_gen gen_sdk

PROTO_DIR=proto
STUBS_GEN_DIR=gen-stubs
PARKINSONS_PROTO_FILE=$(wildcard $(PROTO_DIR)/*.proto)
SDK_OUT_DIR=sdk

stubs_gen: gen_parkinsons_stubs

install_tools: 
	cd tools && $(MAKE) install_tools

gen_parkinsons_stubs:
	mkdir -p $(STUBS_GEN_DIR)
	protoc \
	--proto_path=$(PROTO_DIR) \
	--go_out=$(STUBS_GEN_DIR) --go_opt=paths=source_relative \
	--go-grpc_out=$(STUBS_GEN_DIR) --go-grpc_opt=paths=source_relative \
	$(PARKINSONS_PROTO_FILE)

server_gen:
	mkdir -p internal/api/gen 
	cd open-api && oapi-codegen -config oapi-config.yaml openapi.yaml

sdk_gen:
	mkdir -p $(SDK_OUT_DIR)
	npx openapi-typescript-codegen \
		--input open-api/openapi.yaml \
		--output $(SDK_OUT_DIR) \
		--client fetch \
		--useOptions \
		--exportSchemas true
	cp open-api/sdk-package.json $(SDK_OUT_DIR)/package.json

clean_stubs:
	rm -rf $(STUBS_GEN_DIR)

clean_sdk:
	rm -rf $(SDK_OUT_DIR)