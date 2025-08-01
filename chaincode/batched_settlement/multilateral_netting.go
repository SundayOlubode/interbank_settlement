package settlement

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// CalculateMultilateralOffset calculates netting across all banks for QUEUED payments
func (s *SmartContract) CalculateMultilateralOffset(ctx contractapi.TransactionContextInterface,
) (*MultiOffsetCalculation, error) {
	// Only process bilateral collections between actual banks (exclude CentralBankMSP)
	bankMSPs := []string{"AccessBankMSP", "GTBankMSP", "ZenithBankMSP", "FirstBankMSP"}

	// Scan **every** bilateral PDC for QUEUED items
	netPos := make(map[string]float64)
	var updates []MultiOffsetUpdate

	// Process each bank pair combination only once by using nested loop with i < j
	for i, a := range bankMSPs {
		for j := i + 1; j < len(bankMSPs); j++ {
			b := bankMSPs[j]
			coll := getCollectionName(a, b)
			iter, err := ctx.GetStub().GetPrivateDataByRange(coll, "", "")
			if err != nil {
				return nil, fmt.Errorf("failed to read PDC %s: %v", coll, err)
			}
			defer iter.Close()

			for iter.HasNext() {
				qr, _ := iter.Next()

				var pd PaymentDetails
				if err := json.Unmarshal(qr.Value, &pd); err != nil {
					continue
				}
				if pd.Status != "QUEUED" {
					continue
				}

				// Build net position: incoming minus outgoing
				netPos[pd.PayeeMSP] += pd.AmountToSettle
				netPos[pd.PayerMSP] -= pd.AmountToSettle

				// Buffer an update to mark this row SETTLED (zero out AmountToSettle)
				updates = append(updates, MultiOffsetUpdate{
					ID:             pd.ID,
					PayerMSP:       pd.PayerMSP,
					PayeeMSP:       pd.PayeeMSP,
					AmountToSettle: 0,
					Status:         "SETTLED",
				})
			}
		}
	}

	return &MultiOffsetCalculation{
		NetPositions: netPos,
		Updates:      updates,
	}, nil
}

// CalculateMultilateralOffsetForBatch calculates netting for a specific batch window
func (s *SmartContract) CalculateMultilateralOffsetForBatch(ctx contractapi.TransactionContextInterface, batchWindow int64) (*MultiOffsetCalculation, error) {
	// Only process bilateral collections between actual banks (exclude CentralBankMSP)
	bankMSPs := []string{"AccessBankMSP", "GTBankMSP", "ZenithBankMSP", "FirstBankMSP"}

	// Scan bilateral PDCs for QUEUED items from the specified batch window
	netPos := make(map[string]float64)
	var updates []MultiOffsetUpdate

	for i, a := range bankMSPs {
		for j := i + 1; j < len(bankMSPs); j++ {
			b := bankMSPs[j]
			coll := getCollectionName(a, b)
			iter, err := ctx.GetStub().GetPrivateDataByRange(coll, "", "")
			if err != nil {
				return nil, fmt.Errorf("failed to read PDC %s: %v", coll, err)
			}
			defer iter.Close()

			for iter.HasNext() {
				qr, _ := iter.Next()

				var pd PaymentDetails
				if err := json.Unmarshal(qr.Value, &pd); err != nil {
					continue
				}

				// Only process QUEUED payments from the specified batch window
				if pd.Status != "QUEUED" || pd.BatchWindow != batchWindow {
					continue
				}

				// Build net position: incoming minus outgoing
				netPos[pd.PayeeMSP] += pd.AmountToSettle
				netPos[pd.PayerMSP] -= pd.AmountToSettle

				// Buffer an update to mark this row SETTLED (zero out AmountToSettle)
				updates = append(updates, MultiOffsetUpdate{
					ID:             pd.ID,
					PayerMSP:       pd.PayerMSP,
					PayeeMSP:       pd.PayeeMSP,
					AmountToSettle: 0,
					Status:         "SETTLED",
				})
			}
		}
	}

	return &MultiOffsetCalculation{
		NetPositions: netPos,
		Updates:      updates,
	}, nil
}

