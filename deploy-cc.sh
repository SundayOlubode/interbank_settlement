set -e

# Import the environment variables
source ./env-vars.sh

export PRIVATE_DATA_CONFIG=${PWD}/private-data/collections_config.json

CC_POLICY="OutOf(2, 'AccessBankMSP.peer', 'GTBankMSP.peer', 'ZenithBankMSP.peer', 'FirstBankMSP.peer', 'CentralBankPeerMSP.peer')"

setGlobalsForOrderer() {
    export MSPCONFIGPATHCBN=${PWD}/crypto-config/ordererOrganizations/cbn.naijachain.org/users/Admin@cbn.naijachain.org/msp
    export CORE_PEER_LOCALMSPID="CentralBankMSP"
    export CORE_PEER_TLS_ROOTCERT_FILE=$ORDERER_CA
    export CORE_PEER_MSPCONFIGPATH=$MSPCONFIGPATHCBN
}

presetup() {
    echo Vendoring Go dependencies ...
    pushd ./chaincode
    rm -rf vendor
    GO111MODULE=on go mod vendor
    popd
    echo Finished vendoring Go dependencies
}

presetup

CC_RUNTIME_LANGUAGE="golang"
VERSION="1"
# CC_SRC_PATH="./chaincode"
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
    setGlobalForPeer0CBN
    set -x
    peer lifecycle chaincode install ${CC_NAME}.tar.gz
    set +x
    echo "===================== Chaincode is installed on peer0.cbn ===================== "

    setGlobalForPeer0AccessBank
    set -x
    peer lifecycle chaincode install ${CC_NAME}.tar.gz
    set +x
    echo "===================== Chaincode is installed on peer0.accesbank ===================== "

    setGlobalForPeer0GTBank
    set -x
    peer lifecycle chaincode install ${CC_NAME}.tar.gz
    set +x
    echo "===================== Chaincode is installed on peer0.gtbank ===================== "

    setGlobalForPeer0ZenithBank
    set -x
    peer lifecycle chaincode install ${CC_NAME}.tar.gz
    set +x
    echo "===================== Chaincode is installed on peer0.zenithbank ===================== "

    setGlobalForPeerFirstBank
    set -x
    peer lifecycle chaincode install ${CC_NAME}.tar.gz
    set +x
    echo "===================== Chaincode is installed on peer0.firstbank ===================== "

    echo -e "\n\n"
}

installChaincode

queryInstalled() {
    setGlobalForPeer0AccessBank
    peer lifecycle chaincode queryinstalled >&log.txt
    cat log.txt
    PACKAGE_ID=$(sed -n "/${CC_NAME}_${VERSION}/{s/^Package ID: //; s/, Label:.*$//; p;}" log.txt)
    echo PackageID is ${PACKAGE_ID}
    echo "===================== Query installed successful on peer0.accessbank on channel ===================== "

    echo -e "\n\n"
}

queryInstalled

approveForMyCBNOrg(){
    setGlobalForPeer0CBN
    set -x
    peer lifecycle chaincode approveformyorg -o localhost:7050 \
        --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
        --tls --connTimeout 180s --collections-config $PRIVATE_DATA_CONFIG \
        --cafile $ORDERER_CA --channelID $CHANNEL_NAME --name ${CC_NAME} \
        --version ${VERSION} --sequence ${VERSION} --package-id ${PACKAGE_ID} \
        --init-required --signature-policy "$CC_POLICY"

    set +x

    echo "===================== chaincode approved from CBN Org ===================== "

    echo -e "\n\n"
}

approveForMyAccessBankOrg() {
    setGlobalForPeer0AccessBank
    set -x
    peer lifecycle chaincode approveformyorg -o localhost:7050 \
        --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
        --tls --connTimeout 180s --collections-config $PRIVATE_DATA_CONFIG \
        --cafile $ORDERER_CA --channelID $CHANNEL_NAME --name ${CC_NAME} \
        --version ${VERSION} --sequence ${VERSION} --package-id ${PACKAGE_ID} \
        --init-required --signature-policy "$CC_POLICY"

    set +x

    echo "===================== chaincode approved from AccessBank Org ===================== "

    echo -e "\n\n"
}

