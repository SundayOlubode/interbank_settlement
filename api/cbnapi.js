/* -------------------------------------------------------------
   cbn-api/cbnapi.js — CBN Payment & Settlement Service
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
const MSP_ID = process.env.CBN_MSP_ID ?? "CentralBankMSP";
const PEER_ENDPOINT = process.env.CBN_PEER_ENDPOINT ?? "localhost:11051";
const TLS_CERT_PATH = process.env.CBN_TLS_CERT_PATH;
const ID_CERT_PATH = process.env.CBN_ID_CERT_PATH;
const KEY_PATH = process.env.CBN_KEY_PATH;
const CHANNEL = process.env.CHANNEL;
const CHAINCODE = process.env.CHAINCODE_NAME;

// Settlement configuration
const SETTLEMENT_INTERVAL_MS = 2 * 60 * 1000; // 2 minutes

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

let gateway;
let settlementTimer;
let isSettlementRunning = false;

/* ---------- payment acknowledgment processing ------------------------------ */
async function processPaymentAcknowledgedEvent(evt, contract, cp) {
  const paymentData = JSON.parse(Buffer.from(evt.payload).toString("utf8"));
  const { id, payerMSP, payeeMSP } = paymentData;

  console.log(
    `🔄 Processing PaymentAcknowledged: ${id} (${payerMSP} → ${payeeMSP})`
  );

  try {
    // CBN immediately batches the acknowledged payment
    await contract.submitTransaction(
      "BatchAcknowledgedPaymentSimple",
      id,
      payerMSP,
      payeeMSP
    );

    console.log(`✅ Payment ${id} batched successfully`);
  } catch (err) {
    console.error(`❌ Failed to batch payment ${id}:`, err.message);
  }

  await cp.checkpointChaincodeEvent(evt);
}

/* ---------- settlement execution ------------------------------------------- */
async function executeSettlementCycle() {
  if (isSettlementRunning) {
    console.log("⏭️  Settlement cycle already running, skipping...");
    return;
  }

  isSettlementRunning = true;
  const cycleStart = new Date();

  console.log(`\n🏦 ================= CBN SETTLEMENT CYCLE =================`);
  console.log(`⏰ Cycle Start: ${formatTime(cycleStart)}`);
  console.log(`========================================================`);

  try {
    const network = gateway.getNetwork(CHANNEL);
    const contract = network.getContract(CHAINCODE);

    // Execute netting settlement of all batched payments
    console.log("💰 Executing netting settlement...");
    await executeNettingSettlement(contract);

    const cycleEnd = new Date();
    const duration = cycleEnd - cycleStart;

    console.log(`✅ Settlement cycle completed in ${duration}ms`);
    console.log(`⏰ Cycle End: ${formatTime(cycleEnd)}`);
    console.log(`🔄 Next cycle: ${formatTime(getNextSettlementTime())}`);
    console.log(`========================================================\n`);
  } catch (error) {
    console.error(`❌ Settlement cycle failed:`, error.message);
  } finally {
    isSettlementRunning = false;
  }
}

/* ---------- netting-based settlement execution ----------------------------- */
async function executeNettingSettlement(contract) {
  try {
    console.log("🧮 Step 1: Calculating netting offsets...");

    // Step 1: Calculate netting offsets
    const calculationResult = await contract.evaluateTransaction(
      "CalculateNettingOffsets"
    );
    const calculation = JSON.parse(
      Buffer.from(calculationResult).toString("utf8")
    );

    // const payload = JSON.parse(Buffer.from(calcBytes).toString("utf8"));

    console.log(
      `📊 Calculated net positions for ${
        Object.keys(calculation.netPositions).length
      } banks`
    );
    console.log(`💰 Total payments to settle: ${calculation.totalPayments}`);

    if (calculation.totalPayments === 0) {
      console.log("ℹ️  No batched payments found for settlement");
      return;
    }

    console.log("⚡ Step 2: Applying netting offsets...");

    // Step 2: Apply netting offsets with calculation as transient data
    const applicationResult = await contract.submit(
      "ApplyNettingOffsets",
      {
        transientData: {
          // nettingOffsets: Buffer.from(calculationResult),
          nettingOffsets: calculationResult,
        },
      }
    );

    console.log("✅ Netting offsets applied successfully");

    const result = JSON.parse(Buffer.from(applicationResult).toString("utf8"));

    console.log("📊 Settlement Results:");

    // Log detailed results
    console.log("📊 Netting Settlement Results:");
    console.log(`   Payments Settled: ${result.settledPayments}`);
    console.log(`   Failed Payments: ${result.failedPayments}`);
    console.log(
      `   Total Net Amount: ${
        result.totalNetAmount?.toLocaleString() || 0
      } eNaira`
    );

    // Log net positions and bank movements
    if (
      calculation.netPositions &&
      Object.keys(calculation.netPositions).length > 0
    ) {
      console.log("💰 Net Positions:");
      Object.entries(calculation.netPositions).forEach(([bank, position]) => {
        if (position !== 0) {
          const status = position > 0 ? "RECEIVES" : "OWES";
          const amount = Math.abs(position).toLocaleString();
          console.log(`     ${bank}: ${status} ${amount} eNaira`);
        }
      });
    }
  } catch (error) {
    console.error("❌ Netting settlement failed:", error.message);
    console.log(error);
    throw error;
  }
}