// ApplyMultilateralOffset applies multilateral netting updates
func (s *SmartContract) ApplyMultilateralOffset(ctx contractapi.TransactionContextInterface) error {
	// Read the payload from transient
	trans, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("transient error: %v", err)
	}
	data, ok := trans["multilateralUpdate"]
	if !ok {
		return fmt.Errorf("multilateralUpdate required in transient")
	}
	var payload MultiOffsetCalculation
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %v", err)
	}

	// Apply every queued‐payment update
	for _, u := range payload.Updates {
		coll := getCollectionName(u.PayerMSP, u.PayeeMSP)
		existing, err := ctx.GetStub().GetPrivateData(coll, u.ID)
		if err != nil || existing == nil {
			return fmt.Errorf("payment %s not found", u.ID)
		}
		var pd PaymentDetails
		json.Unmarshal(existing, &pd)

		pd.AmountToSettle = u.AmountToSettle
		pd.Status = u.Status
		updated, _ := json.Marshal(pd)
		if err := ctx.GetStub().PutPrivateData(coll, u.ID, updated); err != nil {
			return fmt.Errorf("write failed %s: %v", u.ID, err)
		}

		// Update public stub status as well
		if err := s.updatePublicPaymentStatus(ctx, u.ID, u.Status); err != nil {
			return fmt.Errorf("failed to update public payment status for %s: %v", u.ID, err)
		}
	}

	// Move real money once per bank
	for msp, net := range payload.NetPositions {
		switch {
		case net < 0:
			if err := s.DebitNetting(ctx, msp, -net); err != nil {
				return err
			}
		case net > 0:
			if err := s.CreditNetting(ctx, msp, net); err != nil {
				return err
			}
		}
	}

	// Create detailed event
	evt := struct {
		NetPositions   map[string]float64 `json:"netPositions"`
		UpdatesCount   int                `json:"updatesCount"`
		TotalSettled   float64            `json:"totalSettled"`
		Timestamp      int64              `json:"timestamp"`
		ProcessedBanks []string           `json:"processedBanks"`
	}{
		NetPositions:   payload.NetPositions,
		UpdatesCount:   len(payload.Updates),
		Timestamp:      time.Now().Unix(),
		ProcessedBanks: getProcessedBanks(payload.NetPositions),
	}

	// Calculate total settled amount
	for _, amount := range payload.NetPositions {
		if amount > 0 {
			evt.TotalSettled += amount
		}
	}

	evtBytes, _ := json.Marshal(evt)
	return ctx.GetStub().SetEvent("MultilateralOffsetExecuted", evtBytes)
}

// ExecuteScheduledMultilateralNetting performs system-wide multilateral netting
// This should be called by Central Bank backend service
func (s *SmartContract) ExecuteScheduledMultilateralNetting(ctx contractapi.TransactionContextInterface) (string, error) {
	// Calculate multilateral offset for all queued payments
	offsetCalc, err := s.CalculateMultilateralOffset(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to calculate multilateral offset: %v", err)
	}

	// Create response structure
	response := struct {
		Success      bool               `json:"success"`
		Message      string             `json:"message"`
		NetPositions map[string]float64 `json:"netPositions"`
		UpdatesCount int                `json:"updatesCount"`
		TotalSettled float64            `json:"totalSettled"`
		Timestamp    int64              `json:"timestamp"`
		EventType    string             `json:"eventType"`
	}{
		Success:      true,
		NetPositions: offsetCalc.NetPositions,
		UpdatesCount: len(offsetCalc.Updates),
		Timestamp:    time.Now().Unix(),
		EventType:    "ScheduledMultilateralNetting",
	}

	// Check if there are any updates to apply
	if len(offsetCalc.Updates) == 0 {
		response.Message = "No queued payments found for multilateral netting"
		response.NetPositions = make(map[string]float64) // Empty map instead of nil

		// Emit event indicating no netting was needed
		evt := struct {
			Message   string `json:"message"`
			Timestamp int64  `json:"timestamp"`
		}{
			Message:   response.Message,
			Timestamp: response.Timestamp,
		}
		evtBytes, _ := json.Marshal(evt)
		ctx.GetStub().SetEvent("MultilateralNettingSkipped", evtBytes)
	} else {
		// Apply the multilateral offset
		err = s.applyMultilateralOffsetInternal(ctx, *offsetCalc)
		if err != nil {
			return "", fmt.Errorf("failed to apply multilateral offset: %v", err)
		}

		response.Message = "Multilateral netting executed successfully"

		// Calculate total settled amount
		for _, amount := range offsetCalc.NetPositions {
			if amount > 0 {
				response.TotalSettled += amount
			}
		}
	}

	// Convert response to JSON and return
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %v", err)
	}

	return string(responseBytes), nil
}

