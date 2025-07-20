import { databaseService } from "../services/databaseService.js";
import { fabricService } from "../services/fabricService.js";

export class DashboardController {
  constructor() {
    this.fabricService = fabricService;
    this.contract = this.fabricService.getContract();
    this.databaseService = databaseService;
  }

  async getBankingOverview(req, res) {
    const result = await this.contract.evaluateTransaction(
      "GetBankingOverview"
    );

    const bankingData = JSON.parse(Buffer.from(result).toString("utf8"));

    return res.status(200).json({
      success: true,
      data: bankingData,
      timestamp: new Date().toISOString(),
    });
  }

  async getTransactionAnalytics(req, res) {
    const result = await this.contract.evaluateTransaction(
      "GetAllTransactionAnalytics"
    );

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
      const result = await this.contract.evaluateTransaction(
        "GetTransactionHistory"
      );

      let transactions = JSON.parse(Buffer.from(result).toString("utf8"));

      if (bank) {
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
      const result = await this.contract.evaluateTransaction(
        "GetCounterpartyStats"
      );
      const relations = JSON.parse(Buffer.from(result).toString("utf8"));

      return res.status(200).json({
        success: true,
        data: relations,
        timestamp: new Date().toISOString(),
      });
    } catch (error) {
      console.error("Error retrieving bank relations count:", error);
      return res.status(500).json({
        success: false,
        message: "Failed to retrieve bank relations count",
        error: error.message,
      });
    }
  }
}
