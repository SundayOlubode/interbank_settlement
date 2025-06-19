import express from "express";

export function createPaymentRoutes(paymentController) {
  const router = express.Router();

  // Payment routes
  router.post("/payments", (req, res) => paymentController.createPayment(req, res));
  
  // Account routes
  router.get("/accounts/:accountId/balance", (req, res) => paymentController.getAccountBalance(req, res));
  router.get("/accounts/:accountId/transactions", (req, res) => paymentController.getTransactionHistory(req, res));
  router.get("/accounts/:accountId/payments", (req, res) => paymentController.getPaymentHistory(req, res));
  
  // Health check
  router.get("/health", (req, res) => paymentController.getHealth(req, res));

  return router;
}