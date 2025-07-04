// settlement.go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// GetSettlementAccount retrieves the settlement account for a specific MSP
func (s *SmartContract) GetSettlementAccount(ctx contractapi.TransactionContextInterface, msp string) (*BankAccount, error) {
	collectionName := fmt.Sprintf("col-settlement-%s", msp)

	accountBytes, err := ctx.GetStub().GetPrivateData(collectionName, msp)
	if err != nil {
		return nil, fmt.Errorf("failed to get settlement account for %s: %v", msp, err)
	}
	if accountBytes == nil {
		return nil, fmt.Errorf("settlement account for %s not found", msp)
	}

	var account BankAccount
	if err := json.Unmarshal(accountBytes, &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account data: %v", err)
	}

	return &account, nil
}

// DebitAccount debits amount from specified MSP settlement account
// Returns "SUCCESS" if debit successful, "QUEUED" if insufficient funds
func (s *SmartContract) DebitAccount(ctx contractapi.TransactionContextInterface, paymentDetails PaymentEventDetails) (string, error) {
	payerMSP := paymentDetails.PayerMSP
	payeeMSP := paymentDetails.PayeeMSP
	paymentID := paymentDetails.ID

	// Look up the payment in Bilateral PDC (this works as CentralBank can access bilateral PDCs)
	paymentColl := getCollectionName(paymentDetails.PayerMSP, paymentDetails.PayeeMSP)
	paymentBytes, err := ctx.GetStub().GetPrivateData(paymentColl, paymentID)
	if err != nil || paymentBytes == nil {
		return "", fmt.Errorf("payment %s not found in collection %s", paymentID, paymentColl)
	}

	var paymentInPDC PaymentDetails
	if err := json.Unmarshal(paymentBytes, &paymentInPDC); err != nil {
		return "", fmt.Errorf("failed to unmarshal payment details: %v", err)
	}

	if paymentInPDC.Status != "ACKNOWLEDGED" {
		return "", fmt.Errorf("payment %s is not in ACKNOWLEDGED state", paymentID)
	}

	amount := paymentInPDC.Amount

	// Only access the payer's settlement account
	collectionName := fmt.Sprintf("col-settlement-%s", payerMSP)

	accountBytes, err := ctx.GetStub().GetPrivateData(collectionName, payerMSP)
	if err != nil {
		return "", fmt.Errorf("failed to read account for %s: %v", payerMSP, err)
	}

	if len(accountBytes) == 0 {
		return "", fmt.Errorf("no account found for %s in %s", payerMSP, collectionName)
	}

	var account BankAccount
	if err := json.Unmarshal(accountBytes, &account); err != nil {
		return "", fmt.Errorf("failed to unmarshal account in debit account for %s in %s: %v", err, payerMSP, collectionName)
	}

	// Check if sufficient funds available
	if account.Balance < amount {
		// Update payment status to QUEUED in bilateral PDC
		if err := s.updatePaymentStatusInPDC(ctx, payerMSP, payeeMSP, paymentID, "QUEUED"); err != nil {
			return "", fmt.Errorf("failed to update payment status to QUEUED: %v", err)
		}

		// Update public stub status to QUEUED
		if err := s.updatePublicPaymentStatus(ctx, paymentID, "QUEUED"); err != nil {
			return "", fmt.Errorf("failed to update public payment status to QUEUED: %v", err)
		}

		// Emit queued event
		evt := map[string]interface{}{
			"type":             "PaymentQueued",
			"payerMSP":         payerMSP,
			"payeeMSP":         payeeMSP,
			"paymentID":        paymentID,
			"amount":           amount,
			"availableBalance": account.Balance,
			"reason":           "insufficient_funds",
		}
		evtBytes, _ := json.Marshal(evt)
		ctx.GetStub().SetEvent("PaymentQueued", evtBytes)

		return "QUEUED", nil
	}

	// Sufficient funds available - proceed with debit
	account.Balance -= amount
	updatedBytes, _ := json.Marshal(account)
	if err := ctx.GetStub().PutPrivateData(collectionName, payerMSP, updatedBytes); err != nil {
		return "", fmt.Errorf("failed to update account: %v", err)
	}

	// Update payment status to DEBITED in bilateral PDC
	if err := s.updatePaymentStatusInPDC(ctx, payerMSP, payeeMSP, paymentID, "DEBITED"); err != nil {
		return "", fmt.Errorf("failed to update payment status to DEBITED: %v", err)
	}

	return "SUCCESS", nil
}

