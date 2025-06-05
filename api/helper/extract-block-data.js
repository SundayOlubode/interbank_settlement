import pkg from '@hyperledger/fabric-protos';
const { common: pbCommon, peer: pbPeer } = pkg;

// Helper function to detect if data is binary/protobuf
function isBinaryData(buffer) {
    const str = buffer.toString();
    // Check for control characters (indicates binary data)
    return /[\x00-\x08\x0E-\x1F\x7F-\x9F]/.test(str);
}

// Helper function to safely convert arguments
function parseArgument(arg) {
    const buffer = Buffer.from(arg);
    
    // Check if it's binary data
    if (isBinaryData(buffer)) {
        return {
            type: 'binary',
            hex: buffer.toString('hex'),
            base64: buffer.toString('base64'),
            size: buffer.length,
            preview: buffer.toString().substring(0, 50) + '...' // First 50 chars for preview
        };
    }
    
    // It's text data, try to parse
    const str = buffer.toString();
    
    // Try to parse as JSON
    try {
        return {
            type: 'json',
            value: JSON.parse(str)
        };
    } catch {
        // Return as plain text
        return {
            type: 'text',
            value: str
        };
    }
}

// Updated function extraction with better argument parsing
export function extractSimpleBlockData(block) {
    const header = block.getHeader();
    const data = block.getData();
    
    const blockNumber = typeof header.getNumber() === 'bigint' 
        ? Number(header.getNumber()) 
        : header.getNumber();
    
    const blockHash = header.getDataHash();
    const previousHash = header.getPreviousHash();
    
    const transactions = [];
    const dataList = data.getDataList();
    
    for (let i = 0; i < dataList.length; i++) {
        try {
            const envelope = pbCommon.Envelope.deserializeBinary(dataList[i]);
            const payload = pbCommon.Payload.deserializeBinary(envelope.getPayload());
            const channelHeader = pbCommon.ChannelHeader.deserializeBinary(payload.getHeader().getChannelHeader());
            
            const transaction = {
                index: i,
                txId: channelHeader.getTxId(),
                timestamp: channelHeader.getTimestamp() ? {
                    seconds: channelHeader.getTimestamp().getSeconds(),
                    nanos: channelHeader.getTimestamp().getNanos(),
                    readable: new Date(channelHeader.getTimestamp().getSeconds() * 1000).toISOString()
                } : null,
                channelId: channelHeader.getChannelId(),
                type: channelHeader.getType(),
                typeDescription: getTransactionTypeDescription(channelHeader.getType()),
                dataSize: dataList[i].length
            };
            
            // For ENDORSER_TRANSACTION (type 3)
            if (channelHeader.getType() === 3) {
                try {
                    const txData = pbPeer.Transaction.deserializeBinary(payload.getData());
                    const actions = txData.getActionsList();
                    
                    transaction.chaincodeData = [];
                    
                    for (let j = 0; j < actions.length; j++) {
                        const action = actions[j];
                        const ccActionPayload = pbPeer.ChaincodeActionPayload.deserializeBinary(action.getPayload());
                        
                        try {
                            const ccProposalPayload = pbPeer.ChaincodeProposalPayload.deserializeBinary(ccActionPayload.getChaincodeProposalPayload());
                            const ccInvocationSpec = pbPeer.ChaincodeInvocationSpec.deserializeBinary(ccProposalPayload.getInput());
                            const ccSpec = ccInvocationSpec.getChaincodeSpec();
                            
                            const chaincodeInfo = {
                                chaincodeName: ccSpec.getChaincodeId().getName(),
                                function: '',
                                args: [],
                                isLifecycleTransaction: false
                            };
                            
                            // Extract function and arguments with proper handling
                            const argsList = ccSpec.getInput().getArgsList();
                            if (argsList.length > 0) {
                                chaincodeInfo.function = Buffer.from(argsList[0]).toString();
                                
                                // Check if this is a lifecycle transaction
                                chaincodeInfo.isLifecycleTransaction = chaincodeInfo.chaincodeName === '_lifecycle';
                                
                                // Parse arguments based on transaction type
                                chaincodeInfo.args = argsList.slice(1).map((arg, index) => {
                                    const parsedArg = parseArgument(arg);
                                    
                                    // Add context for lifecycle transactions
                                    if (chaincodeInfo.isLifecycleTransaction) {
                                        if (chaincodeInfo.function === 'ApproveChaincodeDefinitionForMyOrg' && index === 0) {
                                            parsedArg.description = 'Chaincode definition (protobuf encoded)';
                                        }
                                    }
                                    
                                    return parsedArg;
                                });
                            }
                            
                            // Try to extract response
                            try {
                                const proposalResponsePayload = pbPeer.ProposalResponsePayload.deserializeBinary(ccActionPayload.getAction().getProposalResponsePayload());
                                const ccActionResult = pbPeer.ChaincodeAction.deserializeBinary(proposalResponsePayload.getExtension());
                                
                                const response = ccActionResult.getResponse();
                                if (response) {
                                    chaincodeInfo.response = {
                                        status: response.getStatus(),
                                        message: response.getMessage(),
                                        payload: null
                                    };
                                    
                                    if (response.getPayload()) {
                                        const responsePayload = parseArgument(response.getPayload());
                                        chaincodeInfo.response.payload = responsePayload;
                                    }
                                }
                                
                                if (ccActionResult.getResults()) {
                                    chaincodeInfo.hasWriteSet = true;
                                    chaincodeInfo.writeSetSize = ccActionResult.getResults().length;
                                }
                                
                            } catch (responseErr) {
                                console.log('Could not extract chaincode response:', responseErr.message);
                            }
                            
                            transaction.chaincodeData.push(chaincodeInfo);
                            
                        } catch (ccErr) {
                            console.log('Could not parse chaincode proposal:', ccErr.message);
                        }
                    }
                } catch (txErr) {
                    console.log('Could not parse transaction data:', txErr.message);
                }
            }
            
            transactions.push(transaction);
            
        } catch (err) {
            transactions.push({
                index: i,
                error: 'Could not parse transaction',
                errorMessage: err.message,
                rawDataLength: dataList[i].length
            });
        }
    }
    
    return {
        blockNumber: blockNumber,
        blockHash: blockHash ? Buffer.from(blockHash).toString('hex') : null,
        previousBlockHash: previousHash ? Buffer.from(previousHash).toString('hex') : null,
        transactionCount: transactions.length,
        transactions: transactions,
        timestamp: new Date().toISOString()
    };
}

// Helper function to describe transaction types
function getTransactionTypeDescription(type) {
    const types = {
        0: 'MESSAGE',
        1: 'CONFIG',
        2: 'CONFIG_UPDATE', 
        3: 'ENDORSER_TRANSACTION',
        4: 'ORDERER_TRANSACTION',
        5: 'DELIVER_SEEK_INFO',
        6: 'CHAINCODE_PACKAGE'
    };
    return types[type] || `UNKNOWN_TYPE_${type}`;
}