#!/bin/bash
set -e

# Required variables
VERSION_FILE="/opt/gopath/src/github.com/hyperledger/fabric/peer/contracts/cc-version.txt"
ORDERER_ADDRESS="orderer.cbn.naijachain.org:7050"
ORDERER_CA="/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/cbn.naijachain.org/orderers/orderer.cbn.naijachain.org/msp/tlscacerts/tlsca.cbn.naijachain.org-cert.pem"
CHANNEL_NAME="retailchannel"
CHAINCODE_NAME="account"
CRYPTO_PATH="/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto"

# Extract chaincode metadata
CC_VERSION=$(sed -n '1p' "$VERSION_FILE")
CC_SEQUENCE_NUM=$(grep SEQUENCE_NUM "$VERSION_FILE" | cut -d'=' -f2)

echo "🔐 Committing chaincode definition to the channel..."
peer lifecycle chaincode commit \
    -o "$ORDERER_ADDRESS" \
    --tls true \
    --cafile "$ORDERER_CA" \
    -C "$CHANNEL_NAME" \
    -n "$CHAINCODE_NAME" \
    --version "$CC_VERSION" \
    --sequence "$CC_SEQUENCE_NUM" \
    --peerAddresses peer0.accessbank.naijachain.org:7051 \
    --tlsRootCertFiles $CRYPTO_PATH/peerOrganizations/accessbank.naijachain.org/peers/peer0.accessbank.naijachain.org/tls/ca.crt \
    --peerAddresses peer0.gtbank.naijachain.org:8051 \
    --tlsRootCertFiles $CRYPTO_PATH/peerOrganizations/gtbank.naijachain.org/peers/peer0.gtbank.naijachain.org/tls/ca.crt \
    --peerAddresses peer0.zenithbank.naijachain.org:9051 \
    --tlsRootCertFiles $CRYPTO_PATH/peerOrganizations/zenithbank.naijachain.org/peers/peer0.zenithbank.naijachain.org/tls/ca.crt

echo "🎉 Chaincode committed successfully!"

# Query the committed chaincode
echo "🔍 Querying committed chaincode on the channel..."
peer lifecycle chaincode querycommitted \
    -o "$ORDERER_ADDRESS" \
    --tls true \
    --cafile "$ORDERER_CA" \
    -C "$CHANNEL_NAME" \
    -n "$CHAINCODE_NAME"