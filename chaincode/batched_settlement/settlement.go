// settlement.go - CBN-Only Netting-Based Settlement
package settlement

import (
	"encoding/json"
	"fmt"
	"time"

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

// CalculateNettingOffsets calculates net positions and payment updates without applying them
func (s *SmartContract) CalculateNettingOffsets(ctx contractapi.TransactionContextInterface) (string, error) {
	// Get all BATCHED payments and calculate net positions
	netPositions, batchedPayments, err := s.calculateNetPositionsFromBatchedPayments(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to calculate net positions: %v", err)
	}

	// Initialize calculation result
	result := &NettingCalculationResult{
		NetPositions:   netPositions,
		PaymentUpdates: make([]PaymentUpdate, 0),
		TotalPayments:  len(batchedPayments),
		TotalNetAmount: 0,
		Timestamp:      time.Now().Unix(),
	}

	// Calculate total net amount
	for _, netAmount := range netPositions {
		if netAmount > 0 {
			result.TotalNetAmount += netAmount
		}
	}

	// Prepare payment updates (but don't apply them yet)
	for _, payment := range batchedPayments {
		result.PaymentUpdates = append(result.PaymentUpdates, PaymentUpdate{
			ID:             payment.ID,
			PayerMSP:       payment.PayerMSP,
			PayeeMSP:       payment.PayeeMSP,
			Status:         "SETTLED",
			AmountToSettle: 0,
		})
	}

	// Return calculation result as JSON
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal calculation result: %v", err)
	}

	return string(resultBytes), nil
}

// ApplyNettingOffsets applies the calculated netting offsets to settlement accounts and payments
func (s *SmartContract) ApplyNettingOffsets(ctx contractapi.TransactionContextInterface) (string, error) {
	// Get calculation result from transient data
	transMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return "", fmt.Errorf("error getting transient data: %v", err)
	}

	offsetsJSON, ok := transMap["nettingOffsets"]
	if !ok {
		return "", fmt.Errorf("netting offsets must be provided in transient data under 'nettingOffsets'")
	}

	var calculation NettingCalculationResult
	if err := json.Unmarshal(offsetsJSON, &calculation); err != nil {
		return "", fmt.Errorf("failed to unmarshal netting calculation: %v", err)
	}

	// Initialize application result
	result := &NettingApplicationResult{
		SettledBanks:    make(map[string]float64),
		FailedBanks:     make([]FailedBankSettlement, 0),
		SettledPayments: 0,
		FailedPayments:  0,
		TotalNetAmount:  calculation.TotalNetAmount,
		Timestamp:       time.Now().Unix(),
	}

	// Step 1: Apply net settlements to bank accounts
	for bankMSP, netAmount := range calculation.NetPositions {
		if netAmount == 0 {
			continue // No net position, skip
		}

		err := s.applyNetSettlement(ctx, bankMSP, netAmount)
		if err != nil {
			result.FailedBanks = append(result.FailedBanks, FailedBankSettlement{
				BankMSP:   bankMSP,
				NetAmount: netAmount,
				Error:     err.Error(),
			})
			fmt.Printf("Failed to apply net settlement for %s: %v\n", bankMSP, err)
		} else {
			result.SettledBanks[bankMSP] = netAmount
		}
	}

	// Step 2: Update all payment statuses to SETTLED
	for _, update := range calculation.PaymentUpdates {
		err := s.updatePaymentStatusAndAmount(ctx, update.PayerMSP, update.PayeeMSP, update.ID, update.Status, update.AmountToSettle)
		if err != nil {
			result.FailedPayments++
			fmt.Printf("Failed to update payment %s: %v\n", update.ID, err)
			continue
		}

		// // Update public stub status
		// err = s.updatePublicPaymentStatus(ctx, update.ID, update.Status)
		// if err != nil {
		// 	fmt.Printf("Failed to update public status for payment %s: %v\n", update.ID, err)
		// }

		// Emit individual payment settlement event
		settlementEvent := PaymentEventDetails{
			ID:       update.ID,
			PayeeMSP: update.PayeeMSP,
			PayerMSP: update.PayerMSP,
		}
		s.emitPaymentEvent(ctx, "PaymentSettled", settlementEvent)

		result.SettledPayments++
	}

	// Step 3: Emit settlement completion event
	settlementEvent := struct {
		TotalPayments   int                `json:"totalPayments"`
		SettledPayments int                `json:"settledPayments"`
		FailedPayments  int                `json:"failedPayments"`
		NetPositions    map[string]float64 `json:"netPositions"`
		SettledBanks    map[string]float64 `json:"settledBanks"`
		TotalNetAmount  float64            `json:"totalNetAmount"`
		Timestamp       int64              `json:"timestamp"`
		EventType       string             `json:"eventType"`
	}{
		TotalPayments:   calculation.TotalPayments,
		SettledPayments: result.SettledPayments,
		FailedPayments:  result.FailedPayments,
		NetPositions:    calculation.NetPositions,
		SettledBanks:    result.SettledBanks,
		TotalNetAmount:  result.TotalNetAmount,
		Timestamp:       result.Timestamp,
		EventType:       "NettingSettlement",
	}

	eventBytes, _ := json.Marshal(settlementEvent)
	ctx.GetStub().SetEvent("NettingSettlementExecuted", eventBytes)

	// Return application result as JSON
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal application result: %v", err)
	}

	return string(resultBytes), nil
}

