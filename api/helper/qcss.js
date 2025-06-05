import pkg from '@hyperledger/fabric-protos';
const { common: pbCommon, peer: pbPeer } = pkg;

/**
* buildQsccHelpers(gateway, channelName) → { getBlockByNumber, getChainInfo, iterateBlocks }
*/
export async function buildQsccHelpers(gateway, channelName = 'retailchannel') {
    const network = gateway.getNetwork(channelName);
    const qscc = network.getContract('qscc');

    async function getBlockByNumber(num) {
        // Ensure num is a string for the transaction
        const numStr = typeof num === 'bigint' ? num.toString() : num.toString();
        const bytes = await qscc.evaluateTransaction('GetBlockByNumber', channelName, numStr);
        return pbCommon.Block.deserializeBinary(bytes);
    }

    async function getChainInfo() {
        const bytes = await qscc.evaluateTransaction('GetChainInfo', channelName);
        return pbCommon.BlockchainInfo.deserializeBinary(bytes);
    }

    async function* iterateBlocks() {
        const info = await getChainInfo();
        const height = info.getHeight();
        
        // Convert height to number to avoid BigInt issues
        const heightNum = typeof height === 'bigint' ? Number(height) : height;
        
        console.log(`Iterating through ${heightNum} blocks`);
        
        // Use regular numbers instead of BigInt
        for (let i = 0; i < heightNum; i++) {
            try {
                yield await getBlockByNumber(i);
            } catch (error) {
                console.error(`Error getting block ${i}:`, error.message);
                throw error;
            }
        }
    }

    return { getBlockByNumber, getChainInfo, iterateBlocks };
}