# GEN FILE 
# using phony for exporting and importing the cmds
.PHONY: stubs_gen install_tools gen_parkinsons_stubs clean_stubs

PROTO_DIR=proto
STUBS_GEN_DIR=gen-stubs
PARKINSONS_PROTO_FILE=$(wildcard $(PROTO_DIR)/*.proto)

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


clean_stubs:
	rm -rf $(STUBS_GEN_DIR)