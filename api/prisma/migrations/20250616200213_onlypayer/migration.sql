/*
  Warnings:

  - You are about to drop the column `payeeAcct` on the `payments` table. All the data in the column will be lost.

*/
-- DropForeignKey
ALTER TABLE "payments" DROP CONSTRAINT "payments_payeeAcct_fkey";

-- AlterTable
ALTER TABLE "payments" DROP COLUMN "payeeAcct";
