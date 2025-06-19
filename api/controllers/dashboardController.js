import { databaseService } from "../services/databaseService.js";
import { fabricService } from "../services/fabricService.js";

export class DashboardController {
  constructor() {
    this.fabricService = fabricService;
    this.contract = this.fabricService.getContract();
    this.databaseService = databaseService;
  }

  async getBankingOverview(req, res) {
    const result = await this.contract.submit("GetBankingOverview");

    const bankingData = JSON.parse(Buffer.from(result).toString("utf8"));

    console.log(
      `Successfully retrieved banking overview for MSP: ${bankingData.bankAccount}`
    );

    return res.status(200).json({
      success: true,
      data: bankingData,
      timestamp: new Date().toISOString(),
    });
  }

  async getTransactionAnalytics(req, res) {
    const result = await this.contract.submit("GetAllTransactionAnalytics");

    const analytics = JSON.parse(Buffer.from(result).toString("utf8"));

    return res.status(200).json({
      success: true,
      data: analytics,
      timestamp: new Date().toISOString(),
    });
  }

  /**
   * Retrieves all bank transactions from the ledger.
   * @param {Object} req - The request object.
   * @param {Object} res - The response object.
   */
  async getAllBankTransaction(req, res) {
    try {
      const bank = req.query.bank;
      //   const result = await this.contract.submit("GetAllBankTransactions");
      let transactions = await this.databaseService.getAllBankTransactions();

      //   let transactions = JSON.parse(Buffer.from(result).toString("utf8"));

      if (bank) {
        // Filter transactions by bank if specified
        transactions = transactions.filter(
          (tx) => tx.payeeMSP === `${bank}MSP` || tx.payerMSP === `${bank}MSP`
        );
      }

      return res.status(200).json({
        success: true,
        data: transactions,
        timestamp: new Date().toISOString(),
      });
    } catch (error) {
      console.error("Error retrieving bank transactions:", error);
    }
  }

  // Get all transactions and calculate net volume + count for each bank that AccessBankMSP has transacted with
  async getAllBankTransactionCount(req, res) {
    try {
      // Find all transactions
      const transactions = await this.databaseService.getAllBankTransactions();

      // Calculate net volume and count for each bank in the transactions list
      const bankStats = transactions.reduce((acc, tx) => {
        let otherBank = null;
        let volumeChange = 0;

        // Determine which bank AccessBankMSP transacted with and the volume impact
        if (tx.payerMSP === "AccessBankMSP") {
          otherBank = tx.payeeMSP;
          volumeChange = -tx.amount; // Money going out (negative)
        } else if (tx.payeeMSP === "AccessBankMSP") {
          otherBank = tx.payerMSP;
          volumeChange = tx.amount; // Money coming in (positive)
        }

        // Only process if AccessBankMSP is involved in the transaction
        if (otherBank) {
          // Initialize bank entry if it doesn't exist
          if (!acc[otherBank]) {
            acc[otherBank] = {
              netVolume: 0,
              count: 0,
              outgoingVolume: 0,
              incomingVolume: 0,
              outgoingCount: 0,
              incomingCount: 0,
            };
          }

          // Update net volume and count
          acc[otherBank].netVolume += volumeChange;
          acc[otherBank].count += 1;

          // Track outgoing vs incoming separately for more detailed analysis
          if (volumeChange < 0) {
            acc[otherBank].outgoingVolume += Math.abs(volumeChange);
            acc[otherBank].outgoingCount += 1;
          } else {
            acc[otherBank].incomingVolume += volumeChange;
            acc[otherBank].incomingCount += 1;
          }
        }

        return acc;
      }, {});

      // Calculate total statistics across all banks
      const totalStats = Object.values(bankStats).reduce(
        (totals, bankStat) => {
          return {
            totalCount: totals.totalCount + bankStat.count,
            totalNetVolume: totals.totalNetVolume + bankStat.netVolume,
            totalOutgoingVolume:
              totals.totalOutgoingVolume + bankStat.outgoingVolume,
            totalIncomingVolume:
              totals.totalIncomingVolume + bankStat.incomingVolume,
          };
        },
        {
          totalCount: 0,
          totalNetVolume: 0,
          totalOutgoingVolume: 0,
          totalIncomingVolume: 0,
        }
      );

      return res.status(200).json({
        success: true,
        data: {
          bankStats,
          totalStats,
          transactedBanks: Object.keys(bankStats), // List of banks AccessBankMSP has transacted with
          numberOfBanks: Object.keys(bankStats).length,
        },
        timestamp: new Date().toISOString(),
      });
    } catch (error) {
      console.error("Error getting bank transaction count:", error);
      return res.status(500).json({
        success: false,
        error: "Failed to fetch transaction statistics",
        timestamp: new Date().toISOString(),
      });
    }
  }
}