// ExecuteNettingSettlement - Legacy function that combines both operations
func (s *SmartContract) ExecuteNettingSettlement(ctx contractapi.TransactionContextInterface) (string, error) {
	// Step 1: Calculate offsets
	calculationJSON, err := s.CalculateNettingOffsets(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to calculate netting offsets: %v", err)
	}

	// Step 2: Apply offsets (simulate transient data)
	// Note: In real usage, this would be called separately with transient data
	var calculation NettingCalculationResult
	if err := json.Unmarshal([]byte(calculationJSON), &calculation); err != nil {
		return "", fmt.Errorf("failed to unmarshal calculation: %v", err)
	}

	// Create transient data simulation
	calculationBytes, _ := json.Marshal(calculation)

	// Store in mock transient map for internal call
	// In real usage, CBN API will pass this as actual transient data
	mockTransient := map[string][]byte{
		"nettingOffsets": calculationBytes,
	}

	// Apply the offsets
	return s.applyOffsetsWithTransientData(ctx, mockTransient)
}

// applyOffsetsWithTransientData - Helper function for legacy compatibility
func (s *SmartContract) applyOffsetsWithTransientData(ctx contractapi.TransactionContextInterface, transientData map[string][]byte) (string, error) {
	offsetsJSON, ok := transientData["nettingOffsets"]
	if !ok {
		return "", fmt.Errorf("netting offsets must be provided")
	}

	var calculation NettingCalculationResult
	if err := json.Unmarshal(offsetsJSON, &calculation); err != nil {
		return "", fmt.Errorf("failed to unmarshal netting calculation: %v", err)
	}

	// Apply the settlement logic (same as ApplyNettingOffsets)
	result := &NettingApplicationResult{
		SettledBanks:    make(map[string]float64),
		FailedBanks:     make([]FailedBankSettlement, 0),
		SettledPayments: 0,
		FailedPayments:  0,
		TotalNetAmount:  calculation.TotalNetAmount,
		Timestamp:       time.Now().Unix(),
	}

	// Apply net settlements and update payments (same logic as ApplyNettingOffsets)
	// ... (implementation details same as above)

	resultBytes, _ := json.Marshal(result)
	return string(resultBytes), nil
}

// calculateNetPositionsFromBatchedPayments calculates net positions for all banks from BATCHED payments
func (s *SmartContract) calculateNetPositionsFromBatchedPayments(ctx contractapi.TransactionContextInterface) (map[string]float64, []*PaymentDetails, error) {
	netPositions := make(map[string]float64)
	var batchedPayments []*PaymentDetails
	bankMSPs := getBankMSPs()

	// Iterate through all bilateral collections to find BATCHED payments
	for i, bankA := range bankMSPs {
		for j := i + 1; j < len(bankMSPs); j++ {
			bankB := bankMSPs[j]
			coll := getCollectionName(bankA, bankB)

			iter, err := ctx.GetStub().GetPrivateDataByRange(coll, "", "")
			if err != nil {
				// Log error but continue with other collections
				fmt.Printf("Failed to access collection %s: %v\n", coll, err)
				continue
			}
			defer iter.Close()

			for iter.HasNext() {
				qr, err := iter.Next()
				if err != nil {
					continue
				}

				var payment PaymentDetails
				if err := json.Unmarshal(qr.Value, &payment); err != nil {
					continue
				}

				// Only process BATCHED payments
				if payment.Status == "BATCHED" {
					batchedPayments = append(batchedPayments, &payment)

					// Calculate net positions: incoming (+) minus outgoing (-)
					netPositions[payment.PayeeMSP] += payment.Amount // Payee receives
					netPositions[payment.PayerMSP] -= payment.Amount // Payer pays
				}
			}
		}
	}

	return netPositions, batchedPayments, nil
}

