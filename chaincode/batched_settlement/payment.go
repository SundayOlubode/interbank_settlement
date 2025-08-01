package settlement

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// CreatePayment records a new payment in PENDING status (banks create payments)
func (s *SmartContract) CreatePayment(ctx contractapi.TransactionContextInterface) error {
	transMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("error getting transient data: %v", err)
	}
	paymentJSON, ok := transMap["payment"]
	if !ok {
		return fmt.Errorf("payment details must be provided in transient data under 'payment'")
	}

	var details PaymentDetails
	if err := json.Unmarshal(paymentJSON, &details); err != nil {
		return fmt.Errorf("failed to unmarshal payment details: %v", err)
	}

	// Validate caller is an authorized bank (not CBN)
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client MSP: %v", err)
	}

	// Validate payer MSP matches caller
	if details.PayerMSP != clientMSP {
		return fmt.Errorf("payer MSP must match calling MSP")
	}

	// Set mandatory fields
	details.AmountToSettle = details.Amount
	details.Status = "PENDING"
	details.BatchWindow = getCurrentBatchWindow()

	// Verify BVN
	if err := s.verifyBVN(ctx, details.User); err != nil {
		return err
	}

	updatedBytes, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal updated PaymentDetails: %v", err)
	}

	// Store full details in bilateral collection
	coll := getCollectionName(details.PayerMSP, details.PayeeMSP)
	if err := ctx.GetStub().PutPrivateData(coll, details.ID, updatedBytes); err != nil {
		return fmt.Errorf("failed to put private payment data: %v", err)
	}

	// Create and store public stub
	hash := computeHash(createHashablePayment(details))
	stub := PaymentStub{
		ID:          details.ID,
		Hash:        hash,
		PayerMSP:    details.PayerMSP,
		PayeeMSP:    details.PayeeMSP,
		Status:      details.Status,
		Timestamp:   details.Timestamp,
		BatchWindow: details.BatchWindow,
	}

	stubJSON, err := json.Marshal(stub)
	if err != nil {
		return fmt.Errorf("failed to marshal payment stub: %v", err)
	}
	if err := ctx.GetStub().PutState(details.ID, stubJSON); err != nil {
		return fmt.Errorf("failed to put payment stub: %v", err)
	}

	// Emit event
	return s.emitPaymentEvent(ctx, "PaymentPending", PaymentEventDetails{
		ID:          details.ID,
		PayeeMSP:    details.PayeeMSP,
		PayerMSP:    details.PayerMSP,
		BatchWindow: details.BatchWindow,
	})
}

// AcknowledgePayment moves payment to ACKNOWLEDGED status (banks acknowledge payments)
func (s *SmartContract) AcknowledgePayment(ctx contractapi.TransactionContextInterface, paymentDetails PaymentEventDetails) error {
	// Validate caller is an authorized bank (not CBN)
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client MSP: %v", err)
	}

	// Validate caller is the payee bank
	if paymentDetails.PayeeMSP != clientMSP {
		return fmt.Errorf("only payee bank can acknowledge payment")
	}

	// Update payment status to ACKNOWLEDGED
	err = s.updatePaymentStatusInPDC(ctx, paymentDetails.PayerMSP, paymentDetails.PayeeMSP, paymentDetails.ID, "ACKNOWLEDGED")
	if err != nil {
		return fmt.Errorf("failed to update payment status: %v", err)
	}

	// Update public stub status
	err = s.updatePublicPaymentStatus(ctx, paymentDetails.ID, "ACKNOWLEDGED")
	if err != nil {
		return fmt.Errorf("failed to update public payment status: %v", err)
	}

	// Emit event for CBN to pick up and batch
	return s.emitPaymentEvent(ctx, "PaymentAcknowledged", PaymentEventDetails{
		ID:       paymentDetails.ID,
		PayeeMSP: paymentDetails.PayeeMSP,
		PayerMSP: paymentDetails.PayerMSP,
	})
}

// AcknowledgePaymentSimple - simplified version for easier bank API integration
func (s *SmartContract) AcknowledgePaymentSimple(ctx contractapi.TransactionContextInterface, id, payerMSP, payeeMSP string) error {
	// Validate caller is an authorized bank (not CBN)
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client MSP: %v", err)
	}

	// Validate caller is the payee bank
	if payeeMSP != clientMSP {
		return fmt.Errorf("only payee bank can acknowledge payment")
	}

	// Update payment status to ACKNOWLEDGED
	err = s.updatePaymentStatusInPDC(ctx, payerMSP, payeeMSP, id, "ACKNOWLEDGED")
	if err != nil {
		return fmt.Errorf("failed to update payment status: %v", err)
	}

	// Update public stub status
	err = s.updatePublicPaymentStatus(ctx, id, "ACKNOWLEDGED")
	if err != nil {
		return fmt.Errorf("failed to update public payment status: %v", err)
	}

	// Emit event for CBN to pick up and batch
	return s.emitPaymentEvent(ctx, "PaymentAcknowledged", PaymentEventDetails{
		ID:       id,
		PayeeMSP: payeeMSP,
		PayerMSP: payerMSP,
	})
}

