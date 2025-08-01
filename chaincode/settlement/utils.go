package settlement

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

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

// Helper function to check if MSP is authorized
func (s *SmartContract) isAuthorizedMSP(mspID string) bool {
	for _, authorizedMSP := range authorizedMSPs {
		if mspID == authorizedMSP {
			return true
		}
	}
	return false
}

var authorizedMSPs = []string{
	"AccessBankMSP",
	"GTBankMSP",
	"ZenithBankMSP",
	"FirstBankMSP",
}
