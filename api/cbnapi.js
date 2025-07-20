/* -------------------------------------------------------------
   cbn-api/cbnapi.js — CBN 2-Minute Batch Settlement Service
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
const CHECKPOINT_FILE = process.env.CHECKPOINT_FILE ?? "./cbn-settlement-events.chk";

// Settlement configuration
const SETTLEMENT_INTERVAL_MS = 2 * 60 * 1000; // 2 minutes
const MULTILATERAL_NETTING_ENABLED = process.env.MULTILATERAL_NETTING_ENABLED !== 'false';

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

/* ---------- batch window utilities ----------------------------------------- */
function getCurrentBatchWindow() {
  return Math.floor(Date.now() / 1000 / 120);
}

function getNextSettlementTime() {
  const now = Date.now();
  const nextBoundary = Math.ceil(now / SETTLEMENT_INTERVAL_MS) * SETTLEMENT_INTERVAL_MS;
  return new Date(nextBoundary);
}

function formatTime(date) {
  return date.toISOString().substr(11, 8);
}

/* ---------- multilateral settlement cycle ---------------------------------- */
async function executeMultilateralSettlementCycle() {
  if (isSettlementRunning) {
    console.log("⏭️  Settlement cycle already running, skipping...");
    return;
  }

  isSettlementRunning = true;
  const cycleStart = new Date();
  const batchWindow = getCurrentBatchWindow() - 1; // Process previous window
  
  console.log(`\n🏦 ================= CBN SETTLEMENT CYCLE =================`);
  console.log(`⏰ Cycle Start: ${formatTime(cycleStart)}`);
  console.log(`📊 Processing Batch Window: ${batchWindow}`);
  console.log(`========================================================`);

  try {
    const network = gateway.getNetwork(CHANNEL);
    const contract = network.getContract(CHAINCODE);

    // Phase 1: Execute Scheduled Multilateral Netting
    console.log("🔄 Phase 1: Executing Multilateral Netting...");
    await executeScheduledMultilateralNetting(contract);

    // Phase 2: Monitor Settlement Status
    console.log("📈 Phase 2: Checking Settlement Status...");
    await checkSystemSettlementStatus(contract);

    const cycleEnd = new Date();
    const duration = cycleEnd - cycleStart;
    
    console.log(`✅ Settlement cycle completed in ${duration}ms`);
    console.log(`⏰ Cycle End: ${formatTime(cycleEnd)}`);
    console.log(`🔄 Next cycle: ${formatTime(getNextSettlementTime())}`);
    console.log(`========================================================\n`);

  } catch (error) {
    console.error(`❌ Settlement cycle failed:`, error);
    
    // Send alert for failed settlement cycle
    await sendSettlementAlert('CYCLE_FAILED', {
      batchWindow,
      error: error.message,
      timestamp: new Date().toISOString()
    });
  } finally {
    isSettlementRunning = false;
  }
}

/* ---------- multilateral netting execution --------------------------------- */
async function executeScheduledMultilateralNetting(contract) {
  try {
    console.log("🌐 Executing system-wide multilateral netting...");
    
    // Call the chaincode function for scheduled multilateral netting
    const result = await contract.submitTransaction("ExecuteScheduledMultilateralNetting");
    
    const resultData = JSON.parse(Buffer.from(result).toString("utf8"));
    
    if (resultData.netPositions && Object.keys(resultData.netPositions).length > 0) {
      console.log("💰 Net Positions:");
      Object.entries(resultData.netPositions).forEach(([bank, position]) => {
        const status = position > 0 ? "RECEIVES" : position < 0 ? "OWES" : "BALANCED";
        const amount = Math.abs(position).toLocaleString();
        console.log(`   ${bank}: ${status} ${amount} eNaira`);
      });
      
      console.log(`📊 Total Payments Settled: ${resultData.updatesCount || 0}`);
      console.log(`💵 Total Amount Settled: ${(resultData.totalSettled || 0).toLocaleString()} eNaira`);
    } else {
      console.log("ℹ️  No queued payments found for multilateral netting");
    }

  } catch (error) {
    // Check if it's just "no payments to process" vs actual error
    if (error.message && error.message.includes("No queued payments")) {
      console.log("ℹ️  No queued payments found for multilateral netting");
    } else {
      console.error("❌ Multilateral netting failed:", error.message);
      throw error;
    }
  }
}