// Internal function to apply multilateral offset (used by scheduled netting)
func (s *SmartContract) applyMultilateralOffsetInternal(ctx contractapi.TransactionContextInterface, payload MultiOffsetCalculation) error {
	// Apply every queued‐payment update
	for _, u := range payload.Updates {
		coll := getCollectionName(u.PayerMSP, u.PayeeMSP)
		existing, err := ctx.GetStub().GetPrivateData(coll, u.ID)
		if err != nil || existing == nil {
			return fmt.Errorf("payment %s not found", u.ID)
		}
		var pd PaymentDetails
		json.Unmarshal(existing, &pd)

		pd.AmountToSettle = u.AmountToSettle
		pd.Status = u.Status
		updated, _ := json.Marshal(pd)
		if err := ctx.GetStub().PutPrivateData(coll, u.ID, updated); err != nil {
			return fmt.Errorf("write failed %s: %v", u.ID, err)
		}

		// Update public stub status as well
		if err := s.updatePublicPaymentStatus(ctx, u.ID, u.Status); err != nil {
			return fmt.Errorf("failed to update public payment status for %s: %v", u.ID, err)
		}
	}

	// Move real money once per bank
	for msp, net := range payload.NetPositions {
		switch {
		case net < 0:
			if err := s.DebitNetting(ctx, msp, -net); err != nil {
				return err
			}
		case net > 0:
			if err := s.CreditNetting(ctx, msp, net); err != nil {
				return err
			}
		}
	}

	// Create detailed event
	evt := struct {
		NetPositions   map[string]float64 `json:"netPositions"`
		UpdatesCount   int                `json:"updatesCount"`
		TotalSettled   float64            `json:"totalSettled"`
		Timestamp      int64              `json:"timestamp"`
		ProcessedBanks []string           `json:"processedBanks"`
		EventType      string             `json:"eventType"`
	}{
		NetPositions:   payload.NetPositions,
		UpdatesCount:   len(payload.Updates),
		Timestamp:      time.Now().Unix(),
		ProcessedBanks: getProcessedBanks(payload.NetPositions),
		EventType:      "ScheduledMultilateralNetting",
	}

	// Calculate total settled amount
	for _, amount := range payload.NetPositions {
		if amount > 0 {
			evt.TotalSettled += amount
		}
	}

	evtBytes, _ := json.Marshal(evt)
	return ctx.GetStub().SetEvent("ScheduledMultilateralNettingExecuted", evtBytes)
}

// DebitNetting subtracts `amount` from the MSP's settlement account.
// Errors if the account doesn't exist or has insufficient funds.
func (s *SmartContract) DebitNetting(
	ctx contractapi.TransactionContextInterface,
	msp string,
	amount float64,
) error {
	coll := fmt.Sprintf("col-settlement-%s", msp)
	acctBytes, err := ctx.GetStub().GetPrivateData(coll, msp)
	if err != nil {
		return fmt.Errorf("failed to read settlement account for %s: %v", msp, err)
	}
	if acctBytes == nil {
		return fmt.Errorf("no settlement account found for %s", msp)
	}

	var acct BankAccount
	if err := json.Unmarshal(acctBytes, &acct); err != nil {
		return fmt.Errorf("failed to unmarshal account for %s: %v", msp, err)
	}

	// Allow negative balances for netting (Central Bank backing)
	// if acct.Balance < amount {
	// 	return fmt.Errorf(
	// 		"insufficient funds: account %s balance %.2f, need %.2f",
	// 		msp, acct.Balance, amount,
	// 	)
	// }

	// Negative means the bank owes the Central Bank
	acct.Balance -= amount

	updated, err := json.Marshal(acct)
	if err != nil {
		return fmt.Errorf("failed to marshal updated account for %s: %v", msp, err)
	}
	if err := ctx.GetStub().PutPrivateData(coll, msp, updated); err != nil {
		return fmt.Errorf("failed to update settlement account for %s: %v", msp, err)
	}

	// Audit event
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
		Balance:   acct.Balance,
		Timestamp: time.Now().Unix(),
	}
	evtBytes, _ := json.Marshal(evt)
	if err := ctx.GetStub().SetEvent("NettingDebitExecuted", evtBytes); err != nil {
		return fmt.Errorf("failed to emit debit event for %s: %v", msp, err)
	}

	return nil
}

