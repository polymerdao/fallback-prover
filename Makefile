BUILD_DIR=./bin
BINARY_NAME=native-proof
MODULE_NAME=github.com/polymerdao/fallback_prover

.PHONY: build clean run

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd

clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)

run:
	go run ./cmd/main.go

test:
	go test ./...

all: clean build

help:
	@echo "Available targets:"
	@echo "  build      - Build the binary to $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "  clean      - Remove the binary"
	@echo "  run        - Run the application without building"
	@echo "  test       - Run tests"
	@echo "  all        - Clean and build"
	@echo ""
	@echo "Usage example:"
	@echo "  ./$(BUILD_DIR)/$(BINARY_NAME) prove --src-l2-chain-id 10 --dst-l2-chain-id 42161 \\"
	@echo "    --src-l2-rpc https://mainnet.optimism.io \\"
	@echo "    --dst-l2-rpc https://arb1.arbitrum.io/rpc \\"
	@echo "    --src-address 0x1234567890abcdef1234567890abcdef12345678 \\"
	@echo "    --src-storage-slot 0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890 \\"
	@echo "    --l1-rpc https://ethereum.publicnode.com"