// CreditAccount credits amount to specified MSP settlement account
func (s *SmartContract) CreditAccount(ctx contractapi.TransactionContextInterface, paymentDetails PaymentEventDetails) error {
	payerMSP := paymentDetails.PayerMSP
	payeeMSP := paymentDetails.PayeeMSP
	paymentID := paymentDetails.ID

	// Look up the payment in Bilateral PDC
	paymentColl := getCollectionName(paymentDetails.PayerMSP, paymentDetails.PayeeMSP)
	paymentBytes, err := ctx.GetStub().GetPrivateData(paymentColl, paymentID)
	if err != nil || paymentBytes == nil {
		return fmt.Errorf("payment %s not found in collection %s", paymentID, paymentColl)
	}

	var paymentInPDC PaymentDetails
	if err := json.Unmarshal(paymentBytes, &paymentInPDC); err != nil {
		return fmt.Errorf("failed to unmarshal payment details: %v", err)
	}

	if paymentInPDC.Status != "DEBITED" {
		return fmt.Errorf("payment %s is not yet in DEBITED state", paymentID)
	}

	amount := paymentInPDC.Amount

	// Only access the payee's settlement account
	collectionName := fmt.Sprintf("col-settlement-%s", payeeMSP)

	accountBytes, err := ctx.GetStub().GetPrivateData(collectionName, payeeMSP)
	if err != nil {
		return fmt.Errorf("failed to read account for %s: %v", payeeMSP, err)
	}

	var account BankAccount
	if err := json.Unmarshal(accountBytes, &account); err != nil {
		return fmt.Errorf("failed to unmarshal account in credit account: %v", err)
	}

	account.Balance += amount
	updatedBytes, _ := json.Marshal(account)
	if err := ctx.GetStub().PutPrivateData(collectionName, payeeMSP, updatedBytes); err != nil {
		return fmt.Errorf("failed to update account: %v", err)
	}

	// TODO: Update AmountToSettle to 0 in PDC
	// Update payment status to SETTLED in bilateral PDC
	if err := s.updatePaymentStatusInPDC(ctx, payerMSP, payeeMSP, paymentID, "SETTLED"); err != nil {
		return fmt.Errorf("failed to update payment status to SETTLED: %v", err)
	}

	// Update public stub status to SETTLED
	if err := s.updatePublicPaymentStatus(ctx, paymentID, "SETTLED"); err != nil {
		return fmt.Errorf("failed to update public payment status to SETTLED: %v", err)
	}

	// Emit final settlement event
	settlementEvt := PaymentEventDetails{
		ID:       paymentID,
		PayeeMSP: payeeMSP,
		PayerMSP: payerMSP,
	}
	return s.emitPaymentEvent(ctx, "PaymentSettled", settlementEvt)
}

// Helper function to update payment status in bilateral PDC
func (s *SmartContract) updatePaymentStatusInPDC(ctx contractapi.TransactionContextInterface, payerMSP, payeeMSP, paymentID, status string) error {
	paymentColl := getCollectionName(payerMSP, payeeMSP)
	paymentBytes, err := ctx.GetStub().GetPrivateData(paymentColl, paymentID)
	if err != nil || paymentBytes == nil {
		return fmt.Errorf("payment %s not found in collection %s", paymentID, paymentColl)
	}

	var paymentDetails PaymentDetails
	if err := json.Unmarshal(paymentBytes, &paymentDetails); err != nil {
		return fmt.Errorf("failed to unmarshal payment details: %v", err)
	}

	paymentDetails.Status = status
	updatedPaymentBytes, _ := json.Marshal(paymentDetails)
	return ctx.GetStub().PutPrivateData(paymentColl, paymentID, updatedPaymentBytes)
}

// Helper function to update payment status in public state
func (s *SmartContract) updatePublicPaymentStatus(ctx contractapi.TransactionContextInterface, paymentID, status string) error {
	stubBytes, err := ctx.GetStub().GetState(paymentID)
	if err != nil || stubBytes == nil {
		return fmt.Errorf("payment stub %s not found", paymentID)
	}

	var stub PaymentStub
	if err := json.Unmarshal(stubBytes, &stub); err != nil {
		return fmt.Errorf("failed to unmarshal payment stub: %v", err)
	}

	stub.Status = status
	updatedStubBytes, _ := json.Marshal(stub)
	return ctx.GetStub().PutState(paymentID, updatedStubBytes)
}