// CreditNetting adds `amount` to the MSP's settlement account.
// Creates the account if it doesn't already exist.
func (s *SmartContract) CreditNetting(
	ctx contractapi.TransactionContextInterface,
	msp string,
	amount float64,
) error {
	coll := fmt.Sprintf("col-settlement-%s", msp)
	acctBytes, err := ctx.GetStub().GetPrivateData(coll, msp)
	if err != nil {
		return fmt.Errorf("failed to read settlement account for %s: %v", msp, err)
	}

	var acct BankAccount
	if acctBytes != nil {
		if err := json.Unmarshal(acctBytes, &acct); err != nil {
			return fmt.Errorf("failed to unmarshal account for %s: %v", msp, err)
		}
	} else {
		// No existing account—start from zero
		acct = BankAccount{MSP: msp, Balance: 0}
	}

	acct.Balance += amount
	updated, err := json.Marshal(acct)
	if err != nil {
		return fmt.Errorf("failed to marshal updated account for %s: %v", msp, err)
	}
	if err := ctx.GetStub().PutPrivateData(coll, msp, updated); err != nil {
		return fmt.Errorf("failed to update settlement account for %s: %v", msp, err)
	}

	// Audit event
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
		Balance:   acct.Balance,
		Timestamp: time.Now().Unix(),
	}
	evtBytes, _ := json.Marshal(evt)
	if err := ctx.GetStub().SetEvent("NettingCreditExecuted", evtBytes); err != nil {
		return fmt.Errorf("failed to emit credit event for %s: %v", msp, err)
	}

	return nil
}

// GetMultilateralNettingStatus returns the current status of multilateral netting
func (s *SmartContract) GetMultilateralNettingStatus(ctx contractapi.TransactionContextInterface) (*MultilateralNettingStatus, error) {
	// Only check bilateral collections between actual banks (exclude CentralBankMSP)
	bankMSPs := []string{"AccessBankMSP", "GTBankMSP", "ZenithBankMSP", "FirstBankMSP"}

	// Count total queued payments across all bilateral PDCs
	totalQueued := 0
	totalQueuedAmount := 0.0
	bankCounts := make(map[string]int)
	bankAmounts := make(map[string]float64)

	for i, a := range bankMSPs {
		for j := i + 1; j < len(bankMSPs); j++ {
			b := bankMSPs[j]
			coll := getCollectionName(a, b)
			iter, err := ctx.GetStub().GetPrivateDataByRange(coll, "", "")
			if err != nil {
				continue // Skip inaccessible collections
			}
			defer iter.Close()

			for iter.HasNext() {
				qr, _ := iter.Next()
				var pd PaymentDetails
				if err := json.Unmarshal(qr.Value, &pd); err != nil {
					continue
				}
				if pd.Status != "QUEUED" {
					continue
				}

				totalQueued++
				totalQueuedAmount += pd.AmountToSettle

				// Count by payer bank
				bankCounts[pd.PayerMSP]++
				bankAmounts[pd.PayerMSP] += pd.AmountToSettle
			}
		}
	}

	return &MultilateralNettingStatus{
		TotalQueuedPayments: totalQueued,
		TotalQueuedAmount:   totalQueuedAmount,
		BankCounts:          bankCounts,
		BankAmounts:         bankAmounts,
		LastUpdated:         time.Now().Unix(),
	}, nil
}

// Helper function to extract processed banks from net positions
func getProcessedBanks(netPositions map[string]float64) []string {
	banks := make([]string, 0, len(netPositions))
	for bank := range netPositions {
		banks = append(banks, bank)
	}
	return banks
}

// MultilateralNettingStatus represents the current netting status
type MultilateralNettingStatus struct {
	TotalQueuedPayments int                `json:"totalQueuedPayments"`
	TotalQueuedAmount   float64            `json:"totalQueuedAmount"`
	BankCounts          map[string]int     `json:"bankCounts"`
	BankAmounts         map[string]float64 `json:"bankAmounts"`
	LastUpdated         int64              `json:"lastUpdated"`
}