/* ---------- settlement timer management ------------------------------------ */
function startSettlementTimer() {
  // Calculate time until next 2-minute boundary
  const now = Date.now();
  const nextBoundary =
    Math.ceil(now / SETTLEMENT_INTERVAL_MS) * SETTLEMENT_INTERVAL_MS;
  const waitTime = nextBoundary - now;

  console.log(`⏰ Settlement timer starting...`);
  console.log(
    `⏰ Next settlement in ${Math.round(waitTime / 1000)}s at ${formatTime(
      new Date(nextBoundary)
    )}`
  );

  // Wait until the next boundary, then start regular interval
  setTimeout(() => {
    // Execute immediately at boundary
    executeSettlementCycle();

    // Then set up regular 2-minute interval
    settlementTimer = setInterval(() => {
      executeSettlementCycle();
    }, SETTLEMENT_INTERVAL_MS);

    console.log(`🔄 Settlement timer active - running every 2 minutes`);
  }, waitTime);
}

function stopSettlementTimer() {
  if (settlementTimer) {
    clearInterval(settlementTimer);
    settlementTimer = null;
    console.log("⏹️  Settlement timer stopped");
  }
}

/* ---------- event listener ------------------------------------------------- */
async function startEventListener(gateway) {
  const network = gateway.getNetwork(CHANNEL);
  const contract = network.getContract(CHAINCODE);
  const cp = checkpointers.inMemory();

  console.log(
    "🎧 CBN Event Listener started, waiting for PaymentAcknowledged events..."
  );

  while (true) {
    let stream;
    try {
      stream = await network.getChaincodeEvents(CHAINCODE, { checkpoint: cp });

      for await (const evt of stream) {
        if (evt.eventName === "PaymentAcknowledged") {
          await processPaymentAcknowledgedEvent(evt, contract, cp);
        }
        // Ignore other events for now
      }
    } catch (err) {
      console.error("🔌 Event stream dropped, reconnecting…", err);
      await new Promise((resolve) => setTimeout(resolve, 5000)); // Wait 5s before reconnecting
    } finally {
      stream?.close?.();
    }
  }
}

/* ---------- utility functions ---------------------------------------------- */
function formatTime(date) {
  return date.toISOString().substr(11, 8);
}

function getNextSettlementTime() {
  const now = Date.now();
  const nextBoundary =
    Math.ceil(now / SETTLEMENT_INTERVAL_MS) * SETTLEMENT_INTERVAL_MS;
  return new Date(nextBoundary);
}

/* ---------- express API (minimal) ------------------------------------------ */
const app = express();
app.use(express.json());
app.use(morgan("dev"));

app.get("/health", (req, res) => {
  res.json({
    status: "CBN Payment & Settlement Service Running",
    msp: MSP_ID,
    settlementInterval: `${SETTLEMENT_INTERVAL_MS / 1000}s`,
    nextSettlement: getNextSettlementTime(),
    isSettlementRunning,
  });
});

app.get("/api/settlement/status", async (req, res) => {
  try {
    const network = gateway.getNetwork(CHANNEL);
    const contract = network.getContract(CHAINCODE);

    const statsResult = await contract.evaluateTransaction(
      "GetSettlementStatistics"
    );
    const stats = JSON.parse(Buffer.from(statsResult).toString("utf8"));

    res.json({
      success: true,
      nextSettlement: getNextSettlementTime(),
      isSettlementRunning,
      statistics: stats,
      timestamp: new Date().toISOString(),
    });
  } catch (error) {
    res.status(500).json({
      error: "Failed to get settlement status",
      message: error.message,
    });
  }
});

app.post("/api/settlement/execute", async (req, res) => {
  try {
    if (isSettlementRunning) {
      return res.status(409).json({
        error: "Settlement cycle already running",
        message: "Please wait for current cycle to complete",
      });
    }

    // Execute manual settlement cycle
    executeSettlementCycle();

    res.json({
      success: true,
      message: "Manual settlement cycle started",
      timestamp: new Date().toISOString(),
    });
  } catch (error) {
    res.status(500).json({
      error: "Failed to execute settlement",
      message: error.message,
    });
  }
});

/* ---------- graceful shutdown ----------------------------------------------- */
process.on("SIGINT", () => {
  console.log("\n🛑 Shutting down CBN Payment & Settlement Service...");
  stopSettlementTimer();
  if (gateway) {
    gateway.close();
  }
  process.exit(0);
});

process.on("SIGTERM", () => {
  console.log(
    "\n🛑 Received SIGTERM, shutting down CBN Payment & Settlement Service..."
  );
  stopSettlementTimer();
  if (gateway) {
    gateway.close();
  }
  process.exit(0);
});

/* ---------- bootstrap everything ------------------------------------------- */
(async () => {
  try {
    gateway = await newGateway();
    console.log("✅ Connected to CBN Fabric Gateway");
    console.log(`🏦 MSP: ${MSP_ID}`);

    // Start settlement timer (2-minute intervals)
    startSettlementTimer();

    // Start event listener in background
    startEventListener(gateway).catch(console.error);

    app.maxConnections = 1000; // Set max connections to handle load
    app.timeout = 30000; // Set request timeout to 30 seconds

    app.listen(4002, () => {
      console.log("🏦 CBN Payment & Settlement API listening on port 4002");
      console.log("⏰ Automated settlement every 2 minutes");
      console.log(
        "🎯 Monitoring PaymentAcknowledged events for auto-batching..."
      );
    });
  } catch (err) {
    console.error("❌ Failed to start CBN Payment & Settlement Service:", err);
    process.exit(1);
  }
})();