// BatchAcknowledgedPayment - CBN ONLY function to batch acknowledged payments
func (s *SmartContract) BatchAcknowledgedPayment(ctx contractapi.TransactionContextInterface, paymentDetails PaymentEventDetails) error {
	// STRICT CBN-ONLY AUTHORIZATION
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client MSP: %v", err)
	}

	if clientMSP != "CentralBankMSP" {
		return fmt.Errorf("only Central Bank can batch payments")
	}

	// Verify payment exists and is in ACKNOWLEDGED status
	payment, err := s.getPaymentDetails(ctx, paymentDetails.PayerMSP, paymentDetails.PayeeMSP, paymentDetails.ID)
	if err != nil {
		return fmt.Errorf("failed to get payment details: %v", err)
	}

	if payment.Status != "ACKNOWLEDGED" {
		return fmt.Errorf("payment %s is not in ACKNOWLEDGED status, current status: %s", paymentDetails.ID, payment.Status)
	}

	// Update payment status to BATCHED
	err = s.updatePaymentStatusInPDC(ctx, paymentDetails.PayerMSP, paymentDetails.PayeeMSP, paymentDetails.ID, "BATCHED")
	if err != nil {
		return fmt.Errorf("failed to update payment status to BATCHED: %v", err)
	}

	// Update public stub status
	err = s.updatePublicPaymentStatus(ctx, paymentDetails.ID, "BATCHED")
	if err != nil {
		return fmt.Errorf("failed to update public payment status to BATCHED: %v", err)
	}

	// Emit batching event
	return s.emitPaymentEvent(ctx, "PaymentBatched", PaymentEventDetails{
		ID:       paymentDetails.ID,
		PayeeMSP: paymentDetails.PayeeMSP,
		PayerMSP: paymentDetails.PayerMSP,
	})
}

// BatchAcknowledgedPaymentSimple - CBN ONLY function with simple parameters
func (s *SmartContract) BatchAcknowledgedPaymentSimple(ctx contractapi.TransactionContextInterface, id, payerMSP, payeeMSP string) error {
	// Verify payment exists and is in ACKNOWLEDGED status
	payment, err := s.getPaymentDetails(ctx, payerMSP, payeeMSP, id)
	if err != nil {
		return fmt.Errorf("failed to get payment details: %v", err)
	}

	if payment.Status != "ACKNOWLEDGED" {
		return fmt.Errorf("payment %s is not in ACKNOWLEDGED status, current status: %s", id, payment.Status)
	}

	// Update payment status to BATCHED
	err = s.updatePaymentStatusInPDC(ctx, payerMSP, payeeMSP, id, "BATCHED")
	if err != nil {
		return fmt.Errorf("failed to update payment status to BATCHED: %v", err)
	}

	// Update public stub status
	err = s.updatePublicPaymentStatus(ctx, id, "BATCHED")
	if err != nil {
		return fmt.Errorf("failed to update public payment status to BATCHED: %v", err)
	}

	// Emit batching event
	return s.emitPaymentEvent(ctx, "PaymentBatched", PaymentEventDetails{
		ID:       id,
		PayeeMSP: payeeMSP,
		PayerMSP: payerMSP,
	})
}

// GetIncomingPayment retrieves the private payment details (banks query incoming payments)
func (s *SmartContract) GetIncomingPayment(ctx contractapi.TransactionContextInterface, id string) (*PaymentDetails, error) {
	// Validate caller is an authorized bank or CBN
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("failed to get client MSP: %v", err)
	}

	if !s.isAuthorizedMSP(clientMSP) {
		return nil, fmt.Errorf("unauthorized MSP: %s", clientMSP)
	}

	// Lookup stub to derive collection
	stubBytes, err := ctx.GetStub().GetState(id)
	if err != nil || stubBytes == nil {
		return nil, fmt.Errorf("payment stub %s not found", id)
	}
	var stub PaymentStub
	if err := json.Unmarshal(stubBytes, &stub); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stub: %v", err)
	}

	// Validate caller has access to this payment (payer, payee, or CBN)
	if clientMSP != "CentralBankMSP" && clientMSP != stub.PayerMSP && clientMSP != stub.PayeeMSP {
		return nil, fmt.Errorf("unauthorized access to payment %s", id)
	}

	coll := getCollectionName(stub.PayerMSP, stub.PayeeMSP)
	privBytes, err := ctx.GetStub().GetPrivateData(coll, id)
	if err != nil || privBytes == nil {
		return nil, fmt.Errorf("private data for payment %s not found", id)
	}
	var details PaymentDetails
	if err := json.Unmarshal(privBytes, &details); err != nil {
		return nil, fmt.Errorf("failed to unmarshal private payment: %v", err)
	}

	return &details, nil
}

