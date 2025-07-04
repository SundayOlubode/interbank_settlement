import { databaseService } from "../services/databaseService.js";
import { PrismaClient } from "@prisma/client";

export class UserManager {
  constructor() {
    this.databaseService = databaseService;
    this.prisma = new PrismaClient();
  }

  async getUser(accountNumber) {
    try {
      return await this.databaseService.getUserByAccountNumber(accountNumber);
    } catch (error) {
      console.error(`Error getting user ${accountNumber}:`, error);
      return null;
    }
  }

  async getLoginUser(firstname, password) {
    try {
      const user = await this.prisma.user.findFirst({
        where: {
          firstname: firstname,
        },
      });
      return user;
    } catch (error) {
      console.error(`Error authenticating user ${username}:`, error);
      return null;
    }
  }

  async updateUserBalance(accountNumber, newBalance, transactionData = null) {
    try {
      return await this.databaseService.updateUserBalance(
        accountNumber,
        newBalance,
        transactionData
      );
    } catch (error) {
      console.error(`Error updating balance for ${accountNumber}:`, error);
      throw error;
    }
  }

  async getUsersByBank(bankMSP) {
    try {
      return await this.databaseService.getUsersByBankMSP(bankMSP);
    } catch (error) {
      console.error(`Error getting users for ${bankMSP}:`, error);
      return [];
    }
  }

  async getTransactionHistory(accountNumber, limit = 20) {
    try {
      return await this.databaseService.getTransactionHistory(
        accountNumber,
        limit
      );
    } catch (error) {
      console.error(
        `Error getting transaction history for ${accountNumber}:`,
        error
      );
      return [];
    }
  }

  async getTxsByUsername(accountNumber, limit = 20) {
    try {
      return await this.prisma.transaction.findMany({
        where: { accountNumber },
        orderBy: { timestamp: "desc" },
        take: limit,
      });
    } catch (error) {
      console.error(`Error fetching transactions for ${accountNumber}:`, error);
      return [];
    }
  }
}

export const userManager = new UserManager();
