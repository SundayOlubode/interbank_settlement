/* -------------------------------------------------------------
   bank-api/server.js — Express + Hyperledger Fabric Gateway
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
} from "@hyperledger/fabric-gateway";
import * as grpc from "@grpc/grpc-js";
import * as dotenv from "dotenv";
dotenv.config();
import { buildQsccHelpers } from "./helper/qcss.js";

const utf8Decoder = new TextDecoder();

/* ---------- env / constants ------------------------------------------------ */
const MSP_ID = process.env.ACCESSBANK_MSP_ID ?? "AccessBankMSP";
const PEER_ENDPOINT = process.env.ACCESSBANK_PEER_ENDPOINT ?? "localhost:7051";
const TLS_CERT_PATH = process.env.ACCESSBANK_TLS_CERT_PATH;
const ID_CERT_PATH = process.env.ACCESSBANK_ID_CERT_PATH;
const KEY_PATH = process.env.ACCESSBANK_KEY_PATH;
const CHANNEL = process.env.CHANNEL;
const CHAINCODE = process.env.CHAINCODE_NAME;
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
    discovery: { enabled: true, asLocalhost: true },
  });
}

let qscc;

/* ---------- payment processing --------------------------------------------- */
async function processPaymentEvent(evt, contract, cp) {
  const { id, payeeMSP } = JSON.parse(
    Buffer.from(evt.payload).toString("utf8")
  );

  console.log(
    `Processing event ${evt.eventName} for payment payeeMSP ${payeeMSP}…`
  );

  if (payeeMSP !== MSP_ID) {
    await cp.checkpointChaincodeEvent(evt);
    return; // not my bank
  }

  // fetch private payload from PDC that this peer already stores
  const payBytes = await contract.evaluateTransaction(
    "GetPrivatePayment", // chain-code helper
    { arguments: [id] }
  );
  const pay = JSON.parse(Buffer.from(payBytes).toString("utf8"));

  // now you have pay.amount, pay.payeeAcct, etc.
  // await creditCoreLedger(pay.payeeAcct, pay.amount);
  console.log(`Crediting ${pay.payeeAcct} with ₦${pay.amount}`);

  await contract.submitTransaction("SettlePayment", id);
  console.log(`Payment ${id} marked as SETTLED`);

  await cp.checkpointChaincodeEvent(evt);

  // // 1. credit core ledger (placeholder)
  // console.log(`Crediting ${payload.payeeAcct} with ₦${payload.amount}`);

  // // 2. mark SETTLED on‑chain (idempotent)
  // await contract.submitTransaction("SettlePayment", payload.id);

  // // 3. push mobile notification (placeholder)
  // console.log(`Push notification → acct ${payload.payeeAcct}`);

  // // 4. checkpoint last processed event
  // await cp.checkpointChaincodeEvent(evt);
}

async function startListener(gateway) {
  const network = gateway.getNetwork(CHANNEL);
  const contract = network.getContract(CHAINCODE);
  // const cp = checkpointers.file(CHECKPOINT_FILE);
  const cp = checkpointers.inMemory();

  while (true) {
    let stream;
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

let gatewayGlobal;

app.post("/payments", async (req, res) => {
  try {
    const gw = gatewayGlobal;
    const network = gw.getNetwork(CHANNEL);
    const contract = network.getContract(CHAINCODE);

    const { payerAcct, payeeMSP, payeeAcct, amount } = req.body;
    const paymentID = crypto.randomUUID().toString();

    const payJson = JSON.stringify({
      payerMSP: MSP_ID,
      payerAcct,
      payeeMSP,
      payeeAcct,
      amount,
    });

    await contract.submit("CreatePayment", {
      transientData: {
        payment: Buffer.from(payJson), // <-- value must be bytes
      },
      arguments: [paymentID],
      endorsingOrganizations: [MSP_ID, payeeMSP],
    });

    res.status(201).json({ id: paymentID, status: "PENDING" });
  } catch (err) {
    console.error(err);
    res.status(500).json({
      error: "Could not create payment",
      message: err.details[0]["message"],
    });
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

// GET /blocks  → return *all* blocks (simple demo; streams JSON array)
app.get('/blocks', async (_req, res) => {
  try {
    const chainInfo = await qscc.getChainInfo();
    const height = chainInfo.getHeight();

    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.write('['); // start array
    for await (const blk of qscc.iterateBlocks()) {
      const num = blk.getHeader()?.getNumber();
      const txCount = blk.getData()?.getDataList().length ?? 0;
      res.write(JSON.stringify({ number: num, txCount }));
      if (num < height - 1n) res.write(',');
    }
    res.end(']');
  } catch (err) {
    console.error(err);
    res.status(500).json({ error: 'Could not fetch blocks' });
  }
});

// (optional) GET /blocks/:num  for single‑block detail
app.get('/blocks/:num', async (req, res) => {
  try {
    const blk = await qscc.getBlockByNumber(req.params.num);
    res.json(blk.toJSON());
  } catch (err) {
    res.status(404).json({ error: 'Block not found' });
  }
});

/* ---------- bootstrap everything ------------------------------------------- */
(async () => {
  gatewayGlobal = await newGateway();
  // after gateway connection established:
  qscc = await buildQsccHelpers(gatewayGlobal, CHANNEL);
  startListener(gatewayGlobal).catch(console.error);
  app.listen(4000, () => console.log("Bank‑API listening on 4000"));
})();
