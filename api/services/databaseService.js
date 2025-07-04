import { PrismaClient } from "@prisma/client";

class DatabaseService {
  constructor() {
    this.prisma = new PrismaClient();
  }

  async connect() {
    try {
      await this.prisma.$connect();
      console.log("✅ Database connected successfully");
    } catch (error) {
      console.error("❌ Database connection failed:", error);
      throw error;
    }
  }

  async disconnect() {
    await this.prisma.$disconnect();
  }

  // User operations
  async getUserByAccountNumber(accountNumber) {
    return await this.prisma.user.findUnique({
      where: { accountNumber },
    });
  }

  async updateUserBalance(accountNumber, newBalance, transaction = null) {
    const user = await this.prisma.user.findUnique({
      where: { accountNumber },
    });

    if (!user) {
      throw new Error(`User with account ${accountNumber} not found`);
    }

    const balanceBefore = user.balance;

    // Update user balance and create transaction record
    const result = await this.prisma.$transaction(async (tx) => {
      // Update user balance
      const updatedUser = await tx.user.update({
        where: { accountNumber },
        data: { balance: newBalance },
      });

      // Create transaction record
      if (transaction) {
        await tx.transaction.create({
          data: {
            accountNumber,
            type: transaction.type,
            amount: Math.abs(newBalance - balanceBefore),
            balanceBefore,
            balanceAfter: newBalance,
            description: transaction.description,
            paymentId: transaction.paymentId,
          },
        });
      }

      return updatedUser;
    });

    return result;
  }

  async getUsersByBankMSP(bankMSP) {
    return await this.prisma.user.findMany({
      where: { bankMSP },
    });
  }

  // Payment operations
  async createPayment(paymentData) {
    return await this.prisma.payment.create({
      data: paymentData,
    //   include: {
    //     payer: true,
    //     payee: true,
    //   },
    });
  }

  async updatePaymentStatus(paymentId, status, additionalData = {}) {
    return await this.prisma.payment.update({
      where: { paymentId },
      data: {
        status,
        ...additionalData,
      },
    });
  }

  async getPaymentByPaymentId(paymentId) {
    return await this.prisma.payment.findUnique({
      where: { paymentId },
      include: {
        payer: true,
        payee: true,
      },
    });
  }

  async getPaymentsByUser(accountNumber, limit = 10) {
    return await this.prisma.payment.findMany({
      where: {
        OR: [{ payerAcct: accountNumber }, { payeeAcct: accountNumber }],
      },
      include: {
        payer: true,
        payee: true,
      },
      orderBy: { timestamp: "desc" },
      take: limit,
    });
  }

  // Settlement operations
  async createSettlement(settlementData) {
    return await this.prisma.settlement.create({
      data: settlementData,
    });
  }

  async updateSettlementStatus(paymentId, status, additionalData = {}) {
    return await this.prisma.settlement.update({
      where: { paymentId },
      data: {
        status,
        ...additionalData,
      },
    });
  }

  async getSettlementByPaymentId(paymentId) {
    return await this.prisma.settlement.findUnique({
      where: { paymentId },
    });
  }

  async getQueuedSettlements() {
    return await this.prisma.settlement.findMany({
      where: { status: "QUEUED" },
      orderBy: { timestamp: "asc" },
    });
  }

  // Transaction operations
  async getTransactionHistory(accountNumber, limit = 20) {
    return await this.prisma.transaction.findMany({
      where: { accountNumber },
      orderBy: { timestamp: "desc" },
      take: limit,
    });
  }

  // Get all bank's transactions
  async getAllBankTransactions(bankMSP = "", limit = 100) {
    return await this.prisma.payment.findMany({
      orderBy: { timestamp: "desc" },
      take: limit,
    });
  }

  // Analytics operations
  async getPaymentStats(bankMSP, startDate, endDate) {
    const payments = await this.prisma.payment.findMany({
      where: {
        AND: [
          { OR: [{ payerMSP: bankMSP }, { payeeMSP: bankMSP }] },
          { timestamp: { gte: startDate } },
          { timestamp: { lte: endDate } },
        ],
      },
    });

    return {
      totalPayments: payments.length,
      totalAmount: payments.reduce((sum, p) => sum + p.amount, 0),
      byStatus: payments.reduce((acc, p) => {
        acc[p.status] = (acc[p.status] || 0) + 1;
        return acc;
      }, {}),
    };
  }

  // Health check
  async healthCheck() {
    try {
      await this.prisma.$queryRaw`SELECT 1`;
      return { status: "healthy" };
    } catch (error) {
      return { status: "unhealthy", error: error.message };
    }
  }
}

export const databaseService = new DatabaseService();