/* ---------- settlement status monitoring ----------------------------------- */
async function checkSystemSettlementStatus(contract) {
  try {
    const statusResult = await contract.evaluateTransaction("GetMultilateralNettingStatus");
    const status = JSON.parse(Buffer.from(statusResult).toString("utf8"));
    
    console.log("📊 System Settlement Status:");
    console.log(`   Total Queued Payments: ${status.totalQueuedPayments}`);
    console.log(`   Total Queued Amount: ${status.totalQueuedAmount.toLocaleString()} eNaira`);
    
    if (status.bankCounts && Object.keys(status.bankCounts).length > 0) {
      console.log("   Bank Breakdown:");
      Object.entries(status.bankCounts).forEach(([bank, count]) => {
        const amount = status.bankAmounts[bank] || 0;
        console.log(`     ${bank}: ${count} payments, ${amount.toLocaleString()} eNaira`);
      });
    }

    // Alert if too many queued payments
    if (status.totalQueuedPayments > 100) {
      await sendSettlementAlert('HIGH_QUEUE_COUNT', {
        queuedCount: status.totalQueuedPayments,
        queuedAmount: status.totalQueuedAmount
      });
    }

  } catch (error) {
    console.error("⚠️  Failed to check settlement status:", error.message);
  }
}

/* ---------- settlement alerts ---------------------------------------------- */
async function sendSettlementAlert(alertType, data) {
  const alert = {
    type: alertType,
    timestamp: new Date().toISOString(),
    msp: MSP_ID,
    data
  };
  
  console.log(`🚨 Settlement Alert [${alertType}]:`, JSON.stringify(data, null, 2));
  
  // Here you could integrate with monitoring systems, email, Slack, etc.
  // For now, just log the alert
}

