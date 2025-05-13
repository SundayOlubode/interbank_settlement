# Generate the crypto material for the network
cryptogen generate --config=./crypto-config.yaml

# Generate the genesis block for the orderer
configtxgen -profile NaijaBanksOrdererGenesis -channelID retail-sys-channel -outputBlock ./channel-artifacts/genesis.block

# Generate the channel configuration transaction
configtxgen -profile RetailChannel -outputCreateChannelTx ./channel-artifacts/retailchannel/channel.tx -channelID retailchannel

# Generate anchor peer update transactions for Org1 and Org2
configtxgen -profile RetailChannel -outputAnchorPeersUpdate ./channel-artifacts/retailchannel/AccessBankMSPanchors.tx -channelID retailchannel -asOrg AccessBankOrg
configtxgen -profile RetailChannel -outputAnchorPeersUpdate ./channel-artifacts/retailchannel/GTBankMSPanchors.tx -channelID retailchannel -asOrg GTBankOrg
configtxgen -profile RetailChannel -outputAnchorPeersUpdate ./channel-artifacts/retailchannel/ZenithBankMSPanchors.tx -channelID retailchannel -asOrg ZenithBankOrg
configtxgen -profile RetailChannel -outputAnchorPeersUpdate ./channel-artifacts/retailchannel/FirstBankMSPanchors.tx -channelID retailchannel -asOrg FirstBankOrg