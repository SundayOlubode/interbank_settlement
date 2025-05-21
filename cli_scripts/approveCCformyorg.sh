#!/bin/bash
set -e

# === Required variables ===
VERSION_FILE="/opt/gopath/src/github.com/hyperledger/fabric/peer/contracts/cc-version.txt"
CHAINCODE_PATH="/opt/gopath/src/github.com/hyperledger/fabric/peer/contracts"
ORDERER_ADDRESS="orderer.cbn.naijachain.org:7050"
ORDERER_CA="/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/cbn.naijachain.org/orderers/orderer.cbn.naijachain.org/msp/tlscacerts/tlsca.cbn.naijachain.org-cert.pem"
CHANNEL_NAME="retailchannel"
CHAINCODE_NAME="account"

# === TLS Certificate paths for peers ===
ACCESS_BANK_TLS_CERT="/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/accessbank.naijachain.org/peers/peer0.accessbank.naijachain.org/tls/ca.crt"
GT_BANK_TLS_CERT="/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/gtbank.naijachain.org/peers/peer0.gtbank.naijachain.org/tls/ca.crt"
ZENITH_BANK_TLS_CERT="/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/zenithbank.naijachain.org/peers/peer0.zenithbank.naijachain.org/tls/ca.crt"
FIRST_BANK_TLS_CERT="/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/firstbank.naijachain.org/peers/peer0.firstbank.naijachain.org/tls/ca.crt"

# === Pre-checks ===
: "${CORE_PEER_ADDRESS:?❌ Please set CORE_PEER_ADDRESS before running this script.}"
: "${CORE_PEER_LOCALMSPID:?❌ Please set CORE_PEER_LOCALMSPID before running this script.}"

# === Extract chaincode metadata ===
CC_VERSION=$(sed -n '1p' "$VERSION_FILE")
CC_LABEL=$(grep LABEL "$VERSION_FILE" | cut -d'=' -f2)
CC_PACKAGE_NAME=$(grep PACKAGE_NAME "$VERSION_FILE" | cut -d'=' -f2)
CC_SEQUENCE_NUM=$(grep SEQUENCE_NUM "$VERSION_FILE" | cut -d'=' -f2)
CC_PACKAGE_PATH="$CHAINCODE_PATH/cc-packages/$CC_PACKAGE_NAME"

# === Calculate package ID ===
echo "🔍 Calculating package ID..."
CC_PACKAGE_ID=$(peer lifecycle chaincode calculatepackageid "$CC_PACKAGE_PATH")
echo "📦 PACKAGE_ID: $CC_PACKAGE_ID"

# === Function to handle chaincode approval ===
approve_chaincode() {
    local org_msp=$1
    local peer_address=$2
    
    echo "🏦 Setting environment for $org_msp at $peer_address..."
    export CORE_PEER_LOCALMSPID=$org_msp
    export CORE_PEER_ADDRESS=$peer_address
    
    echo "✅ Approving chaincode version $CC_VERSION with sequence $CC_SEQUENCE_NUM for $org_msp..."
    peer lifecycle chaincode approveformyorg \
        --connTimeout 120s \
        -o "$ORDERER_ADDRESS" \
        --tls "$CORE_PEER_TLS_ENABLED" \
        --cafile "$ORDERER_CA" \
        -C "$CHANNEL_NAME" \
        -n "$CHAINCODE_NAME" \
        --version "$CC_VERSION" \
        --sequence "$CC_SEQUENCE_NUM" \
        --package-id "$CC_PACKAGE_ID"
    
    echo "🎉 Chaincode approved for $org_msp!"
}

# === Function to check the commit readiness ===
check_commit_readiness() {
    echo "🔍 Checking commit readiness for chaincode $CHAINCODE_NAME..."
    peer lifecycle chaincode checkcommitreadiness \
        -o "$ORDERER_ADDRESS" \
        --tls "$CORE_PEER_TLS_ENABLED" \
        --cafile "$ORDERER_CA" \
        -C "$CHANNEL_NAME" \
        -n "$CHAINCODE_NAME" \
        --version "$CC_VERSION" \
        --sequence "$CC_SEQUENCE_NUM" \
        --output json
}

# === Function to commit the chaincode definition ===
commit_chaincode() {
    echo "🔐 Committing chaincode definition to the channel..."
    peer lifecycle chaincode commit \
        --connTimeout 120s \
        -o "$ORDERER_ADDRESS" \
        --tls "$CORE_PEER_TLS_ENABLED" \
        --cafile "$ORDERER_CA" \
        -C "$CHANNEL_NAME" \
        -n "$CHAINCODE_NAME" \
        --version "$CC_VERSION" \
        --sequence "$CC_SEQUENCE_NUM" \
        --peerAddresses peer0.accessbank.naijachain.org:7051 \
        --tlsRootCertFiles "$ACCESS_BANK_TLS_CERT" \
        --peerAddresses peer0.gtbank.naijachain.org:8051 \
        --tlsRootCertFiles "$GT_BANK_TLS_CERT" \
        --peerAddresses peer0.zenithbank.naijachain.org:9051 \
        --tlsRootCertFiles "$ZENITH_BANK_TLS_CERT"
    
    echo "🎉 Chaincode committed successfully!"
}

# === Function to query committed chaincode ===
query_committed() {
    echo "🔍 Querying committed chaincode on the channel..."
    peer lifecycle chaincode querycommitted \
        -o "$ORDERER_ADDRESS" \
        --tls "$CORE_PEER_TLS_ENABLED" \
        --cafile "$ORDERER_CA" \
        -C "$CHANNEL_NAME" \
        -n "$CHAINCODE_NAME"
}

# === Main execution ===
case "$1" in
    "approve-access")
        approve_chaincode "AccessBankMSP" "peer0.accessbank.naijachain.org:7051"
        ;;
    "approve-gt")
        approve_chaincode "GTBankMSP" "peer0.gtbank.naijachain.org:8051"
        ;;
    "approve-zenith")
        approve_chaincode "ZenithBankMSP" "peer0.zenithbank.naijachain.org:9051"
        ;;
    "approve-first")
        approve_chaincode "FirstBankMSP" "peer0.firstbank.naijachain.org:10051"
        ;;
    "check")
        check_commit_readiness
        ;;
    "commit")
        commit_chaincode
        query_committed
        ;;
    "all")
        echo "🚀 Performing approvals for all organizations and committing..."
        approve_chaincode "AccessBankMSP" "peer0.accessbank.naijachain.org:7051"
        approve_chaincode "GTBankMSP" "peer0.gtbank.naijachain.org:8051"
        approve_chaincode "ZenithBankMSP" "peer0.zenithbank.naijachain.org:9051"
        check_commit_readiness
        commit_chaincode
        query_committed
        ;;
    *)
        echo "Usage: $0 {approve-access|approve-gt|approve-zenith|approve-first|check|commit|all}"
        echo ""
        echo "Commands:"
        echo "  approve-access   Approve chaincode for AccessBank"
        echo "  approve-gt       Approve chaincode for GTBank"
        echo "  approve-zenith   Approve chaincode for ZenithBank"
        echo "  approve-first    Approve chaincode for FirstBank"
        echo "  check            Check commit readiness across organizations"
        echo "  commit           Commit the chaincode definition after approvals"
        echo "  all              Approve for Access, GT, and Zenith Banks, then commit"
        exit 1
        ;;
esac

echo "✅ Chaincode operation completed successfully!"