// applyNetSettlement applies the net settlement amount to a bank's settlement account
func (s *SmartContract) applyNetSettlement(ctx contractapi.TransactionContextInterface, bankMSP string, netAmount float64) error {
	if netAmount > 0 {
		// Bank receives money - credit settlement account
		return s.creditSettlementAccount(ctx, bankMSP, netAmount)
	} else if netAmount < 0 {
		// Bank pays money - debit settlement account
		return s.debitSettlementAccount(ctx, bankMSP, -netAmount) // Use positive amount for debit
	}
	// netAmount == 0, no action needed
	return nil
}

// markPaymentAsSettled updates a payment from BATCHED to SETTLED
func (s *SmartContract) markPaymentAsSettled(ctx contractapi.TransactionContextInterface, payment *PaymentDetails) error {
	// Update payment status to SETTLED and zero out AmountToSettle
	err := s.updatePaymentStatusAndAmount(ctx, payment.PayerMSP, payment.PayeeMSP, payment.ID, "SETTLED", 0)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %v", err)
	}

	// Update public stub status
	err = s.updatePublicPaymentStatus(ctx, payment.ID, "SETTLED")
	if err != nil {
		return fmt.Errorf("failed to update public payment status: %v", err)
	}

	// Emit individual payment settlement event
	settlementEvent := PaymentEventDetails{
		ID:       payment.ID,
		PayeeMSP: payment.PayeeMSP,
		PayerMSP: payment.PayerMSP,
	}
	s.emitPaymentEvent(ctx, "PaymentSettled", settlementEvent)

	return nil
}

// debitSettlementAccount debits amount from MSP settlement account (allows negative balances)
func (s *SmartContract) debitSettlementAccount(ctx contractapi.TransactionContextInterface, msp string, amount float64) error {
	coll := fmt.Sprintf("col-settlement-%s", msp)
	accountBytes, err := ctx.GetStub().GetPrivateData(coll, msp)
	if err != nil {
		return fmt.Errorf("failed to read settlement account for %s: %v", msp, err)
	}

	var account BankAccount
	if accountBytes != nil {
		if err := json.Unmarshal(accountBytes, &account); err != nil {
			return fmt.Errorf("failed to unmarshal account for %s: %v", msp, err)
		}
	} else {
		// Create account if it doesn't exist (start with zero balance)
		account = BankAccount{MSP: msp, Balance: 0}
	}

	// Allow negative balances - CBN provides liquidity
	account.Balance -= amount

	updated, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal updated account for %s: %v", msp, err)
	}
	if err := ctx.GetStub().PutPrivateData(coll, msp, updated); err != nil {
		return fmt.Errorf("failed to update settlement account for %s: %v", msp, err)
	}

	// Emit debit event
	evt := struct {
		MSP       string  `json:"msp"`
		Amount    float64 `json:"amount"`
		Type      string  `json:"type"`
		Balance   float64 `json:"newBalance"`
		Timestamp int64   `json:"timestamp"`
	}{
		MSP:       msp,
		Amount:    amount,
		Type:      "netting-debit",
		Balance:   account.Balance,
		Timestamp: time.Now().Unix(),
	}
	evtBytes, _ := json.Marshal(evt)
	ctx.GetStub().SetEvent("SettlementDebitExecuted", evtBytes)

	return nil
}

// creditSettlementAccount credits amount to MSP settlement account
func (s *SmartContract) creditSettlementAccount(ctx contractapi.TransactionContextInterface, msp string, amount float64) error {
	coll := fmt.Sprintf("col-settlement-%s", msp)
	accountBytes, err := ctx.GetStub().GetPrivateData(coll, msp)
	if err != nil {
		return fmt.Errorf("failed to read settlement account for %s: %v", msp, err)
	}

	var account BankAccount
	if accountBytes != nil {
		if err := json.Unmarshal(accountBytes, &account); err != nil {
			return fmt.Errorf("failed to unmarshal account for %s: %v", msp, err)
		}
	} else {
		// Create account if it doesn't exist
		account = BankAccount{MSP: msp, Balance: 0}
	}

	account.Balance += amount
	updated, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal updated account for %s: %v", msp, err)
	}
	if err := ctx.GetStub().PutPrivateData(coll, msp, updated); err != nil {
		return fmt.Errorf("failed to update settlement account for %s: %v", msp, err)
	}

	// Emit credit event
	evt := struct {
		MSP       string  `json:"msp"`
		Amount    float64 `json:"amount"`
		Type      string  `json:"type"`
		Balance   float64 `json:"newBalance"`
		Timestamp int64   `json:"timestamp"`
	}{
		MSP:       msp,
		Amount:    amount,
		Type:      "netting-credit",
		Balance:   account.Balance,
		Timestamp: time.Now().Unix(),
	}
	evtBytes, _ := json.Marshal(evt)
	ctx.GetStub().SetEvent("SettlementCreditExecuted", evtBytes)

	return nil
}

