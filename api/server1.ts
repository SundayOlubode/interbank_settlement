/* -------------------------------------------------------------
   bank-api/server.ts — Express + Hyperledger Fabric Gateway
   ------------------------------------------------------------- */
import express from "express";
import morgan from "morgan";
import { promises as fs } from "node:fs";
import * as crypto from "node:crypto";
import path from "node:path";
import {
  connect,
  hash,
  signers,
  checkpointers,
  ChaincodeEventsRequest,
  ChaincodeEvent,
} from "@hyperledger/fabric-gateway";
import * as grpc from "@grpc/grpc-js";

/* ---------- env / constants ------------------------------------------------ */
const MSP_ID = process.env.MSP_ID ?? "AccessBankMSP";
const PEER_ENDPOINT = process.env.PEER_ENDPOINT ?? "localhost:7051";
const TLS_CERT_PATH =
  process.env.TLS_CERT_PATH ??
  path.resolve(
    "..",
    "..",
    "crypto-config/peerOrganizations/accessbank.naijachain.org/tlsca/tlsca.accessbank.naijachain.org-cert.pem"
  );
const ID_CERT_PATH =
  process.env.ID_CERT_PATH ??
  path.resolve(
    "..",
    "..",
    "crypto-config/peerOrganizations/accessbank.naijachain.org/users/User1@accessbank.naijachain.org/msp/signcerts/User1@accessbank.naijachain.org-cert.pem"
  );
const KEY_PATH =
  process.env.KEY_PATH ??
  path.resolve(
    "..",
    "..",
    "crypto-config/peerOrganizations/accessbank.naijachain.org/users/User1@accessbank.naijachain.org/msp/keystore/priv_sk"
  );
const CHANNEL = process.env.CHANNEL ?? "retailchannel";
const CHAINCODE = process.env.CHAINCODE ?? "account";
const CHECKPOINT_FILE = process.env.CHECKPOINT_FILE ?? "./payment-events.chk";

/* ---------- fabric helper --------------------------------------------------- */
async function newGateway() {
  const tlsCert = await fs.readFile(TLS_CERT_PATH);
  const creds = grpc.credentials.createSsl(tlsCert);
  const client = new grpc.Client(PEER_ENDPOINT, creds);
  const idBytes = await fs.readFile(ID_CERT_PATH);
  const keyPem = await fs.readFile(KEY_PATH);
  const signerKey = crypto.createPrivateKey(keyPem);

  return connect({
    client,
    identity: { mspId: MSP_ID, credentials: idBytes },
    signer: signers.newPrivateKeySigner(signerKey),
    hash: hash.sha256,
  });
}

/* ---------- payment processing --------------------------------------------- */
// async function processPaymentEvent(evt: ChaincodeEvent, contract: any, cp: ReturnType<typeof checkpointers.file>) {
async function processPaymentEvent(
  evt: ChaincodeEvent,
  contract: any,
  cp: ReturnType<typeof checkpointers.inMemory>
) {
  const payload = JSON.parse(evt.payload.toString());
  if (payload.payeeMSP !== MSP_ID) {
    await cp.checkpointChaincodeEvent(evt);
    return; // not my bank
  }

  // 1. credit core ledger (placeholder)
  console.log(`Crediting ${payload.payeeAcct} with ₦${payload.amount}`);

  // 2. mark SETTLED on‑chain (idempotent)
  await contract.submitTransaction("SettlePayment", payload.id);

  // 3. push mobile notification (placeholder)
  console.log(`Push notification → acct ${payload.payeeAcct}`);

  // 4. checkpoint last processed event
  await cp.checkpointChaincodeEvent(evt);
}

async function startListener(gateway: any) {
  const network = gateway.getNetwork(CHANNEL);
  const contract = network.getContract(CHAINCODE);
  //   const cp       = checkpointers.file(CHECKPOINT_FILE);
  const cp = checkpointers.inMemory();

  while (true) {
    let stream: AsyncIterable<ChaincodeEvent> & { close(): void };
    stream = await network.getChaincodeEvents(CHAINCODE, { checkpoint: cp });

    try {
      for await (const evt of stream) {
        if (evt.eventName !== "PaymentPending") continue;
        await processPaymentEvent(evt, contract, cp);
      }
    } catch (err) {
      console.error("🔌 event stream dropped, reconnecting…", err);
    } finally {
      stream?.close?.();
    }
  }
}

/* ---------- express API ----------------------------------------------------- */
const app = express();
app.use(express.json());
app.use(morgan("dev"));

let gatewayGlobal: any;

app.post("/payments", async (req, res) => {
  try {
    const gw = gatewayGlobal;
    const network = gw.getNetwork(CHANNEL);
    const contract = network.getContract(CHAINCODE);

    const { payerAcct, payeeMSP, payeeAcct, amount } = req.body;
    const paymentID = crypto.randomUUID();

    await contract.submitTransaction(
      "CreatePayment",
      paymentID,
      payerAcct,
      payeeMSP,
      payeeAcct,
      amount.toString()
    );
    res.status(201).json({ id: paymentID, status: "PENDING" });
  } catch (err) {
    console.error(err);
    res.status(500).json({ error: "Could not create payment" });
  }
});

app.post("/payments/:id/settle", async (req, res) => {
  try {
    const { id } = req.params;
    const network = gatewayGlobal.getNetwork(CHANNEL);
    const contract = network.getContract(CHAINCODE);
    await contract.submitTransaction("SettlePayment", id);
    res.json({ id, status: "SETTLED" });
  } catch (err) {
    console.error(err);
    res.status(500).json({ error: "settle failed" });
  }
});

/* ---------- bootstrap everything ------------------------------------------- */
(async () => {
  gatewayGlobal = await newGateway();
  startListener(gatewayGlobal).catch(console.error);
  app.listen(4000, () => console.log("Bank‑API listening on 4000"));
})();
