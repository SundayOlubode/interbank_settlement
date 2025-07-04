source ./env-vars.sh

queryCommitted() {
    setGlobalForPeer0AccessBank
    set -x
    peer lifecycle chaincode querycommitted --channelID $CHANNEL_NAME --name ${CC_NAME}
    set +x

    echo "\n\n"
}

queryCommitted