package settlement

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

var col_BVN = "col-BVN"

// getCollectionName returns the PDC name for a payer/payee pair in alphabetical order
func getCollectionName(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return fmt.Sprintf("col-%s-%s", a, b)
}

// computeHash returns a SHA256 hash of the input data
func computeHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// createHashablePayment creates a consistent representation for hashing
func createHashablePayment(details PaymentDetails) []byte {
	// Create a copy without status field for consistent hashing
	hashable := struct {
		ID        string   `json:"id"`
		PayerAcct string   `json:"payerAcct"`
		PayeeAcct string   `json:"payeeAcct"`
		Amount    float64  `json:"amount"`
		Currency  string   `json:"currency"`
		BVN       string   `json:"bvn"`
		PayerMSP  string   `json:"payerMSP"`
		PayeeMSP  string   `json:"payeeMSP"`
		Timestamp int64    `json:"timestamp"`
		User      BankUser `json:"user"`
	}{
		ID:        details.ID,
		PayerAcct: details.PayerAcct,
		PayeeAcct: details.PayeeAcct,
		Amount:    details.Amount,
		Currency:  details.Currency,
		BVN:       details.BVN,
		PayerMSP:  details.PayerMSP,
		PayeeMSP:  details.PayeeMSP,
		Timestamp: details.Timestamp,
		User:      details.User,
	}

	data, _ := json.Marshal(hashable)
	return data
}

// emitPaymentEvent emits a Fabric event with the given name and payload
func (s *SmartContract) emitPaymentEvent(ctx contractapi.TransactionContextInterface, eventName string, payload PaymentEventDetails) error {
	evtBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event details: %v", err)
	}

	if err := ctx.GetStub().SetEvent(eventName, evtBytes); err != nil {
		return fmt.Errorf("failed to set %s event: %v", eventName, err)
	}
	return nil
}

// emitBatchEvent emits a batch-related event with enhanced details
func (s *SmartContract) emitBatchEvent(ctx contractapi.TransactionContextInterface, eventName string, payload BatchProcessingEvent) error {
	payload.Timestamp = time.Now().Unix()

	evtBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal batch event details: %v", err)
	}

	if err := ctx.GetStub().SetEvent(eventName, evtBytes); err != nil {
		return fmt.Errorf("failed to set %s event: %v", eventName, err)
	}
	return nil
}

// emitSettlementEvent emits settlement-related events with detailed information
func (s *SmartContract) emitSettlementEvent(ctx contractapi.TransactionContextInterface, eventName string, payload interface{}) error {
	evtBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal settlement event details: %v", err)
	}

	if err := ctx.GetStub().SetEvent(eventName, evtBytes); err != nil {
		return fmt.Errorf("failed to set %s event: %v", eventName, err)
	}
	return nil
}

// Helper function to check if MSP is authorized
func (s *SmartContract) isAuthorizedMSP(mspID string) bool {
	for _, authorizedMSP := range authorizedMSPs {
		if mspID == authorizedMSP {
			return true
		}
	}
	return false
}

// Helper function to check if MSP is an authorized bank (excludes Central Bank)
func (s *SmartContract) isAuthorizedBank(mspID string) bool {
	bankMSPs := getBankMSPs()
	for _, authorizedBank := range bankMSPs {
		if mspID == authorizedBank {
			return true
		}
	}
	return false
}

// validateBatchWindow checks if a batch window is valid (not in the future)
func validateBatchWindow(batchWindow int64) error {
	currentWindow := getCurrentBatchWindow()
	if batchWindow > currentWindow {
		return fmt.Errorf("batch window %d is in the future (current: %d)", batchWindow, currentWindow)
	}
	return nil
}

// getBatchWindowInfo returns detailed information about a batch window
func getBatchWindowInfo(batchWindow int64) BatchWindowInfo {
	startTime := getBatchWindowStart(batchWindow)
	endTime := getBatchWindowEnd(batchWindow)
	currentWindow := getCurrentBatchWindow()

	var status string
	switch {
	case batchWindow > currentWindow:
		status = "FUTURE"
	case batchWindow == currentWindow:
		status = "ACTIVE"
	case batchWindow == currentWindow-1:
		status = "PROCESSING"
	default:
		status = "COMPLETED"
	}

	return BatchWindowInfo{
		WindowID:  batchWindow,
		StartTime: startTime,
		EndTime:   endTime,
		Status:    status,
		IsCurrent: batchWindow == currentWindow,
	}
}

// formatBatchWindowTime formats a batch window timestamp for display
func formatBatchWindowTime(batchWindow int64) string {
	startTime := getBatchWindowStart(batchWindow)
	return startTime.Format("15:04:05")
}

