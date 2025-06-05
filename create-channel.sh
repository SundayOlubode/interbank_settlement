set -e

source ./env-vars.sh

#if peer docker containers are running, start docker containers with docker-compose up -d
if [ "$(docker ps -q -f name=peer0.accessbank.naijachain.org)" ]; then
	echo "Peer containers are already running."
else
	echo "Starting peer containers..."
	docker-compose -f docker-compose.yaml up -d
fi

sleep 3

echo $FABRIC_CFG_PATH

# Create the channel
createChannel(){
	set -x
	setGlobalForPeer0AccessBank

  echo ">>> Creating channel: $CHANNEL_NAME"
  # echo ">>> Using orderer at: orderer.cbn.naijachain.org:7050"
  echo ">>> Using orderer at: localhost:7050"
  echo ">>> TLS enabled: $CORE_PEER_TLS_ENABLED"
  echo ">>> Orderer CA file: $ORDERER_CA"

	# peer channel create -o orderer.cbn.naijachain.org:7050 -c $CHANNEL_NAME \
	peer channel create -o localhost:7050 -c $CHANNEL_NAME \
	-f ./channel-artifacts/${CHANNEL_NAME}/channel.tx \
	--ordererTLSHostnameOverride orderer.cbn.naijachain.org \
	--outputBlock ./channel-artifacts/${CHANNEL_NAME}/${CHANNEL_NAME}.block \
	--tls $CORE_PEER_TLS_ENABLED --cafile $ORDERER_CA --connTimeout 120s

	echo -e "\n\n"
}


joinChannel(){
	set -x
	setGlobalForPeer0AccessBank
	peer channel join -b ./channel-artifacts/${CHANNEL_NAME}/${CHANNEL_NAME}.block
	
	setGlobalForPeer0GTBank
	peer channel join -b ./channel-artifacts/${CHANNEL_NAME}/${CHANNEL_NAME}.block
	
	setGlobalForPeer0ZenithBank
	peer channel join -b ./channel-artifacts/${CHANNEL_NAME}/${CHANNEL_NAME}.block
	
	setGlobalForPeerFirstBank
	peer channel join -b ./channel-artifacts/${CHANNEL_NAME}/${CHANNEL_NAME}.block

	echo -e "\n\n"
}

updateAnchorPeers(){
	set -x

	setGlobalForPeer0AccessBank
	peer channel update -o localhost:7050 --ordererTLSHostnameOverride orderer.cbn.naijachain.org -c $CHANNEL_NAME -f ./channel-artifacts/retailchannel/${CORE_PEER_LOCALMSPID}anchors.tx --tls $CORE_PEER_TLS_ENABLED --cafile $ORDERER_CA
	
	setGlobalForPeer0GTBank
	peer channel update -o localhost:7050 --ordererTLSHostnameOverride orderer.cbn.naijachain.org -c $CHANNEL_NAME -f ./channel-artifacts/retailchannel/${CORE_PEER_LOCALMSPID}anchors.tx --tls $CORE_PEER_TLS_ENABLED --cafile $ORDERER_CA

	setGlobalForPeer0ZenithBank
	peer channel update -o localhost:7050 --ordererTLSHostnameOverride orderer.cbn.naijachain.org -c $CHANNEL_NAME -f ./channel-artifacts/retailchannel/${CORE_PEER_LOCALMSPID}anchors.tx --tls $CORE_PEER_TLS_ENABLED --cafile $ORDERER_CA

	setGlobalForPeerFirstBank
	peer channel update -o localhost:7050 --ordererTLSHostnameOverride orderer.cbn.naijachain.org -c $CHANNEL_NAME -f ./channel-artifacts/retailchannel/${CORE_PEER_LOCALMSPID}anchors.tx --tls $CORE_PEER_TLS_ENABLED --cafile $ORDERER_CA

	echo -e "\n\n"
}


createChannel
joinChannel
updateAnchorPeers


set +x
