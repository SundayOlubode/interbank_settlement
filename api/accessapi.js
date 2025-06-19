import express from "express";
import morgan from "morgan";
import { config } from "./config/env.js";
import { fabricService } from "./services/fabricService.js";
import { EventService } from "./services/eventService.js";
import { PaymentController } from "./controllers/paymentController.js";
import { DashboardController } from "./controllers/dashboardController.js";
import { AuthController } from "./controllers/authController.js";
import { TxController } from "./controllers/txController.js";
import { createPaymentRoutes } from "./routes/paymentRoutes.js";
import { buildQsccHelpers } from "./helper/qcss.js";
import cors from "cors";

class BankApplication {
  constructor() {
    this.app = express();
    this.eventService = null;
    this.paymentController = null;
  }

  setupMiddleware() {
    this.app.use(express.json());
    this.app.use(morgan("dev"));
    this.app.use(cors({
      origin: ["http://localhost:5173", "*"],
      methods: "GET,POST,PUT,DELETE",
      credentials: true,
    }));
  }

  setupRoutes() {
    this.paymentController = new PaymentController(fabricService);
    this.authController = new AuthController();
    this.dashboardController = new DashboardController();
    this.txController = new TxController();
    const paymentRoutes = createPaymentRoutes(this.paymentController);

    // Dashboard route
    this.app.use("/api/balance", (req, res) =>
      this.dashboardController.getBankingOverview(req, res)
    );

    // Tx Analytics
    this.app.use("/api/transactions/analytics", (req, res) =>
      this.dashboardController.getTransactionAnalytics(req, res)
    );

    // Bank's transactions
    this.app.use("/api/transactions/history", (req, res) =>
      this.dashboardController.getAllBankTransaction(req, res)
    );

    // Bank's transactions
    this.app.use("/api/interbank/relations", (req, res) =>
      this.dashboardController.getAllBankTransactionCount(req, res)
    );

    // Auth routes
    this.app.post("/auth/login", (req, res) =>
      this.authController.login(req, res)
    );

    // Tx routes
    this.app.use("/tx/:username", (req, res) =>
      this.txController.getTransactionHistory(req, res)
    );

    // Payment routes
    this.app.use("/", paymentRoutes);
  }

  async initialize() {
    try {
      console.log(`Starting ${config.MSP_ID} API server...`);

      // Connect to Fabric
      await fabricService.connect();
      console.log("Gateway connection established");

      // Build QSCC helpers
      const qscc = await buildQsccHelpers(
        fabricService.getGateway(),
        config.CHANNEL
      );

      // Setup middleware and routes
      this.setupMiddleware();
      this.setupRoutes();

      // Start event listeners
      console.log("Setting up event listeners...");
      this.eventService = new EventService(fabricService);
      this.eventService.startListener().catch(console.error);

      // Start server
      this.app.listen(4000, () => {
        console.log(`${config.MSP_ID} API listening on port 4000`);
        console.log("Event listeners configured and running");
      });
    } catch (error) {
      console.error("Failed to start application:", error);
      process.exit(1);
    }
  }
}

const bankApp = new BankApplication();
bankApp.initialize();