approveForMyGTBankOrg() {
    setGlobalForPeer0GTBank
    peer lifecycle chaincode approveformyorg -o localhost:7050 \
        --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
        --tls --connTimeout 180s --collections-config $PRIVATE_DATA_CONFIG \
        --cafile $ORDERER_CA --channelID $CHANNEL_NAME --name ${CC_NAME} \
        --version ${VERSION} --sequence ${VERSION} --package-id ${PACKAGE_ID} \
        --init-required --signature-policy "$CC_POLICY"

    echo "===================== chaincode approved from GTBank Org ===================== "
    echo -e "\n\n"
}

approveForMyZenithBankOrg() {
    setGlobalForPeer0ZenithBank
    peer lifecycle chaincode approveformyorg -o localhost:7050 \
        --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
        --tls --connTimeout 180s --collections-config $PRIVATE_DATA_CONFIG \
        --cafile $ORDERER_CA --channelID $CHANNEL_NAME --name ${CC_NAME} \
        --version ${VERSION} --sequence ${VERSION} --package-id ${PACKAGE_ID} \
        --init-required --signature-policy "$CC_POLICY"

    echo "===================== chaincode approved from ZenithBank Org ===================== "
    echo -e "\n\n"
}

approveForMyFirstBankOrg() {
    setGlobalForPeerFirstBank
    peer lifecycle chaincode approveformyorg -o localhost:7050 \
        --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
        --tls --connTimeout 180s --collections-config $PRIVATE_DATA_CONFIG \
        --cafile $ORDERER_CA --channelID $CHANNEL_NAME --name ${CC_NAME} \
        --version ${VERSION} --sequence ${VERSION} --package-id ${PACKAGE_ID} \
        --init-required --signature-policy "$CC_POLICY"

    echo "===================== chaincode approved from FirstBank Org ===================== "
    echo -e "\n\n"
}

approveForMyCBNOrg
approveForMyAccessBankOrg
approveForMyGTBankOrg
approveForMyZenithBankOrg
approveForMyFirstBankOrg

checkCommitReadyness() {
    setGlobalForPeer0AccessBank

    peer lifecycle chaincode checkcommitreadiness \
        --collections-config $PRIVATE_DATA_CONFIG \
        --channelID $CHANNEL_NAME --name ${CC_NAME} --version ${VERSION} \
        --collections-config $PRIVATE_DATA_CONFIG \
        --sequence ${VERSION} --init-required --output json

    echo "===================== checking commit readyness from access ===================== "
    echo -e "\n\n"
}

checkCommitReadyness

commitChaincodeDefination() {
    setGlobalForPeer0AccessBank
    peer lifecycle chaincode commit -o localhost:7050 --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
        --tls $CORE_PEER_TLS_ENABLED --cafile $ORDERER_CA \
        --channelID $CHANNEL_NAME --name ${CC_NAME} \
        --collections-config $PRIVATE_DATA_CONFIG \
        --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0ACCESSBANK_CA \
        --peerAddresses localhost:8051 --tlsRootCertFiles $PEER0GTBANK_CA \
        --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0ZENITHBANK_CA \
        --peerAddresses localhost:10051 --tlsRootCertFiles $PEER0FIRSTBANK_CA \
        --version ${VERSION} --sequence ${VERSION} \
        --init-required --signature-policy "$CC_POLICY"

    echo -e "\n\n"
}

commitChaincodeDefination

queryCommitted() {
    setGlobalForPeer0AccessBank
    set -x
    peer lifecycle chaincode querycommitted --channelID $CHANNEL_NAME --name ${CC_NAME}
    set +x

    echo "\n\n"
}

queryCommitted
