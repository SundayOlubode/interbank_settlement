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
import { extractSimpleBlockData } from "./helper/extract-block-data.js";

const userAccounts = {
  "0506886519": {
    firstname: "Oluwaseun",
    lastname: "Adebanjo",
    middlename: "Temitope",
    bvn: "22133455678",
    gender: "Female",
    balance: 20000,
    birthdate: "15-04-1990",
  },
  "0506886390": {
    firstname: "Emeka",
    lastname: "Okafor",
    middlename: "Chukwuemeka",
    gender: "Male",
    phone: "08134567890",
    birthdate: "02-11-1985",
    bvn: "23455677890",
    balance: 45000,
  },
};

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

    console.log("User accounts:", user);

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

    // console.log(`Debiting ${payerAcct} with ₦${amount}`);
    const paymentID = crypto.randomUUID().toString();
    const bvn = user.bvn;

    if (!bvn) {
      return res.status(400).json({
        error: "Missing BVN",
        message: `Account ${payerAcct} does not have a valid BVN`,
      });
    }

    // add bvn
    const payJson = JSON.stringify({
      id: paymentID,
      payerMSP: MSP_ID,
      payerAcct,
      payeeMSP,
      payeeAcct,
      amount,
      user: {
        firstname: user.firstname,
        lastname: user.lastname,
        gender: user.gender,
        birthdate: user.birthdate,
        bvn: user.bvn,
      },
    });

    await contract.submit("CreatePayment", {
      transientData: {
        payment: Buffer.from(payJson), // <-- value must be bytes
      },
      // arguments: [paymentID],
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

// API endpoint to get ALL data in a private collection
app.get("/private-data/:collection/all", async (req, res) => {
  const { collection } = req.params;

  try {
    const network = gatewayGlobal.getNetwork(CHANNEL);
    const contract = network.getContract(CHAINCODE);

    // Call chaincode function to get ALL private data (no range specified)
    const result = await contract.evaluateTransaction(
      "GetAllPrivateData",
      collection
    );

    const privateDataList = JSON.parse(Buffer.from(result).toString("utf8"));

    res.json({
      collection: collection,
      totalRecords: privateDataList.length,
      data: privateDataList,
      timestamp: new Date().toISOString(),
    });
  } catch (error) {
    console.error("Error getting all private data:", error);
    res.status(500).json({
      error: "Could not retrieve all private data",
      message: error.message,
      collection: collection,
    });
  }
});

// Complete /blocks route implementation
app.get("/blocks", async (req, res) => {
  const {
    businessOnly = false,
    chaincodeName = null,
    txType = null,
    startBlock = null,
    endBlock = null,
  } = req.query;

  let height;
  try {
    const chainInfo = await qscc.getChainInfo();
    height = chainInfo.getHeight();
    height = typeof height === "bigint" ? Number(height) : height;
  } catch (err) {
    console.error("Error fetching chain info:", err);
    return res.status(500).json({ error: "Could not fetch chain info" });
  }

  console.log(`Streaming ${height} blocks with transaction data...`);
  console.log("Filters:", {
    businessOnly,
    chaincodeName,
    txType,
    startBlock,
    endBlock,
  });

  // Set response headers for streaming JSON
  res.writeHead(200, {
    "Content-Type": "application/json",
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Methods": "GET",
    "Access-Control-Allow-Headers": "Content-Type",
  });
  res.write("[");

  let isFirstBlock = true;
  let processedBlocks = 0;

  try {
    for await (const block of qscc.iterateBlocks()) {
      const blockData = extractSimpleBlockData(block);

      // Apply block range filter
      if (startBlock && blockData.blockNumber < parseInt(startBlock)) {
        continue;
      }
      if (endBlock && blockData.blockNumber > parseInt(endBlock)) {
        break;
      }

      // Filter transactions if requested
      if (businessOnly || chaincodeName || txType) {
        const originalTxCount = blockData.transactions.length;

        blockData.transactions = blockData.transactions.filter((tx) => {
          // Filter by transaction type
          if (txType && tx.typeDescription !== txType) {
            return false;
          }

          // For chaincode-related filters
          if (businessOnly || chaincodeName) {
            // Skip non-chaincode transactions
            if (!tx.chaincodeData || tx.chaincodeData.length === 0) {
              return !businessOnly;
            }

            return tx.chaincodeData.some((cc) => {
              // Filter out lifecycle transactions if businessOnly is true
              if (businessOnly && cc.isLifecycleTransaction) {
                return false;
              }

              // Filter by chaincode name if specified
              if (chaincodeName && cc.chaincodeName !== chaincodeName) {
                return false;
              }

              return true;
            });
          }

          return true;
        });

        // Add filter info to block data
        blockData.filterInfo = {
          originalTxCount: originalTxCount,
          filteredTxCount: blockData.transactions.length,
          filtersApplied: { businessOnly, chaincodeName, txType },
        };
      }

      // Only include blocks that have transactions after filtering (or if no filters applied)
      const shouldIncludeBlock =
        (!businessOnly && !chaincodeName && !txType) ||
        blockData.transactions.length > 0;

      if (shouldIncludeBlock) {
        if (!isFirstBlock) {
          res.write(",");
        }

        res.write(JSON.stringify(blockData, null, 2));
        isFirstBlock = false;
        processedBlocks++;
      }

      // Optional: Add a limit to prevent overwhelming responses
      if (processedBlocks >= 100) {
        console.log("Reached maximum block limit (100), stopping...");
        break;
      }
    }

    res.end("]");
    console.log(`Successfully streamed ${processedBlocks} blocks`);
  } catch (loopErr) {
    console.error("Error while streaming blocks:", loopErr);
    // Close the JSON array so the client does not hang
    res.end("]");
  }
});

// (optional) GET /blocks/:num  for single‑block detail
app.get("/blocks/:num", async (req, res) => {
  try {
    const blk = await qscc.getBlockByNumber(req.params.num);
    res.json(blk.toJSON());
  } catch (err) {
    res.status(404).json({ error: "Block not found" });
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
