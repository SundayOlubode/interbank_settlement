import * as crypto from "node:crypto";
import { databaseService } from "../services/databaseService.js";
import { acknowledgmentService } from "../services/paymentAcknowledgmentService.js";
import { config } from "../config/env.js";
import { userManager } from "../data/userManager.js";
import { fabricService } from "../services/fabricService.js";

export class PaymentController {
  constructor() {
    this.config = config;
    this.fabricService = fabricService;
    this.userManager = userManager;
    this.acknowledgmentService = acknowledgmentService;
    this.databaseService = databaseService;
  }

  async createPayment(req, res) {
    try {
      const contract = this.fabricService.getContract();
      const { payerAcct, payeeMSP, payeeAcct, amount } = req.body;

      // Validate user exists
      const user = await this.userManager.getUser(payerAcct);
      if (!user) {
        return res.status(400).json({
          error: "Invalid payer account",
          message: `Account ${payerAcct} does not exist`,
        });
      }

      // Validate payee bank
      if (![this.config.MSP_ID, payeeMSP].includes(payeeMSP)) {
        return res.status(400).json({
          error: "Invalid payee bank",
          message: `Payee MSP ${payeeMSP} is not supported`,
        });
      }

      // Check if recipient is not same account as payer
      if (payerAcct === payeeAcct) {
        return res.status(400).json({
          error: "Invalid transaction",
          message: `Payer account ${payerAcct} cannot be the same as payee account ${payeeAcct}`,
        });
      }

      // Check balance
      if (user.balance < amount) {
        return res.status(400).json({
          error: "Insufficient funds",
          message: `Account ${payerAcct} has insufficient funds (₦${user.balance}) for this transaction (₦${amount})`,
        });
      }

      // Check BVN
      if (!user.bvn) {
        return res.status(400).json({
          error: "Missing BVN",
          message: `Account ${payerAcct} does not have a valid BVN`,
        });
      }

      const paymentID = crypto.randomUUID().toString();

      // Create payment record in database
      const paymentRecord = await this.databaseService.createPayment({
        paymentId: paymentID,
        payerAcct,
        payerMSP: this.config.MSP_ID,
        payeeMSP,
        amount,
        status: "PENDING",
      });

      // Deduct amount from payer's account with transaction record
      const newBalance = user.balance - amount;
      await this.userManager.updateUserBalance(payerAcct, newBalance, {
        type: "DEBIT",
        description: `Payment to ${payeeAcct}`,
        paymentId: paymentID,
      });

      console.log(
        `Debited account ${payerAcct} (${user.firstname}) with ₦${amount}. New balance: ₦${newBalance}`
      );

      // Prepare payment data for blockchain
      const payJson = JSON.stringify({
        id: paymentID,
        payerMSP: this.config.MSP_ID,
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

      // Start waiting for acknowledgment
      const acknowledgmentPromise =
        this.acknowledgmentService.waitForPaymentAcknowledgment(
          paymentID,
          10000
        );

      console.log(
        `Submitting payment ${paymentID} and waiting for acknowledgment...`
      );

      try {
        // Submit the transaction to blockchain
        await contract.submit("CreatePayment", {
          transientData: {
            payment: Buffer.from(payJson),
          },
          endorsingOrganizations: [this.config.MSP_ID, payeeMSP],
        });

        console.log(
          `Payment ${paymentID} submitted to blockchain, waiting for acknowledgment...`
        );

        try {
          // Wait for the PaymentAcknowledged event
          const ackData = await acknowledgmentPromise;

          // Update payment status in database
          await this.databaseService.updatePaymentStatus(
            paymentID,
            "ACKNOWLEDGED"
          );

          console.log(`Payment ${paymentID} Acknowledged!`);

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
        await this.userManager.updateUserBalance(payerAcct, user.balance, {
          type: "CREDIT",
          description: `Refund for failed payment ${paymentID}`,
          paymentId: paymentID,
        });

        // Update payment status
        await this.databaseService.updatePaymentStatus(paymentID, "FAILED");

        console.log(
          `Refunded account ${payerAcct} with ₦${amount} due to transaction failure`
        );

        throw submitError;
      }
    } catch (err) {
      console.error("Payment creation error:", err);

      res.status(500).json({
        error: "Could not create payment",
        message: err.details ? err.details[0]["message"] : err.message,
      });
    }
  }

  async getAllBilateralPDCData(req, res) {
    const { collection } = req.params;

    try {
      const contract = this.fabricService.getContract();

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
  }

  async getAccountBalance(req, res) {
    const { accountId } = req.params;
    const user = await this.userManager.getUser(accountId);

    if (!user) {
      return res.status(404).json({
        error: "Account not found",
        message: `Account ${accountId} does not exist`,
      });
    }

    res.json({
      accountId,
      balance: user.balance,
      name: `${user.firstname} ${user.lastname}`,
      timestamp: new Date().toISOString(),
    });
  }

  async getTransactionHistory(req, res) {
    const { accountId } = req.params;
    const limit = parseInt(req.query.limit) || 20;

    try {
      const transactions = await this.userManager.getTransactionHistory(
        accountId,
        limit
      );
      res.json({
        accountId,
        transactions,
        total: transactions.length,
      });
    } catch (error) {
      res.status(500).json({
        error: "Failed to get transaction history",
        message: error.message,
      });
    }
  }

  async getPaymentHistory(req, res) {
    const { accountId } = req.params;
    const limit = parseInt(req.query.limit) || 10;

    try {
      const payments = await this.databaseService.getPaymentsByUser(
        accountId,
        limit
      );
      res.json({
        accountId,
        payments,
        total: payments.length,
      });
    } catch (error) {
      res.status(500).json({
        error: "Failed to get payment history",
        message: error.message,
      });
    }
  }

  async getHealth(req, res) {
    const dbHealth = await this.databaseService.healthCheck();

    res.json({
      status: dbHealth.status === "healthy" ? "healthy" : "degraded",
      msp: this.config.MSP_ID,
      database: dbHealth,
      timestamp: new Date().toISOString(),
      pendingAcknowledgments: this.acknowledgmentService.getPendingCount(),
    });
  }
}
