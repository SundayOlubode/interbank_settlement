import { PrismaClient } from "@prisma/client";
import { readFile } from 'fs/promises';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const prisma = new PrismaClient();

// Import user data from JSON files
// Load user data files using readFile instead of import
const accessBankUsers = JSON.parse(
  await readFile("/Users/sam/Documents/Blockchain/interbank_settlement/api/accessbank-users.json"), "utf8");
const gtBankUsers = JSON.parse(
  await readFile("/Users/sam/Documents/Blockchain/interbank_settlement/api/gtbank-users.json"), "utf8");
const zenithBankUsers = JSON.parse(
  await readFile("/Users/sam/Documents/Blockchain/interbank_settlement/api/zenithbank-users.json"), "utf8");
const firstBankUsers = JSON.parse(
  await readFile("/Users/sam/Documents/Blockchain/interbank_settlement/api/firstbank-users.json"), "utf8");

async function main() {
  console.log("🌱 Starting database seed...");

  // Seed AccessBank users
  console.log("Seeding AccessBank users...");
  for (const user of accessBankUsers) {
    await prisma.user.upsert({
      where: { accountNumber: user.accountNumber },
      update: {},
      create: {
        accountNumber: user.accountNumber,
        firstname: user.firstname,
        lastname: user.lastname,
        middlename: user.middlename,
        bvn: user.bvn,
        gender: user.gender,
        balance: user.balance,
        birthdate: user.birthdate,
        bankMSP: "AccessBankMSP",
      },
    });
  }

  // Seed GTBank users
  //   console.log("Seeding GTBank users...");
  //   for (const user of gtBankUsers) {
  //     await prisma.user.upsert({
  //       where: { accountNumber: user.accountNumber },
  //       update: {},
  //       create: {
  //         accountNumber: user.accountNumber,
  //         firstname: user.firstname,
  //         lastname: user.lastname,
  //         middlename: user.middlename,
  //         bvn: user.bvn,
  //         gender: user.gender,
  //         balance: user.balance,
  //         birthdate: user.birthdate,
  //         bankMSP: "GTBankMSP",
  //       },
  //     });
  //   }

  //   // Seed ZenithBank users
  //   console.log("Seeding ZenithBank users...");
  //   for (const user of zenithBankUsers) {
  //     await prisma.user.upsert({
  //       where: { accountNumber: user.accountNumber },
  //       update: {},
  //       create: {
  //         accountNumber: user.accountNumber,
  //         firstname: user.firstname,
  //         lastname: user.lastname,
  //         middlename: user.middlename,
  //         bvn: user.bvn,
  //         gender: user.gender,
  //         balance: user.balance,
  //         birthdate: user.birthdate,
  //         bankMSP: "ZenithBankMSP",
  //       },
  //     });
  //   }

  //   // Seed FirstBank users
  //   console.log("Seeding FirstBank users...");
  //   for (const user of firstBankUsers) {
  //     await prisma.user.upsert({
  //       where: { accountNumber: user.accountNumber },
  //       update: {},
  //       create: {
  //         accountNumber: user.accountNumber,
  //         firstname: user.firstname,
  //         lastname: user.lastname,
  //         middlename: user.middlename,
  //         bvn: user.bvn,
  //         gender: user.gender,
  //         balance: user.balance,
  //         birthdate: user.birthdate,
  //         bankMSP: "FirstBankMSP",
  //       },
  //     });
  //   }

  console.log("✅ Database seeded successfully!");
}

main()
  .then(async () => {
    await prisma.$disconnect();
  })
  .catch(async (e) => {
    console.error(e);
    await prisma.$disconnect();
    process.exit(1);
  });