// calculateBatchWindowDuration returns the duration of a batch window (always 2 minutes)
func calculateBatchWindowDuration() time.Duration {
	return 2 * time.Minute
}

// getNextBatchWindow returns the next batch window after the given one
func getNextBatchWindow(currentWindow int64) int64 {
	return currentWindow + 1
}

// getPreviousBatchWindow returns the previous batch window before the given one
func getPreviousBatchWindow(currentWindow int64) int64 {
	if currentWindow <= 0 {
		return 0
	}
	return currentWindow - 1
}

// generatePaymentEventId generates a unique event ID for payment events
func generatePaymentEventId(paymentID string, eventType string) string {
	timestamp := time.Now().Unix()
	data := fmt.Sprintf("%s:%s:%d", paymentID, eventType, timestamp)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for shorter ID
}

// validatePaymentStatus checks if a payment status transition is valid
func validatePaymentStatus(currentStatus, newStatus string) error {
	validTransitions := map[string][]string{
		"PENDING":      {"ACKNOWLEDGED"},
		"ACKNOWLEDGED": {"BATCHED", "QUEUED"},
		"BATCHED":      {"DEBITED", "QUEUED"},
		"DEBITED":      {"SETTLED"},
		"QUEUED":       {"SETTLED", "BATCHED"}, // Can be re-batched or settled through netting
		"SETTLED":      {},                     // Terminal state
	}

	allowedNext, exists := validTransitions[currentStatus]
	if !exists {
		return fmt.Errorf("unknown current status: %s", currentStatus)
	}

	for _, allowed := range allowedNext {
		if newStatus == allowed {
			return nil
		}
	}

	return fmt.Errorf("invalid status transition from %s to %s", currentStatus, newStatus)
}

// createAuditTrail creates an audit trail entry for payment status changes
func createAuditTrail(paymentID, oldStatus, newStatus, msp string) AuditTrailEntry {
	return AuditTrailEntry{
		PaymentID: paymentID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		ChangedBy: msp,
		Timestamp: time.Now().Unix(),
		EventID:   generatePaymentEventId(paymentID, "status_change"),
	}
}

// Helper function to determine if a batch window is ready for settlement
func isBatchWindowReadyForSettlement(batchWindow int64) bool {
	currentWindow := getCurrentBatchWindow()
	// A window is ready for settlement if it's the previous window or older
	return batchWindow < currentWindow
}

// Helper function to get settlement cycle information
func getSettlementCycleInfo() SettlementCycleInfo {
	now := time.Now()
	currentWindow := getCurrentBatchWindow()
	windowStart := getBatchWindowStart(currentWindow)
	windowEnd := getBatchWindowEnd(currentWindow)

	timeInWindow := now.Sub(windowStart)
	timeRemaining := windowEnd.Sub(now)

	return SettlementCycleInfo{
		CurrentWindow:   currentWindow,
		WindowStart:     windowStart,
		WindowEnd:       windowEnd,
		TimeInWindow:    timeInWindow,
		TimeRemaining:   timeRemaining,
		ProgressPercent: float64(timeInWindow) / float64(2*time.Minute) * 100,
	}
}

var authorizedMSPs = []string{
	"AccessBankMSP",
	"GTBankMSP",
	"ZenithBankMSP",
	"FirstBankMSP",
	"CentralBankMSP", // Central Bank for settlement operations only, no bilateral collections
}

// getBankMSPs returns only the bank MSPs (excluding Central Bank)
func getBankMSPs() []string {
	return []string{
		"AccessBankMSP",
		"GTBankMSP",
		"ZenithBankMSP",
		"FirstBankMSP",
	}
}

// Supporting types for enhanced utility functions
type BatchWindowInfo struct {
	WindowID  int64     `json:"windowId"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Status    string    `json:"status"`
	IsCurrent bool      `json:"isCurrent"`
}

type AuditTrailEntry struct {
	PaymentID string `json:"paymentId"`
	OldStatus string `json:"oldStatus"`
	NewStatus string `json:"newStatus"`
	ChangedBy string `json:"changedBy"`
	Timestamp int64  `json:"timestamp"`
	EventID   string `json:"eventId"`
}

type SettlementCycleInfo struct {
	CurrentWindow   int64         `json:"currentWindow"`
	WindowStart     time.Time     `json:"windowStart"`
	WindowEnd       time.Time     `json:"windowEnd"`
	TimeInWindow    time.Duration `json:"timeInWindow"`
	TimeRemaining   time.Duration `json:"timeRemaining"`
	ProgressPercent float64       `json:"progressPercent"`
}
