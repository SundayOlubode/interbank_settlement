class PaymentAcknowledgmentService {
  constructor() {
    this.pendingAcknowledgments = new Map();
  }

  waitForPaymentAcknowledgment(paymentID, timeout = 10000) {
    return new Promise((resolve, reject) => {
      const timeoutId = setTimeout(() => {
        this.pendingAcknowledgments.delete(paymentID);
        reject(new Error('Payment acknowledgment timeout'));
      }, timeout);

      this.pendingAcknowledgments.set(paymentID, {
        resolve: (ackData) => {
          clearTimeout(timeoutId);
          this.pendingAcknowledgments.delete(paymentID);
          resolve(ackData);
        },
        reject: (error) => {
          clearTimeout(timeoutId);
          this.pendingAcknowledgments.delete(paymentID);
          reject(error);
        }
      });
    });
  }

  handleAcknowledgment(paymentID, ackData) {
    const pending = this.pendingAcknowledgments.get(paymentID);
    if (pending) {
      pending.resolve(ackData);
    }
  }

  getPendingCount() {
    return this.pendingAcknowledgments.size;
  }
}

export const acknowledgmentService = new PaymentAcknowledgmentService();
