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

const utf8Decoder = new TextDecoder();

/* ---------- env / constants ------------------------------------------------ */
const MSP_ID = process.env.GTBANK_MSP_ID ?? "GTBankMSP";
const PEER_ENDPOINT = process.env.GTBANK_PEER_ENDPOINT ?? "localhost:8051";
const TLS_CERT_PATH = process.env.GTBANK_TLS_CERT_PATH;
const ID_CERT_PATH = process.env.GTBANK_ID_CERT_PATH;
const KEY_PATH = process.env.GTBANK_KEY_PATH;
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
  });
}

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

  // Remove the Buffer conversion since it's now a string
  const payBytes = await contract.evaluateTransaction("GetPrivatePayment", id);
  const pay = JSON.parse(Buffer.from(payBytes).toString("utf8"));
  console.log(`Payment details:`, pay);

  // now you have pay.amount, pay.payeeAcct, etc.
  // await creditCoreLedger(pay.payeeAcct, pay.amount);
  console.log(`Crediting ${pay.payeeAcct} with ₦${pay.amount}`);

  await contract.submitTransaction("SettlePayment", id);
  console.log(`Payment ${id} marked as SETTLED`);

  await cp.checkpointChaincodeEvent(evt);
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
    const paymentID = crypto.randomUUID();

    await contract.submit("CreatePayment", {
      transientData: {
        payerAcct,
        payeeMSP,
        payeeAcct,
        amount: amount.toString(),
      },
      arguments: [paymentID],
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

/* ---------- bootstrap everything ------------------------------------------- */
(async () => {
  gatewayGlobal = await newGateway();
  startListener(gatewayGlobal).catch(console.error);
  app.listen(4001, () => console.log("Bank‑API listening on 4001"));
})();
