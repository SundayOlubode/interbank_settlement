set -e

# Import the environment variables
source ./env-vars.sh

export MSPCONFIGPATHCBN=${PWD}/crypto-config/ordererOrganizations/cbn.naijachain.org/users/Admin@cbn.naijachain.org/msp

setGlobalsForOrderer() {
  export CORE_PEER_LOCALMSPID="CentralBankMSP"
  export CORE_PEER_TLS_ROOTCERT_FILE=$ORDERER_CA
  export CORE_PEER_MSPCONFIGPATH=$MSPCONFIGPATHCBN
}

presetup() {
    echo Vendoring Go dependencies ...
    pushd ./contracts
    rm -rf vendor
    GO111MODULE=on go mod vendor
    popd
    echo Finished vendoring Go dependencies
}

presetup

CC_RUNTIME_LANGUAGE="golang"
VERSION="1"
CC_SRC_PATH="./contracts"
CC_NAME="account"

packageChaincode() {
    rm -rf ${CC_NAME}.tar.gz

    setGlobalForPeer0AccessBank

    peer lifecycle chaincode package ${CC_NAME}.tar.gz \
        --path ${CC_SRC_PATH} --lang ${CC_RUNTIME_LANGUAGE} \
        --label ${CC_NAME}_${VERSION}
    echo "===================== Chaincode is packaged on peer0.accessbank.naijachain.org ===================== "
}

packageChaincode

installChaincode() {
    setGlobalForPeer0AccessBank
    peer lifecycle chaincode install ${CC_NAME}.tar.gz
    echo "===================== Chaincode is installed on peer0.accesbank ===================== "

    setGlobalForPeer0GTBank
    peer lifecycle chaincode install ${CC_NAME}.tar.gz
    echo "===================== Chaincode is installed on peer0.gtbank ===================== "

    setGlobalForPeer0ZenithBank
    peer lifecycle chaincode install ${CC_NAME}.tar.gz
    echo "===================== Chaincode is installed on peer0.zenithbank ===================== "

    setGlobalForPeerFirstBank
    peer lifecycle chaincode install ${CC_NAME}.tar.gz
    echo "===================== Chaincode is installed on peer0.firstbank ===================== "
}

installChaincode

queryInstalled() {
    setGlobalForPeer0AccessBank
    peer lifecycle chaincode queryinstalled >&log.txt
    cat log.txt
    PACKAGE_ID=$(sed -n "/${CC_NAME}_${VERSION}/{s/^Package ID: //; s/, Label:.*$//; p;}" log.txt)
    echo PackageID is ${PACKAGE_ID}
    echo "===================== Query installed successful on peer0.accessbank on channel ===================== "
}

queryInstalled

approveForMyAccessBankOrg() {
    setGlobalForPeer0AccessBank
    set -x
    peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.cbn.naijachain.org --tls --connTimeout 180s \
    --cafile $ORDERER_CA --channelID $CHANNEL_NAME --name ${CC_NAME} --version ${VERSION} --sequence ${VERSION} --package-id ${PACKAGE_ID}
        
    set +x

    echo "===================== chaincode approved from AccessBank Org ===================== "

}

approveForMyGTBankOrg() {
    setGlobalForPeer0GTBank
    peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.cbn.naijachain.org --tls \
    --cafile $ORDERER_CA --channelID $CHANNEL_NAME --name ${CC_NAME} --version ${VERSION} --sequence ${VERSION} --package-id ${PACKAGE_ID}

    echo "===================== chaincode approved from GTBank Org ===================== "
}

approveForMyZenithBankOrg() {
    setGlobalForPeer0ZenithBank
    peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.cbn.naijachain.org --tls \
    --cafile $ORDERER_CA --channelID $CHANNEL_NAME --name ${CC_NAME} --version ${VERSION} --sequence ${VERSION} --package-id ${PACKAGE_ID}

    echo "===================== chaincode approved from ZenithBank Org ===================== "
}

approveForMyFirstBankOrg() {
    setGlobalForPeerFirstBank
    peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.cbn.naijachain.org --tls \
    --cafile $ORDERER_CA --channelID $CHANNEL_NAME --name ${CC_NAME} --version ${VERSION} --sequence ${VERSION} --package-id ${PACKAGE_ID}

    echo "===================== chaincode approved from FirstBank Org ===================== "
}

approveForMyAccessBankOrg
approveForMyGTBankOrg
approveForMyZenithBankOrg
approveForMyFirstBankOrg

checkCommitReadyness() {
    setGlobalForPeer0AccessBank

    peer lifecycle chaincode checkcommitreadiness \
        --channelID $CHANNEL_NAME --name ${CC_NAME} --version ${VERSION} \
        --sequence ${VERSION} --output json
    echo "===================== checking commit readyness from access ===================== "
}

checkCommitReadyness

commitChaincodeDefination() {
    setGlobalForPeer0AccessBank
    peer lifecycle chaincode commit -o localhost:7050 --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
        --tls $CORE_PEER_TLS_ENABLED --cafile $ORDERER_CA \
        --channelID $CHANNEL_NAME --name ${CC_NAME} \
        --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0ACCESSBANK_CA \
        --peerAddresses localhost:8051 --tlsRootCertFiles $PEER0GTBANK_CA \
        --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0ZENITHBANK_CA \
        --version ${VERSION} --sequence ${VERSION}
}

commitChaincodeDefination

queryCommitted() {
    setGlobalForPeer0AccessBank
    peer lifecycle chaincode querycommitted --channelID $CHANNEL_NAME --name ${CC_NAME}

}

queryCommitted