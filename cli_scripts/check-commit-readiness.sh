#!/bin/bash
set -e

# Required variables
VERSION_FILE="/opt/gopath/src/github.com/hyperledger/fabric/peer/contracts/cc-version.txt"
ORDERER_ADDRESS="orderer.cbn.naijachain.org:7050"
ORDERER_CA="/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/cbn.naijachain.org/orderers/orderer.cbn.naijachain.org/msp/tlscacerts/tlsca.cbn.naijachain.org-cert.pem"
CHANNEL_NAME="retailchannel"
CHAINCODE_NAME="account"

# Extract chaincode metadata
CC_VERSION=$(sed -n '1p' "$VERSION_FILE")
CC_SEQUENCE_NUM=$(grep SEQUENCE_NUM "$VERSION_FILE" | cut -d'=' -f2)

echo "🔍 Checking commit readiness for chaincode $CHAINCODE_NAME..."
peer lifecycle chaincode checkcommitreadiness \
    -o "$ORDERER_ADDRESS" \
    --tls true \
    --cafile "$ORDERER_CA" \
    -C "$CHANNEL_NAME" \
    -n "$CHAINCODE_NAME" \
    --version "$CC_VERSION" \
    --sequence "$CC_SEQUENCE_NUM" \
    --output json