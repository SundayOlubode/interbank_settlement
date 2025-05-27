export FABRIC_CFG_PATH=${PWD}/config
# ORDERER_CA=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/cbn.naijachain.org/orderers/orderer.cbn.naijachain.org/msp/tlscacerts/tlsca.cbn.naijachain.org-cert.pem
export ORDERER_CA=${PWD}/crypto-config/ordererOrganizations/cbn.naijachain.org/orderers/orderer.cbn.naijachain.org/msp/tlscacerts/tlsca.cbn.naijachain.org-cert.pem
# export ORDERER_CA=${PWD}/crypto-config/ordererOrganizations/cbn.naijachain.org/tlsca/tlsca.cbn.naijachain.org-cert.pem
export CHANNEL_NAME="retailchannel"
export CORE_PEER_TLS_ENABLED=true

# Set the TLS root cert for peers on each bank
export PEER0ACCESSBANK_CA=${PWD}/crypto-config/peerOrganizations/accessbank.naijachain.org/peers/peer0.accessbank.naijachain.org/tls/ca.crt
export PEER0GTBANK_CA=${PWD}/crypto-config/peerOrganizations/gtbank.naijachain.org/peers/peer0.gtbank.naijachain.org/tls/ca.crt
export PEER0ZENITHBANK_CA=${PWD}/crypto-config/peerOrganizations/zenithbank.naijachain.org/peers/peer0.zenithbank.naijachain.org/tls/ca.crt
export PEER0FIRSTBANK_CA=${PWD}/crypto-config/peerOrganizations/firstbank.naijachain.org/peers/peer0.firstbank.naijachain.org/tls/ca.crt

# Set MSPCONFIGPATH for each bank
export MSPCONFIGPATHACCESSBANK=${PWD}/crypto-config/peerOrganizations/accessbank.naijachain.org/users/Admin@accessbank.naijachain.org/msp
export MSPCONFIGPATHGTBANK=${PWD}/crypto-config/peerOrganizations/gtbank.naijachain.org/users/Admin@gtbank.naijachain.org/msp
export MSPCONFIGPATHZENITHBANK=${PWD}/crypto-config/peerOrganizations/zenithbank.naijachain.org/users/Admin@zenithbank.naijachain.org/msp
export MSPCONFIGPATHFIRSTBANK=${PWD}/crypto-config/peerOrganizations/firstbank.naijachain.org/users/Admin@firstbank.naijachain.org/msp


setGlobalForPeer0AccessBank() {
  export CORE_PEER_LOCALMSPID="AccessBankMSP"
  export CORE_PEER_ADDRESS="localhost:7051"
  # export CORE_PEER_ADDRESS="peer0.accessbank.naijachain.org:7051"
  export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0ACCESSBANK_CA
  export CORE_PEER_MSPCONFIGPATH=$MSPCONFIGPATHACCESSBANK
}

setGlobalForPeer0GTBank() {
	export CORE_PEER_LOCALMSPID="GTBankMSP"
	export CORE_PEER_ADDRESS="localhost:8051"
	# export CORE_PEER_ADDRESS="peer0.gtbank.naijachain.org:8051"
	export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0GTBANK_CA
	export CORE_PEER_MSPCONFIGPATH=$MSPCONFIGPATHGTBANK
}

setGlobalForPeer0ZenithBank() {
	export CORE_PEER_LOCALMSPID="ZenithBankMSP"
	export CORE_PEER_ADDRESS="localhost:9051"
	# export CORE_PEER_ADDRESS="peer0.zenithbank.naijachain.org:9051"
	export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0ZENITHBANK_CA
	export CORE_PEER_MSPCONFIGPATH=$MSPCONFIGPATHZENITHBANK
}

setGlobalForPeerFirstBank() {
	export CORE_PEER_LOCALMSPID="FirstBankMSP"
	# export CORE_PEER_ADDRESS="peer0.firstbank.naijachain.org:10051"
	export CORE_PEER_ADDRESS="localhost:10051"
	export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0FIRSTBANK_CA
	export CORE_PEER_MSPCONFIGPATH=$MSPCONFIGPATHFIRSTBANK
}