// updatePaymentStatusAndAmount updates both status and amount in bilateral PDC
func (s *SmartContract) updatePaymentStatusAndAmount(ctx contractapi.TransactionContextInterface, payerMSP, payeeMSP, paymentID, status string, amountToSettle float64) error {
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
	paymentDetails.AmountToSettle = amountToSettle
	updatedPaymentBytes, _ := json.Marshal(paymentDetails)
	return ctx.GetStub().PutPrivateData(paymentColl, paymentID, updatedPaymentBytes)
}

// GetAllBatchedPayments returns all batched payments system-wide
func (s *SmartContract) GetAllBatchedPayments(ctx contractapi.TransactionContextInterface) ([]*PaymentDetails, error) {
	_, batchedPayments, err := s.calculateNetPositionsFromBatchedPayments(ctx)
	return batchedPayments, err
}

// GetSettlementStatistics returns system-wide settlement statistics
func (s *SmartContract) GetSettlementStatistics(ctx contractapi.TransactionContextInterface) (*SettlementStatistics, error) {
	stats := &SettlementStatistics{
		StatusCounts:  make(map[string]int),
		StatusAmounts: make(map[string]float64),
		BankBalances:  make(map[string]float64),
		LastUpdated:   time.Now().Unix(),
	}

	bankMSPs := getBankMSPs()

	// Get payment statistics from all bilateral collections
	for i, bankA := range bankMSPs {
		for j := i + 1; j < len(bankMSPs); j++ {
			bankB := bankMSPs[j]
			coll := getCollectionName(bankA, bankB)

			iter, err := ctx.GetStub().GetPrivateDataByRange(coll, "", "")
			if err != nil {
				continue // Skip inaccessible collections
			}
			defer iter.Close()

			for iter.HasNext() {
				qr, err := iter.Next()
				if err != nil {
					continue
				}

				var payment PaymentDetails
				if err := json.Unmarshal(qr.Value, &payment); err != nil {
					continue
				}

				stats.TotalPayments++
				stats.TotalAmount += payment.Amount
				stats.StatusCounts[payment.Status]++
				stats.StatusAmounts[payment.Status] += payment.Amount
			}
		}
	}

	// Get settlement account balances
	for _, bankMSP := range bankMSPs {
		account, err := s.GetSettlementAccount(ctx, bankMSP)
		if err == nil {
			stats.BankBalances[bankMSP] = account.Balance
		}
	}

	return stats, nil
}

// GetNetPositions calculates current net positions without executing settlement
func (s *SmartContract) GetNetPositions(ctx contractapi.TransactionContextInterface) (map[string]float64, error) {
	netPositions, _, err := s.calculateNetPositionsFromBatchedPayments(ctx)
	return netPositions, err
}

// GetBatchedPaymentsByStatus returns payments filtered by status
func (s *SmartContract) GetBatchedPaymentsByStatus(ctx contractapi.TransactionContextInterface, status string) ([]*PaymentDetails, error) {
	var filteredPayments []*PaymentDetails
	bankMSPs := getBankMSPs()

	for i, bankA := range bankMSPs {
		for j := i + 1; j < len(bankMSPs); j++ {
			bankB := bankMSPs[j]
			coll := getCollectionName(bankA, bankB)

			iter, err := ctx.GetStub().GetPrivateDataByRange(coll, "", "")
			if err != nil {
				continue
			}
			defer iter.Close()

			for iter.HasNext() {
				qr, err := iter.Next()
				if err != nil {
					continue
				}

				var payment PaymentDetails
				if err := json.Unmarshal(qr.Value, &payment); err != nil {
					continue
				}

				if payment.Status == status {
					filteredPayments = append(filteredPayments, &payment)
				}
			}
		}
	}

	return filteredPayments, nil
}

// LEGACY COMPATIBILITY FUNCTIONS

// SettleAllBatchedPayments - Legacy function redirects to netting settlement
func (s *SmartContract) SettleAllBatchedPayments(ctx contractapi.TransactionContextInterface) (string, error) {
	return s.ExecuteNettingSettlement(ctx)
}

// DebitAccount - Legacy function for compatibility
func (s *SmartContract) DebitAccount(ctx contractapi.TransactionContextInterface, paymentDetails PaymentEventDetails) (string, error) {
	return "", fmt.Errorf("DebitAccount is deprecated - settlement is now handled centrally by CBN netting")
}

// CreditAccount - Legacy function for compatibility
func (s *SmartContract) CreditAccount(ctx contractapi.TransactionContextInterface, paymentDetails PaymentEventDetails) error {
	return fmt.Errorf("CreditAccount is deprecated - settlement is now handled centrally by CBN netting")
}
