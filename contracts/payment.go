package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// CreatePayment records a new payment, verifies BVN, transfers tokens, and writes a public stub
func (s *SmartContract) CreatePayment(ctx contractapi.TransactionContextInterface) error {
	// Get transient payment details
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

	// Verify BVN
	if err := s.verifyBVN(ctx, details.User); err != nil {
		return err
	}

	// Store full details in bilateral collection
	coll := getCollectionName(details.PayerMSP, details.PayeeMSP)
	if err := ctx.GetStub().PutPrivateData(coll, details.ID, paymentJSON); err != nil {
		return fmt.Errorf("failed to put private payment data: %v", err)
	}

	// Create and store public stub
	hash := computeHash(createHashablePayment(details))
	stub := PaymentStub{
		ID:        details.ID,
		Hash:      hash,
		PayerMSP:  details.PayerMSP,
		PayeeMSP:  details.PayeeMSP,
		Status:    "PENDING",
		Timestamp: details.Timestamp,
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
		ID:       details.ID,
		PayeeMSP: details.PayeeMSP,
		PayerMSP: details.PayerMSP,
	})
}

// GetPrivatePayment retrieves the private payment details given a stub ID
func (s *SmartContract) GetIncomingPayment(ctx contractapi.TransactionContextInterface, id string) (*PaymentDetails, error) {
	// Lookup stub to derive collection
	stubBytes, err := ctx.GetStub().GetState(id)
	if err != nil || stubBytes == nil {
		return nil, fmt.Errorf("payment stub %s not found", id)
	}
	var stub PaymentStub
	if err := json.Unmarshal(stubBytes, &stub); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stub: %v", err)
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

func (s *SmartContract) AcknowledgePayment(ctx contractapi.TransactionContextInterface, paymentDetails PaymentEventDetails) error {
	s.updatePaymentStatusInPDC(ctx, paymentDetails.PayerMSP, paymentDetails.PayeeMSP, paymentDetails.ID, "ACKNOWLEDGED")

	// Emit event
	return s.emitPaymentEvent(ctx, "PaymentAcknowledged", PaymentEventDetails{
		ID:       paymentDetails.ID,
		PayeeMSP: paymentDetails.PayeeMSP,
		PayerMSP: paymentDetails.PayerMSP,
	})
}

// GetBilateralPayments retrieves all payment records from a bilateral PDC
func (s *SmartContract) GetBilateralPayments(ctx contractapi.TransactionContextInterface, msp1, msp2 string) ([]*PaymentDetails, error) {
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

// GetBilateralPaymentsByStatus retrieves payment records filtered by status
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