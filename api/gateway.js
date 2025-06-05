import * as grpc from '@grpc/grpc-js';
import { connect, hash, signers } from '@hyperledger/fabric-gateway';
import * as crypto from 'node:crypto';
import { promises as fs } from 'node:fs';
import { TextDecoder } from 'node:util';

const utf8Decoder = new TextDecoder();

const channelName = 'retailchannel';
const chaincodeName = 'account';

async function main() {
    const credentials = await fs.readFile(
    '/Users/sam/Documents/Blockchain/interbank_settlement/crypto-config' +
    '/peerOrganizations/accessbank.naijachain.org/users/User1@accessbank.naijachain.org/msp/signcerts/' +
    'User1@accessbank.naijachain.org-cert.pem');

    const privateKeyPem = await fs.readFile(
    '/Users/sam/Documents/Blockchain/interbank_settlement/crypto-config' +
    '/peerOrganizations/accessbank.naijachain.org/users/User1@accessbank.naijachain.org/msp/keystore/' +
    'priv_sk');

    const privateKey = crypto.createPrivateKey(privateKeyPem);
    const signer = signers.newPrivateKeySigner(privateKey);

    const tlsRootCert = await fs.readFile('/Users/sam/Documents/Blockchain/interbank_settlement/crypto-config/peerOrganizations/accessbank.naijachain.org/tlsca/tlsca.accessbank.naijachain.org-cert.pem');
    const client = new grpc.Client('localhost:7051', grpc.credentials.createSsl(tlsRootCert));

    const gateway = connect({
        identity: { mspId: 'AccessBankMSP', credentials },
        signer,
        hash: hash.sha256,
        client,
    });

    try {
        const network = gateway.getNetwork(channelName);
        const contract = network.getContract(chaincodeName);

        // const putResult = await contract.submitTransaction('put', 'time', new Date().toISOString());
        // console.log('Put result:', utf8Decoder.decode(putResult));

        // const getResult = await contract.evaluateTransaction('CreateAccount');
        const getResult = await contract.evaluate('ReadAccount');
        console.log('Get result:', utf8Decoder.decode(getResult));
    } finally {
        gateway.close();
        client.close();
    }
}

main().catch(console.error);