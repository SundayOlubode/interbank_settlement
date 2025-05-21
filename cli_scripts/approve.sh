#!/bin/bash
set -e

# === Required variables ===
VERSION_FILE="/opt/gopath/src/github.com/hyperledger/fabric/peer/contracts/cc-version.txt"
CHAINCODE_PATH="/opt/gopath/src/github.com/hyperledger/fabric/peer/contracts"
ORDERER_ADDRESS="orderer.cbn.naijachain.org:7050"
ORDERER_CA="/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/cbn.naijachain.org/orderers/orderer.cbn.naijachain.org/msp/tlscacerts/tlsca.cbn.naijachain.org-cert.pem"
CHANNEL_NAME="retailchannel"
CHAINCODE_NAME="account"

# === Path to crypto materials ===
CRYPTO_PATH="/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto"

# === Extract chaincode metadata ===
CC_VERSION=$(sed -n '1p' "$VERSION_FILE")
CC_SEQUENCE_NUM=$(grep SEQUENCE_NUM "$VERSION_FILE" | cut -d'=' -f2)
CC_PACKAGE_PATH="$CHAINCODE_PATH/cc-packages/$(grep PACKAGE_NAME "$VERSION_FILE" | cut -d'=' -f2)"

# === Calculate package ID ===
echo "🔍 Calculating package ID..."
CC_PACKAGE_ID=$(peer lifecycle chaincode calculatepackageid "$CC_PACKAGE_PATH")
echo "📦 PACKAGE_ID: $CC_PACKAGE_ID"

# === Function to approve chaincode for current organization ===
approve_for_myorg() {
    echo "✅ Approving chaincode for $CORE_PEER_LOCALMSPID..."
    
    peer lifecycle chaincode approveformyorg \
        -o "$ORDERER_ADDRESS" \
        --tls true \
        --cafile "$ORDERER_CA" \
        -C "$CHANNEL_NAME" \
        -n "$CHAINCODE_NAME" \
        --version "$CC_VERSION" \
        --sequence "$CC_SEQUENCE_NUM" \
        --package-id "$CC_PACKAGE_ID"
    
    echo "🎉 Chaincode approved for $CORE_PEER_LOCALMSPID!"
}

# === Execute approval for current organization ===
approve_for_myorg

echo "✅ Approval process completed!"