// Helper function to get payment details (internal use)
func (s *SmartContract) getPaymentDetails(ctx contractapi.TransactionContextInterface, payerMSP, payeeMSP, id string) (*PaymentDetails, error) {
	coll := getCollectionName(payerMSP, payeeMSP)
	privBytes, err := ctx.GetStub().GetPrivateData(coll, id)
	if err != nil || privBytes == nil {
		return nil, fmt.Errorf("private data for payment %s not found", id)
	}
	var details PaymentDetails
	if err := json.Unmarshal(privBytes, &details); err != nil {
		return nil, fmt.Errorf("failed to unmarshal private payment: %v", err)
	}
	return &details, nil
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

// GetCurrentBatchWindow returns the current batch window identifier
func getCurrentBatchWindow() int64 {
	now := time.Now()
	// Create 2-minute windows: divide by 120 seconds and round down
	return now.Unix() / 120
}

// GetBatchWindowStart returns the start time of a batch window
func getBatchWindowStart(batchWindow int64) time.Time {
	return time.Unix(batchWindow*120, 0)
}

// GetBatchWindowEnd returns the end time of a batch window
func getBatchWindowEnd(batchWindow int64) time.Time {
	return time.Unix((batchWindow+1)*120, 0)
}

// REMOVED FUNCTIONS (No longer needed in CBN-controlled flow):
// - GetBatchedPayments() - CBN handles batching internally
// - ProcessPaymentBatch() - CBN handles batching
// - ProcessSinglePaymentInBatch() - Not needed
// - GetBatchedPaymentsForSettlement() - CBN handles settlement

// GetBilateralPayments retrieves all payment records from a bilateral PDC (kept for compatibility)
func (s *SmartContract) GetBilateralPayments(ctx contractapi.TransactionContextInterface, msp1, msp2 string) ([]*PaymentDetails, error) {
	// Validate caller has access
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("failed to get client MSP: %v", err)
	}

	if clientMSP != "CentralBankMSP" && clientMSP != msp1 && clientMSP != msp2 {
		return nil, fmt.Errorf("unauthorized access to bilateral payments")
	}

	collectionName := getCollectionName(msp1, msp2)

	resultsIterator, err := ctx.GetStub().GetPrivateDataByRange(collectionName, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get private data by range for collection %s: %v", collectionName, err)
	}
	defer resultsIterator.Close()

	var payments []*PaymentDetails
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate over results: %v", err)
		}

		var payment PaymentDetails
		if err := json.Unmarshal(queryResponse.Value, &payment); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payment data for key %s: %v", queryResponse.Key, err)
		}

		payments = append(payments, &payment)
	}

	return payments, nil
}

// GetBilateralPaymentsByStatus retrieves payment records filtered by status (kept for compatibility)
func (s *SmartContract) GetBilateralPaymentsByStatus(ctx contractapi.TransactionContextInterface, msp1, msp2, status string) ([]*PaymentDetails, error) {
	allPayments, err := s.GetBilateralPayments(ctx, msp1, msp2)
	if err != nil {
		return nil, err
	}

	var filteredPayments []*PaymentDetails
	for _, payment := range allPayments {
		if payment.Status == status {
			filteredPayments = append(filteredPayments, payment)
		}
	}

	return filteredPayments, nil
}

// GetAllPrivateData returns all private data in a given collection (CBN only)
func (s *SmartContract) GetAllPrivateData(ctx contractapi.TransactionContextInterface, collection string) (string, error) {
	// CBN-only function
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get client MSP: %v", err)
	}

	if clientMSP != "CentralBankMSP" {
		return "", fmt.Errorf("only Central Bank can access all private data")
	}

	// Simple range query with empty start/end keys
	resultsIterator, err := ctx.GetStub().GetPrivateDataByRange(collection, "", "")
	if err != nil {
		return "", err
	}
	defer resultsIterator.Close()

	var results []string

	for resultsIterator.HasNext() {
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return "", err
		}

		record := fmt.Sprintf(`{"key":"%s","value":%s}`, queryResult.Key, string(queryResult.Value))
		results = append(results, record)
	}

	// Return JSON array
	return fmt.Sprintf("[%s]", strings.Join(results, ",")), nil
}
