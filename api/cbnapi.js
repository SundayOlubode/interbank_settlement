/* -------------------------------------------------------------
   cbn-api/cbnapi.js — CBN Event Listener for Payment Settlement
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
const MSP_ID = process.env.CBN_MSP_ID ?? "CentralBankPeerMSP";
const PEER_ENDPOINT = process.env.CBN_PEER_ENDPOINT ?? "localhost:11051";
const TLS_CERT_PATH = process.env.CBN_TLS_CERT_PATH;
const ID_CERT_PATH = process.env.CBN_ID_CERT_PATH;
const KEY_PATH = process.env.CBN_KEY_PATH;
const CHANNEL = process.env.CHANNEL;
const CHAINCODE = process.env.CHAINCODE_NAME;
const CHECKPOINT_FILE =
  process.env.CHECKPOINT_FILE ?? "./cbn-settlement-events.chk";

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

/* ---------- payment settlement processing ---------------------------------- */
async function processSettlementEvent(evt, contract, cp) {
  const paymentData = JSON.parse(Buffer.from(evt.payload).toString("utf8"));

  const { id } = paymentData;

  console.log(`Processing event ${evt.eventName} for payment ID ${id}…`);

  try {
    // Call SettlePayment transaction with the payment ID
    const result = await contract.submit("SettlePayment", {
      arguments: [Buffer.from(evt.payload)],
    });
    console.log(
      `Payment ${id} successfully ${Buffer.from(result)} through CBN`
    );
  } catch (err) {
    console.error(`Failed to settle payment ${id}:`, err);
  }

  await cp.checkpointChaincodeEvent(evt);
}

/* ---------- payment settlement processing ---------------------------------- */
async function processSettlementEventV2(evt, contract, cp) {
  const paymentData = JSON.parse(Buffer.from(evt.payload).toString("utf8"));
  const { id, payerMSP, payeeMSP } = paymentData;
  console.log(
    `Processing event ${evt.eventName} for payment ID ${id} from ${payerMSP} to ${payeeMSP}...`
  );

  try {
    // Step 1: Attempt to debit the payer's account
    console.log(`Debiting from ${payerMSP}...`);
    const debitResult = await contract.submit("DebitAccount", {
      arguments: [Buffer.from(evt.payload)],
    });

    const debitStatus = Buffer.from(debitResult).toString();
    console.log(`Debit result: ${debitStatus}`);

    // Check if debit was successful
    if (debitStatus === "QUEUED") {
      console.log(
        `Payment ${id} queued due to insufficient funds in ${payerMSP}. Settlement will be attempted later through netting.`
      );
      await cp.checkpointChaincodeEvent(evt);
      return; // Exit early - do not proceed to credit
    }

    if (debitStatus !== "SUCCESS") {
      throw new Error(`Unexpected debit result: ${debitStatus}`);
    }

    // Step 2: If debit was successful, proceed to credit the payee
    console.log(`Crediting ${payeeMSP}...`);
    await contract.submit("CreditAccount", {
      arguments: [Buffer.from(evt.payload)],
    });

    console.log(
      `Payment ${id} successfully settled: ${payerMSP} -> ${payeeMSP} in eNaira`
    );
  } catch (err) {
    console.error(`Failed to settle payment ${id}:`, err);

    // If credit failed after successful debit, we need to handle this scenario
    // This would require implementing a rollback mechanism or compensation transaction
    if (err.message && err.message.includes("CreditAccount")) {
      console.error(
        `CRITICAL: Debit succeeded but credit failed for payment ${id}. Manual intervention may be required.`
      );
    }
  }

  await cp.checkpointChaincodeEvent(evt);
}

async function startListener(gateway) {
  const network = gateway.getNetwork(CHANNEL);
  const contract = network.getContract(CHAINCODE);
  const cp = checkpointers.inMemory();

  console.log(
    "🎧 CBN Settlement Listener started, waiting for PaymentAcknowledged events..."
  );

  while (true) {
    let stream;
    stream = await network.getChaincodeEvents(CHAINCODE, { checkpoint: cp });

    try {
      for await (const evt of stream) {
        if (evt.eventName !== "PaymentAcknowledged") continue;
        await processSettlementEventV2(evt, contract, cp);
      }
    } catch (err) {
      console.error("🔌 event stream dropped, reconnecting…", err);
    } finally {
      stream?.close?.();
    }
  }
}

/* ---------- express API (minimal, mainly for health check) ----------------- */
const app = express();
app.use(express.json());
app.use(morgan("dev"));

app.get("/health", (req, res) => {
  res.json({ status: "CBN Settlement Service Running" });
});

/* ---------- bootstrap everything ------------------------------------------- */
(async () => {
  try {
    const gateway = await newGateway();
    console.log("✅ Connected to CBN Fabric Gateway");

    startListener(gateway).catch(console.error);

    app.listen(4002, () => {
      console.log("🏦 CBN Settlement API listening on port 4002");
      console.log("🎯 Monitoring PaymentCompleted events for settlement...");
    });
  } catch (err) {
    console.error("❌ Failed to start CBN API:", err);
    process.exit(1);
  }
})();
