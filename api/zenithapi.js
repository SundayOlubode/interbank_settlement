/* -------------------------------------------------------------
   bank-api/server.js — Express + Hyperledger Fabric Gateway
   ------------------------------------------------------------- */
import express from "express";
import morgan from "morgan";
import { promises as fs } from "node:fs";
import * as crypto from "node:crypto";

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
import { extractSimpleBlockData } from "./helper/extract-block-data.js";

const userAccounts = {
  1012345678: {
    firstname: "Musa",
    lastname: "Garba",
    middlename: "Ibrahim",
    bvn: "42344556789",
    gender: "Male",
    balance: 9500000,
    birthdate: "18-08-1994",
  },
  1012456789: {
    firstname: "Ngozi",
    lastname: "Ikwuemesi",
    middlename: "Adanna",
    bvn: "43455667891",
    gender: "Female",
    balance: 57500000,
    birthdate: "02-10-1985",
  },
  1012567890: {
    firstname: "Gbenga",
    lastname: "Olumide",
    middlename: "Ayodeji",
    bvn: "44566778902",
    gender: "Male",
    balance: 4450000,
    birthdate: "21-12-1996",
  },
};

/* ---------- env / constants ------------------------------------------------ */
const MSP_ID = process.env.ZENITHBANK_MSP_ID ?? "ZenithBankMSP";
const PEER_ENDPOINT = process.env.ZENITHBANK_PEER_ENDPOINT ?? "localhost:9051";
const TLS_CERT_PATH = process.env.ZENITHBANK_TLS_CERT_PATH;
const ID_CERT_PATH = process.env.ZENITHBANK_ID_CERT_PATH;
const KEY_PATH = process.env.ZENITHBANK_KEY_PATH;
const CHANNEL = process.env.CHANNEL;
const CHAINCODE = process.env.CHAINCODE_NAME;
const CHECKPOINT_FILE = process.env.CHECKPOINT_FILE ?? "./payment-events.chk";

// Global map to track pending payment acknowledgments
const pendingAcknowledgments = new Map();

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

// Helper function to wait for PaymentAcknowledged event with timeout
function waitForPaymentAcknowledgment(paymentID, timeout = 10000) {
  return new Promise((resolve, reject) => {
    const timeoutId = setTimeout(() => {
      // Clean up pending acknowledgment
      pendingAcknowledgments.delete(paymentID);
      reject(new Error("Payment acknowledgment timeout"));
    }, timeout);

    // Store the resolve function to be called when event is received
    pendingAcknowledgments.set(paymentID, {
      resolve: (ackData) => {
        clearTimeout(timeoutId);
        pendingAcknowledgments.delete(paymentID);
        resolve(ackData);
      },
      reject: (error) => {
        clearTimeout(timeoutId);
        pendingAcknowledgments.delete(paymentID);
        reject(error);
      },
    });
  });
}