/* ---------- settlement timer management ------------------------------------ */
function startSettlementTimer() {
  // Calculate time until next 2-minute boundary
  const now = Date.now();
  const nextBoundary = Math.ceil(now / SETTLEMENT_INTERVAL_MS) * SETTLEMENT_INTERVAL_MS;
  const waitTime = nextBoundary - now;
  
  console.log(`⏰ Settlement timer starting...`);
  console.log(`⏰ Next settlement in ${Math.round(waitTime / 1000)}s at ${formatTime(new Date(nextBoundary))}`);
  
  // Wait until the next boundary, then start regular interval
  setTimeout(() => {
    // Execute immediately at boundary
    executeMultilateralSettlementCycle();
    
    // Then set up regular 2-minute interval
    settlementTimer = setInterval(() => {
      executeMultilateralSettlementCycle();
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

/* ---------- event monitoring (for system events) -------------------------- */
async function startEventMonitoring(gateway) {
  const network = gateway.getNetwork(CHANNEL);
  const cp = checkpointers.inMemory();

  console.log("🎧 CBN Event Monitor started...");

  while (true) {
    let stream;
    try {
      stream = await network.getChaincodeEvents(CHAINCODE, { checkpoint: cp });

      for await (const evt of stream) {
        await processSystemEvent(evt, cp);
      }
    } catch (err) {
      console.error("🔌 Event stream dropped, reconnecting…", err);
      await new Promise(resolve => setTimeout(resolve, 5000)); // Wait 5s before reconnecting
    } finally {
      stream?.close?.();
    }
  }
}

async function processSystemEvent(evt, cp) {
  try {
    const eventData = JSON.parse(Buffer.from(evt.payload).toString("utf8"));
    
    switch (evt.eventName) {
      case "ScheduledMultilateralNettingExecuted":
        console.log(`📅 Scheduled netting completed:`, {
          updatesCount: eventData.updatesCount,
          totalSettled: eventData.totalSettled,
          banksProcessed: eventData.processedBanks?.length || 0
        });
        break;
        
      case "MultilateralOffsetExecuted":
        console.log(`🔄 Manual multilateral netting executed:`, {
          updatesCount: eventData.updatesCount,
          totalSettled: eventData.totalSettled
        });
        break;
        
      case "InsufficientFunds":
        console.log(`💳 Insufficient funds detected: ${eventData.payerMSP} - ${eventData.requiredAmount.toLocaleString()} eNaira required`);
        break;
        
      default:
        // Log other settlement-related events
        if (evt.eventName.includes("Settlement") || evt.eventName.includes("Netting")) {
          console.log(`📡 Event: ${evt.eventName}`, eventData);
        }
        break;
    }
  } catch (error) {
    console.error(`❌ Error processing event ${evt.eventName}:`, error);
  }
  
  await cp.checkpointChaincodeEvent(evt);
}

/* ---------- express API ---------------------------------------------------- */
const app = express();
app.use(express.json());
app.use(morgan("dev"));

app.get("/health", (req, res) => {
  res.json({ 
    status: "CBN 2-Minute Settlement Service Running",
    msp: MSP_ID,
    settlementInterval: `${SETTLEMENT_INTERVAL_MS / 1000}s`,
    nextSettlement: getNextSettlementTime(),
    isSettlementRunning
  });
});

app.get("/api/settlement/status", async (req, res) => {
  try {
    const network = gateway.getNetwork(CHANNEL);
    const contract = network.getContract(CHAINCODE);
    
    const statusResult = await contract.evaluateTransaction("GetMultilateralNettingStatus");
    const status = JSON.parse(Buffer.from(statusResult).toString("utf8"));
    
    res.json({
      success: true,
      currentBatchWindow: getCurrentBatchWindow(),
      nextSettlement: getNextSettlementTime(),
      isSettlementRunning,
      systemStatus: status,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      error: "Failed to get settlement status",
      message: error.message
    });
  }
});

app.post("/api/settlement/execute", async (req, res) => {
  try {
    if (isSettlementRunning) {
      return res.status(409).json({
        error: "Settlement cycle already running",
        message: "Please wait for current cycle to complete"
      });
    }

    // Execute manual settlement cycle
    executeMultilateralSettlementCycle();
    
    res.json({
      success: true,
      message: "Manual settlement cycle started",
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      error: "Failed to execute settlement",
      message: error.message
    });
  }
});

// Legacy bilateral netting endpoint (still available for manual use)
app.post("/api/netting/bilateral", async (req, res) => {
  const { BankA, BankB } = req.body;
  try {
    const network = gateway.getNetwork(CHANNEL);
    const contract = network.getContract(CHAINCODE);

    const calc = await contract.evaluateTransaction("CalculateBilateralOffset", BankA, BankB);
    const payload = JSON.parse(Buffer.from(calc).toString("utf8"));

    if (payload.offset === 0) {
      return res.json({
        success: true,
        message: `No bilateral offset needed between ${BankA} and ${BankB}`,
        offset: 0,
        updates: 0
      });
    }

    await contract.submitTransaction("ApplyBilateralOffset", BankA, BankB, {
      transientData: {
        offsetUpdate: Buffer.from(calc),
      },
    });

    console.log(`================= Manual Bilateral Netting =======================`);
    console.log(`Banks: ${BankA} & ${BankB}`);
    console.log(`Bilateral netting offset amount: ${payload.offset.toLocaleString()} eNaira`);
    console.log(`Number of txs offset: ${payload.updates.length}`);
    console.log("==================================================================");

    return res.status(200).json({
      success: true,
      message: `Bilateral offsetting performed successfully between ${BankA} and ${BankB}`,
      offset: payload.offset,
      updatesCount: payload.updates.length,
      timestamp: new Date().toISOString(),
    });
  } catch (error) {
    console.error("Bilateral offsetting error:", error);
    res.status(500).json({
      error: "Bilateral offsetting failed",
      message: error.message || "An unexpected error occurred during bilateral offsetting",
    });
  }
});

// Legacy multilateral netting endpoint (still available for manual use)
app.post("/api/netting/multilateral", async (req, res) => {
  try {
    const network = gateway.getNetwork(CHANNEL);
    const contract = network.getContract(CHAINCODE);

    const calcBytes = await contract.evaluateTransaction("CalculateMultilateralOffset");
    const payload = JSON.parse(Buffer.from(calcBytes).toString("utf8"));

    if (payload.updates.length === 0) {
      return res.json({
        success: true,
        message: "No queued payments found for multilateral netting",
        netPositions: {},
        updatesCount: 0
      });
    }

    await contract.submitTransaction("ApplyMultilateralOffset", {
      transientData: {
        multilateralUpdate: Buffer.from(calcBytes),
      },
    });

    console.log(`================= Manual Multilateral Netting =======================`);
    console.log("Multilateral NetPositions: ", payload.netPositions);
    console.log(`Number of txs updated: ${payload.updates.length}`);
    console.log("=====================================================================");

    return res.status(200).json({
      success: true,
      message: "Multilateral offsetting performed successfully",
      netPositions: payload.netPositions,
      updatesCount: payload.updates.length,
      timestamp: new Date().toISOString(),
    });
  } catch (error) {
    console.error("Multilateral offsetting error:", error);
    res.status(500).json({
      error: "Multilateral offsetting failed",
      message: error.message || "An unexpected error occurred during multilateral offsetting",
    });
  }
});

/* ---------- graceful shutdown ----------------------------------------------- */
process.on('SIGINT', () => {
  console.log('\n🛑 Shutting down CBN Settlement Service...');
  stopSettlementTimer();
  if (gateway) {
    gateway.close();
  }
  process.exit(0);
});

process.on('SIGTERM', () => {
  console.log('\n🛑 Received SIGTERM, shutting down CBN Settlement Service...');
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

    // Start event monitoring in background
    startEventMonitoring(gateway).catch(console.error);

    app.listen(4002, () => {
      console.log("🏦 CBN 2-Minute Settlement API listening on port 4002");
      console.log("⏰ Automated multilateral settlement every 2 minutes");
      console.log("🎯 Monitoring system-wide settlement events...");
    });
  } catch (err) {
    console.error("❌ Failed to start CBN Settlement Service:", err);
    process.exit(1);
  }
})();