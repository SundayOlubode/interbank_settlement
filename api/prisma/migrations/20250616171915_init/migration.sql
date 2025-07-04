-- CreateEnum
CREATE TYPE "PaymentStatus" AS ENUM ('PENDING', 'ACKNOWLEDGED', 'SETTLED', 'QUEUED', 'FAILED');

-- CreateEnum
CREATE TYPE "TransactionType" AS ENUM ('DEBIT', 'CREDIT', 'SETTLEMENT_DEBIT', 'SETTLEMENT_CREDIT');

-- CreateEnum
CREATE TYPE "SettlementStatus" AS ENUM ('PENDING', 'DEBITED', 'CREDITED', 'SETTLED', 'QUEUED', 'FAILED');

-- CreateTable
CREATE TABLE "users" (
    "id" TEXT NOT NULL,
    "accountNumber" TEXT NOT NULL,
    "firstname" TEXT NOT NULL,
    "lastname" TEXT NOT NULL,
    "middlename" TEXT,
    "bvn" TEXT NOT NULL,
    "gender" TEXT NOT NULL,
    "balance" DOUBLE PRECISION NOT NULL DEFAULT 0,
    "birthdate" TEXT NOT NULL,
    "phone" TEXT,
    "bankMSP" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "payments" (
    "id" TEXT NOT NULL,
    "paymentId" TEXT NOT NULL,
    "payerAcct" TEXT NOT NULL,
    "payeeAcct" TEXT NOT NULL,
    "payerMSP" TEXT NOT NULL,
    "payeeMSP" TEXT NOT NULL,
    "amount" DOUBLE PRECISION NOT NULL,
    "currency" TEXT NOT NULL DEFAULT 'eNaira',
    "status" "PaymentStatus" NOT NULL DEFAULT 'PENDING',
    "blockchainTx" TEXT,
    "hash" TEXT,
    "timestamp" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "settledAt" TIMESTAMP(3),

    CONSTRAINT "payments_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "transactions" (
    "id" TEXT NOT NULL,
    "accountNumber" TEXT NOT NULL,
    "type" "TransactionType" NOT NULL,
    "amount" DOUBLE PRECISION NOT NULL,
    "balanceBefore" DOUBLE PRECISION NOT NULL,
    "balanceAfter" DOUBLE PRECISION NOT NULL,
    "description" TEXT NOT NULL,
    "paymentId" TEXT,
    "timestamp" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "transactions_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "settlements" (
    "id" TEXT NOT NULL,
    "paymentId" TEXT NOT NULL,
    "payerMSP" TEXT NOT NULL,
    "payeeMSP" TEXT NOT NULL,
    "amount" DOUBLE PRECISION NOT NULL,
    "status" "SettlementStatus" NOT NULL DEFAULT 'PENDING',
    "debitedAt" TIMESTAMP(3),
    "creditedAt" TIMESTAMP(3),
    "settledAt" TIMESTAMP(3),
    "queuedReason" TEXT,
    "timestamp" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "settlements_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "users_accountNumber_key" ON "users"("accountNumber");

-- CreateIndex
CREATE UNIQUE INDEX "users_bvn_key" ON "users"("bvn");

-- CreateIndex
CREATE UNIQUE INDEX "payments_paymentId_key" ON "payments"("paymentId");

-- CreateIndex
CREATE UNIQUE INDEX "settlements_paymentId_key" ON "settlements"("paymentId");

-- AddForeignKey
ALTER TABLE "payments" ADD CONSTRAINT "payments_payerAcct_fkey" FOREIGN KEY ("payerAcct") REFERENCES "users"("accountNumber") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payments" ADD CONSTRAINT "payments_payeeAcct_fkey" FOREIGN KEY ("payeeAcct") REFERENCES "users"("accountNumber") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transactions" ADD CONSTRAINT "transactions_accountNumber_fkey" FOREIGN KEY ("accountNumber") REFERENCES "users"("accountNumber") ON DELETE RESTRICT ON UPDATE CASCADE;
