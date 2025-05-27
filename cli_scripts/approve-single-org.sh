#!/bin/bash
set -e

# Required variables
VERSION_FILE="/opt/gopath/src/github.com/hyperledger/fabric/peer/contracts/cc-version.txt"
CHAINCODE_PATH="/opt/gopath/src/github.com/hyperledger/fabric/peer/contracts"
ORDERER_ADDRESS="orderer.cbn.naijachain.org:7050"
# ORDERER_ADDRESS="127.0.0.1:7050"
ORDERER_CA="/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/cbn.naijachain.org/tlsca/tlsca.cbn.naijachain.org-cert.pem"
CHANNEL_NAME="retailchannel"
CHAINCODE_NAME="account"
ORDERER_HOST="orderer.cbn.naijachain.org"

set -x

# 1. pick the org name out of the peer address:  peer0.accessbank.naijachain.org → accessbank
ORG=$(echo "$CORE_PEER_ADDRESS" | cut -d'.' -f2)

# 2. switch the CLI to the org-admin MSP (leave the peer daemon untouched)
export CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/${ORG}.naijachain.org/users/Admin@${ORG}.naijachain.org/msp
export CORE_PEER_LOCALMSPID=${CORE_PEER_LOCALMSPID}        # AccessBank → AccessbankMSP, FirstBank → FirstbankMSP, …

# 3. keep the TLS root cert that matches the orderer
export CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/${ORG}.naijachain.org/peers/peer0.${ORG}.naijachain.org/tls/ca.crt


# CORE_PEER_TLS_ROOTCERT_FILE="$PWD/peer/crypto/peerOrganizations/$(echo $CORE_PEER_ADDRESS | cut -d'.' -f2).naijachain.org/tlsca/tlsca.$(echo $CORE_PEER_ADDRESS | cut -d'.' -f2).naijachain.org-cert.pem"

# echo "TLS ROOT CERT FILE $CORE_PEER_TLS_ROOTCERT_FILE"

# export CORE_PEER_TLS_ROOTCERT_FILE="$CORE_PEER_TLS_ROOTCERT_FILE"

# Extract chaincode metadata
CC_VERSION=$(sed -n '1p' "$VERSION_FILE")
CC_SEQUENCE_NUM=$(grep SEQUENCE_NUM "$VERSION_FILE" | cut -d'=' -f2)
CC_PACKAGE_PATH="$CHAINCODE_PATH/cc-packages/$(grep PACKAGE_NAME "$VERSION_FILE" | cut -d'=' -f2)"
CC_LABEL=$(grep LABEL "$VERSION_FILE" | cut -d'=' -f2)

# Calculate package ID
echo "🔍 Calculating package ID..."
CC_PACKAGE_ID=$(peer lifecycle chaincode calculatepackageid "$CC_PACKAGE_PATH")
echo "📦 PACKAGE_ID: $CC_PACKAGE_ID"

# Print current org for verification
echo "🏦 Current organization: $CORE_PEER_LOCALMSPID"
echo "🖥️ Current peer: $CORE_PEER_ADDRESS"

# Approve chaincode definition for this org only
echo "✅ Approving chaincode definition for $CORE_PEER_LOCALMSPID..."


# peer lifecycle chaincode approveformyorg \
#     -o "$ORDERER_ADDRESS" \
#     --tls true \
#     --cafile "$ORDERER_CA" \
#     -C "$CHANNEL_NAME" \
#     -n "$CHAINCODE_NAME" \
#     --version "$CC_VERSION" \
#     --sequence "$CC_SEQUENCE_NUM" \
#     --package-id "$CC_PACKAGE_ID"

peer lifecycle chaincode approveformyorg \
  --connTimeout 120s \
  -o $ORDERER_ADDRESS \
  --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
  --channelID $CHANNEL_NAME \
  --name $CHAINCODE_NAME \
  --version $CC_VERSION \
  --sequence $CC_SEQUENCE_NUM \
  --package-id $CC_PACKAGE_ID \
  --tls true \
  --cafile $ORDERER_CA \
  # --waitForEvent --waitForEventTimeout 30s \
  # --signature-policy "OR('$CORE_PEER_LOCALMSPID.peer')"

set +x

echo "🎉 Chaincode approved for $CORE_PEER_LOCALMSPID!"

# /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/zenithbank.naijachain.org/tlca/tlsca.zenithbank.naijachain.org-cert.pem
# /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/zenithbank.naijachain.org/tlsca/tlsca.zenithbank.naijachain.org-cert.pem