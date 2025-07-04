import { promises as fs } from "node:fs";
import * as crypto from "node:crypto";
import {
  connect,
  hash,
  signers,
  checkpointers,
} from "@hyperledger/fabric-gateway";
import * as grpc from "@grpc/grpc-js";
import { config } from "../config/env.js";

class FabricService {
  constructor() {
    this.gateway = null;
    this.network = null;
    this.contract = null;
  }

  async connect() {
    const tlsCert = await fs.readFile(config.TLS_CERT_PATH);
    const creds = grpc.credentials.createSsl(tlsCert);
    const client = new grpc.Client(config.PEER_ENDPOINT, creds);
    const idBytes = await fs.readFile(config.ID_CERT_PATH);
    const keyPem = await fs.readFile(config.KEY_PATH);
    const signerKey = crypto.createPrivateKey(keyPem);

    this.gateway = connect({
      client,
      identity: { mspId: config.MSP_ID, credentials: idBytes },
      signer: signers.newPrivateKeySigner(signerKey),
      hash: hash.sha256,
      discovery: { enabled: true, asLocalhost: true },
    });

    this.network = this.gateway.getNetwork(config.CHANNEL);
    this.contract = this.network.getContract(config.CHAINCODE);

    return this.gateway;
  }

  getGateway() {
    return this.gateway;
  }

  getNetwork() {
    return this.network;
  }

  getContract() {
    return this.contract;
  }
}

export const fabricService = new FabricService();
