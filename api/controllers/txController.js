import { PrismaClient } from "@prisma/client";
import { userManager } from "../data/userManager.js";
import express from "express";

export class TxController {
  constructor() {
    this.userManager = userManager;

    this.router = express.Router();
    this.initializeRoutes();
  }

  initializeRoutes() {
    this.router.use("/", this.getTransactionHistory.bind(this));
  }

  async getTransactionHistory(req, res) {
    const { username } = req.params;
    const limit = parseInt(req.query.limit) || 20;

    try {
      const user = await this.userManager.getLoginUser(username);
      if (!user) {
        return res.status(404).json({
          error: "User not found",
          message: `User ${username} does not exist`,
        });
      }
      const transactions = await this.userManager.getTxsByUsername(
        user.accountNumber,
        limit
      );
      if (!transactions || transactions.length === 0) {
        return res.status(404).json({
          error: "No transactions found",
          message: `No transaction history for user ${username}`,
        });
      }

      res.status(200).json(transactions);
    } catch (error) {
      console.error(
        `Error fetching transaction history for ${username}:`,
        error
      );
      res.status(500).json({ error: "Internal server error" });
    }
  }
}
