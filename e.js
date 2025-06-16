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