/* ---------- payment processing --------------------------------------------- */
async function processPaymentEvent(evt, contract, cp) {
  const { id, payerMSP, payeeMSP } = JSON.parse(
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
  const payBytes = await contract.submit("GetIncomingPayment", {
    arguments: [id],
  });
  const pay = JSON.parse(Buffer.from(payBytes).toString("utf8"));

  // await creditCoreLedger(pay.payeeAcct, pay.amount);
  if (!userAccounts[pay.payeeAcct]) {
    console.error(`Account ${pay.payeeAcct} not found`);
    return;
  }

  userAccounts[pay.payeeAcct].balance += pay.amount;
  console.log(
    `Credited account ${pay.payeeAcct} (${
      userAccounts[pay.payeeAcct].firstname
    }) with ₦${pay.amount}. New balance: ₦${
      userAccounts[pay.payeeAcct].balance
    }`
  );

  console.log(`Crediting ${pay.payeeAcct} with ₦${pay.amount}`);

  await contract.submit("AcknowledgePayment", {
    arguments: [JSON.stringify({ id, payerMSP, payeeMSP, batchWindow: 0 })],
  });

  // await contract.submitTransaction("SettlePayment", id);
  console.log(`Payment ${id} marked as SETTLED`);

  await cp.checkpointChaincodeEvent(evt);
}

// Handle PaymentAcknowledged events
async function processPaymentAcknowledgedEvent(evt, cp) {
  try {
    const ackData = JSON.parse(Buffer.from(evt.payload).toString("utf8"));

    if (ackData.payerMSP !== MSP_ID) return;

    const { id: paymentID } = ackData;

    console.log(
      `Received PaymentAcknowledged for payment ${paymentID}:`,
      ackData
    );

    // Check if someone is waiting for this acknowledgment
    const pending = pendingAcknowledgments.get(paymentID);
    if (pending) {
      pending.resolve(ackData);
    }

    await cp.checkpointChaincodeEvent(evt);
  } catch (error) {
    console.error("Error processing PaymentAcknowledged event:", error);
    await cp.checkpointChaincodeEvent(evt);
  }
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
        console.log(`Received event: ${evt.eventName}`);

        if (evt.eventName === "PaymentPending") {
          await processPaymentEvent(evt, contract, cp);
        } else if (evt.eventName === "PaymentAcknowledged") {
          await processPaymentAcknowledgedEvent(evt, cp);
        } else {
          // Checkpoint other events
          await cp.checkpointChaincodeEvent(evt);
        }
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

    const user = userAccounts[payerAcct];
    if (!user) {
      return res.status(400).json({
        error: "Invalid payer account",
        message: `Account ${payerAcct} does not exist`,
      });
    }

    // Check if recipient is not same account as payer
    if (payerAcct === payeeAcct) {
      return res.status(400).json({
        error: "Invalid transaction",
        message: `Payer account ${payerAcct} cannot be the same as payee account ${payeeAcct}`,
      });
    }

    const payerBalance = user.balance;

    if (payerBalance < amount) {
      return res.status(400).json({
        error: "Insufficient funds",
        message: `Account ${payerAcct} has insufficient funds (₦${payerBalance}) for this transaction (₦${amount})`,
      });
    }

    // Deduct amount from payer's account
    user.balance -= amount;
    console.log(
      `Debited account ${payerAcct} (${user.firstname}) with ₦${amount}. New balance: ₦${user.balance}`
    );

    const paymentID = crypto.randomUUID().toString();
    const bvn = user.bvn;

    if (!bvn) {
      // Refund the user since we can't process without BVN
      user.balance += amount;
      return res.status(400).json({
        error: "Missing BVN",
        message: `Account ${payerAcct} does not have a valid BVN`,
      });
    }

    // Prepare payment data
    const payJson = JSON.stringify({
      id: paymentID,
      payerMSP: MSP_ID,
      payerAcct,
      payeeMSP,
      payeeAcct,
      amount,
      timestamp: Date.now(),
      user: {
        firstname: user.firstname,
        lastname: user.lastname,
        gender: user.gender,
        birthdate: user.birthdate,
        bvn: user.bvn,
      },
    });

    // Start waiting for acknowledgment before submitting transaction
    const acknowledgmentPromise = waitForPaymentAcknowledgment(
      paymentID,
      10000
    ); // 10 second timeout

    console.log(
      `Submitting payment ${paymentID} and waiting for acknowledgment...`
    );

    try {
      // Submit the transaction
      await contract.submit("CreatePayment", {
        transientData: {
          payment: Buffer.from(payJson),
        },
        endorsingOrganizations: [MSP_ID, payeeMSP],
      });

      console.log(
        `Payment ${paymentID} submitted to blockchain, waiting for acknowledgment...`
      );

      try {
        // Wait for the PaymentAcknowledged event (with 10s timeout)
        const ackData = await acknowledgmentPromise;

        console.log(`Payment ${paymentID} acknowledged:`, ackData);

        // Return success response with acknowledgment data
        res.status(201).json({
          id: paymentID,
          status: "Successful",
          message: "Payment created and acknowledged by settlement system",
          acknowledgment: ackData,
          timestamp: new Date().toISOString(),
        });
      } catch (timeoutError) {
        console.warn(
          `Payment ${paymentID} submitted but acknowledgment timed out:`,
          timeoutError.message
        );

        // Return success but indicate acknowledgment timeout
        res.status(202).json({
          id: paymentID,
          status: "PENDING",
          message:
            "Payment created successfully but acknowledgment timed out. Payment is being processed.",
          warning:
            "Settlement system acknowledgment not received within 10 seconds",
          timestamp: new Date().toISOString(),
        });
      }
    } catch (submitError) {
      // Transaction submission failed, refund the user
      user.balance += amount;
      console.log(
        `Refunded account ${payerAcct} with ₦${amount} due to transaction failure`
      );

      throw submitError; // Re-throw to be caught by outer catch
    }
  } catch (err) {
    console.error("Payment creation error:", err);

    res.status(500).json({
      error: "Could not create payment",
      message: err.details ? err.details[0]["message"] : err.message,
    });
  }
});

/* ---------- bootstrap everything ------------------------------------------- */
(async () => {
  try {
    console.log(`Starting ${MSP_ID} API server...`);

    gatewayGlobal = await newGateway();
    console.log("Gateway connection established");

    // after gateway connection established:
    const network = gatewayGlobal.getNetwork(CHANNEL);
    qscc = await buildQsccHelpers(gatewayGlobal, CHANNEL);

    // Start event listeners (acknowledgment is now integrated into startListener)
    console.log("Setting up event listeners...");
    startListener(gatewayGlobal).catch(console.error);

    app.maxConnections = 1000; // Set max connections to handle load
    app.timeout = 30000; // Set request timeout to 30 seconds

    app.listen(4003, () => {
      console.log(`${MSP_ID} API listening on port 4003`);
      console.log("Event listeners configured and running");
    });
  } catch (error) {
    console.error("Failed to start application:", error);
    process.exit(1);
  }
})();
