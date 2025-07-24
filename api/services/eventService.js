import { checkpointers } from "@hyperledger/fabric-gateway";
import { config } from "../config/env.js";
import { userAccounts } from "../data/userAccounts.js";
import { acknowledgmentService } from "./paymentAcknowledgmentService.js";
import { UserManager } from "../data/userManager.js";

export class EventService {
  constructor(fabricService) {
    this.fabricService = fabricService;
    this.userManager = new UserManager();
  }

  async processPaymentEvent(evt, contract, cp) {
    const { id, payerMSP, payeeMSP } = JSON.parse(
      Buffer.from(evt.payload).toString("utf8")
    );

    console.log(
      `Processing event ${evt.eventName} for payment payeeMSP ${payeeMSP}…`
    );

    if (payeeMSP !== config.MSP_ID) {
      await cp.checkpointChaincodeEvent(evt);
      return; // not my bank
    }

    try {
      const payBytes = await contract.submit("GetIncomingPayment", {
        arguments: [id],
      });
      const pay = JSON.parse(Buffer.from(payBytes).toString("utf8"));

      // Fetch user
      const user = await this.userManager.getUser(pay.payeeAcct);
      if (!user) {
        console.error(`User ${pay.payeeAcct} not found`);
        return;
      }

      await contract.submit("AcknowledgePayment", {
        arguments: [JSON.stringify({ id, payerMSP, payeeMSP, batchWindow: 0 })],
      });

      // Add amount to payee's account with transaction record
      const newBalance = user.balance + pay.amount;
      await this.userManager.updateUserBalance(pay.payeeAcct, newBalance, {
        type: "CREDIT",
        description: `Payment from ${pay.payerAcct}`,
        paymentId: id,
      });

      console.log(`Payment ${id} Acknowledged!`);
    } catch (error) {
      console.error(`Error processing payment ${id}:`, error);
    } finally {
      await cp.checkpointChaincodeEvent(evt);
    }
  }

  async processPaymentAcknowledgedEvent(evt, cp) {
    try {
      const ackData = JSON.parse(Buffer.from(evt.payload).toString("utf8"));

      if (ackData.payerMSP !== config.MSP_ID) {
        await cp.checkpointChaincodeEvent(evt);
        return;
      }

      const { id: paymentID } = ackData;

      console.log(
        `Received PaymentAcknowledged for payment ${paymentID}:`,
        ackData
      );

      acknowledgmentService.handleAcknowledgment(paymentID, ackData);

      await cp.checkpointChaincodeEvent(evt);
    } catch (error) {
      console.error("Error processing PaymentAcknowledged event:", error);
      await cp.checkpointChaincodeEvent(evt);
    }
  }

  async startListener() {
    const network = this.fabricService.getNetwork();
    const contract = this.fabricService.getContract();
    const cp = checkpointers.inMemory();

    while (true) {
      let stream;
      try {
        stream = await network.getChaincodeEvents(config.CHAINCODE, {
          checkpoint: cp,
        });

        for await (const evt of stream) {
          console.log(`Received event: ${evt.eventName}`);

          if (evt.eventName === "PaymentPending") {
            await this.processPaymentEvent(evt, contract, cp);
          } else if (evt.eventName === "PaymentAcknowledged") {
            await this.processPaymentAcknowledgedEvent(evt, cp);
          } else {
            await cp.checkpointChaincodeEvent(evt);
          }
        }
      } catch (err) {
        console.error("🔌 event stream dropped, reconnecting…", err);
      } finally {
        stream?.close?.();
      }
    }
  }
}
