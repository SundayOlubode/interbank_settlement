export FABRIC_CFG_PATH=${PWD}

set -x

# If crypto-config directory exists, remove it
if [ -d "crypto-config" ]; then
    rm -rf crypto-config
fi

# Generate the crypto material for the network
cryptogen generate --config=./crypto-config.yaml

# If channel-artifacts directory exists, remove it
if [ -d "channel-artifacts" ]; then
    rm -rf channel-artifacts
fi

# Generate the genesis block for the orderer
configtxgen -profile NaijaBanksOrdererGenesis -channelID retail-sys-channel -outputBlock ./channel-artifacts/genesis.block

# Generate the channel configuration transaction
configtxgen -profile RetailChannel -outputCreateChannelTx ./channel-artifacts/retailchannel/channel.tx -channelID retailchannel

# Generate anchor peer update transactions for each organization
configtxgen -profile RetailChannel -outputAnchorPeersUpdate ./channel-artifacts/retailchannel/AccessBankMSPanchors.tx -channelID retailchannel -asOrg AccessBankMSP
configtxgen -profile RetailChannel -outputAnchorPeersUpdate ./channel-artifacts/retailchannel/GTBankMSPanchors.tx -channelID retailchannel -asOrg GTBankMSP
configtxgen -profile RetailChannel -outputAnchorPeersUpdate ./channel-artifacts/retailchannel/ZenithBankMSPanchors.tx -channelID retailchannel -asOrg ZenithBankMSP
configtxgen -profile RetailChannel -outputAnchorPeersUpdate ./channel-artifacts/retailchannel/FirstBankMSPanchors.tx -channelID retailchannel -asOrg FirstBankMSP

set +x