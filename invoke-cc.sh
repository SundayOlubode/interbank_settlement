#Import environment variables
source ./env-vars.sh

set -x

setGlobalForPeer0AccessBank

chaincodeInvokeInit() {
    setGlobalForPeer0AccessBank
    peer chaincode invoke -o localhost:7050 \
        --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
        --tls $CORE_PEER_TLS_ENABLED \
        --cafile $ORDERER_CA \
        -C $CHANNEL_NAME -n ${CC_NAME} \
        --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0ACCESSBANK_CA \
        --peerAddresses localhost:8051 --tlsRootCertFiles $PEER0GTBANK_CA \
        --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0ZENITHBANK_CA \
        --peerAddresses localhost:10051 --tlsRootCertFiles $PEER0FIRSTBANK_CA \
        --isInit -c '{"Args":[]}'
}
chaincodeInvokeInit

sleep 2

chaincodeCreateAccount(){
    setGlobalForPeer0AccessBank
    peer chaincode invoke -o localhost:7050 \
        --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
        --tls $CORE_PEER_TLS_ENABLED \
        --cafile $ORDERER_CA \
        -C $CHANNEL_NAME -n ${CC_NAME} \
        --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0ACCESSBANK_CA \
        --peerAddresses localhost:8051 --tlsRootCertFiles $PEER0GTBANK_CA \
        --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0ZENITHBANK_CA \
        --peerAddresses localhost:10051 --tlsRootCertFiles $PEER0FIRSTBANK_CA \
        -c '{"function": "InitLedger","Args":[]}'

    setGlobalForPeer0GTBank
    peer chaincode invoke -o localhost:7050 \
        --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
        --tls $CORE_PEER_TLS_ENABLED \
        --cafile $ORDERER_CA \
        -C $CHANNEL_NAME -n ${CC_NAME} \
        --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0ACCESSBANK_CA \
        --peerAddresses localhost:8051 --tlsRootCertFiles $PEER0GTBANK_CA \
        --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0ZENITHBANK_CA \
        --peerAddresses localhost:10051 --tlsRootCertFiles $PEER0FIRSTBANK_CA \
        -c '{"function": "InitLedger","Args":[]}'

    setGlobalForPeer0ZenithBank
    peer chaincode invoke -o localhost:7050 \
        --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
        --tls $CORE_PEER_TLS_ENABLED \
        --cafile $ORDERER_CA \
        -C $CHANNEL_NAME -n ${CC_NAME} \
        --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0ACCESSBANK_CA \
        --peerAddresses localhost:8051 --tlsRootCertFiles $PEER0GTBANK_CA \
        --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0ZENITHBANK_CA \
        --peerAddresses localhost:10051 --tlsRootCertFiles $PEER0FIRSTBANK_CA \
        -c '{"function": "InitLedger","Args":[]}'

    setGlobalForPeerFirstBank
    peer chaincode invoke -o localhost:7050 \
        --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
        --tls $CORE_PEER_TLS_ENABLED \
        --cafile $ORDERER_CA \
        -C $CHANNEL_NAME -n ${CC_NAME} \
        --peerAddresses localhost:7051 --tlsRootCertFiles $PEER0ACCESSBANK_CA \
        --peerAddresses localhost:8051 --tlsRootCertFiles $PEER0GTBANK_CA \
        --peerAddresses localhost:9051 --tlsRootCertFiles $PEER0ZENITHBANK_CA \
        --peerAddresses localhost:10051 --tlsRootCertFiles $PEER0FIRSTBANK_CA \
        -c '{"function": "InitLedger","Args":[]}'
}

chaincodeCreateAccount

# sleep 2

# readAccountBalance(){
#     setGlobalForPeer0AccessBank
#     peer chaincode query -C $CHANNEL_NAME -n ${CC_NAME} -c '{"Args":["ReadAccount"]}'
# }
# readAccountBalance

set +x