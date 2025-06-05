"use strict";

import express from 'express';
import morgan from 'morgan';
import * as grpc from '@grpc/grpc-js';
import { connect, hash, signers, Network, Contract } from '@hyperledger/fabric-gateway';
import { promises as fs } from 'fs';
import crypto from 'node:crypto';
import { v4 as uuid } from 'uuid';
import dotenv from 'dotenv';
dotenv.config();

const MSP_ID = process.env.MSP_ID ?? 'AccessBankMSP';
const PEER_ENDPOINT = process.env.PEER_ENDPOINT ?? 'localhost:7051';
const TLS_CERT_PATH = process.env.TLS_CERT_PATH ?? '/path/to/peer/tls/ca.pem';
const IDENTITY_PATH = process.env.IDENTITY_CERT ?? '/path/to/User1@org-cert.pem';
const PRIVATE_KEY_PATH = process.env.PRIV_KEY ?? '/path/to/priv_sk';

const CHANNEL = 'retailchannel';
const CHAINCODE = 'account';

/** Helper to create a cached gateway connection */
async function newGateway() {
  const creds = await fs.readFile(IDENTITY_PATH);
  const keyPem = await fs.readFile(PRIVATE_KEY_PATH);
  const tlsRoot = await fs.readFile(TLS_CERT_PATH);
  const privateKey = crypto.createPrivateKey(keyPem);

  const client = new grpc.Client(PEER_ENDPOINT, grpc.credentials.createSsl(tlsRoot));
  const signer = signers.newPrivateKeySigner(privateKey);

  return connect({
    identity: { mspId: MSP_ID, credentials: creds },
    signer,
    hash: hash.sha256,
    client,
    // default Deadline: 5s
    // signingIdentity: undefined,
  });
}

// Express app
const app = express();
app.use(express.json());
app.use(morgan('combined'));

// --- POST /payments -------------------------------------------------
app.post('/payments', async (req, res) => {
  /** Expected body: { payeeMSP, payeeAcct, amount, reference } */
  try {
    const { payeeMSP, payeeAcct, amount, reference } = req.body;
    const paymentID = `PAY_${uuid()}`;
    const timestamp = new Date().toISOString();

    // Build private payload
    const payload = {
      id: paymentID,
      payerMSP: MSP_ID,
      payeeMSP,
      payerAcct: req.headers['x-account-id'], // from access token / header
      payeeAcct,
      amount: parseFloat(amount),
      reference,
      timestamp,
    };

    const hashHex = crypto.createHash('sha256').update(JSON.stringify(payload)).digest('hex');
    payload['hash'] = hashHex;

    const gw = await newGateway();
    const network: Network = gw.getNetwork(CHANNEL);
    const contract: Contract = network.getContract(CHAINCODE);

    // invoke transaction with transient map
    const transient = { payment: Buffer.from(JSON.stringify(payload)) };
    await contract.submit('CreatePayment', {
      arguments: [paymentID],
      transientData: transient,
    });

    res.status(201).json({ paymentID, status: 'PENDING' });
    gw.close();
  } catch (err) {
    console.error(err);
    res.status(500).json({ error: 'failed to create payment' });
  }
});

// --- GET /payments/:id (public stub) --------------------------------
app.get('/payments/:id', async (req, res) => {
  try {
    const gw = await newGateway();
    const contract = gw.getNetwork(CHANNEL).getContract(CHAINCODE);
    const result = await contract.evaluate('ReadPayment', { arguments: [req.params.id] });
    res.json(JSON.parse(result.toString()));
    gw.close();
  } catch (err) {
    res.status(404).json({ error: 'not found' });
  }
});

// --- webhook listener for block events (notify mobile‑apps) ---------
// In production you would use a message bus; here we only log.
(async () => {
  const gw = await newGateway();
  const network = gw.getNetwork(CHANNEL);
  const listener = async (event: any) => {
    for (const tx of event.getTransactionEvents()) {
      console.log(`block ${event.blockNumber} committed txid=${tx.transactionId}`);
    }
  };
  // await network.addBlockListener(listener);
  // await network.getBlockAndPrivateDataEvents(listener);
})();

// -------------------------------------------------------------
const PORT = process.env.PORT || 3000;
app.listen(PORT, () => console.log(`Bank API listening on ${PORT}`));