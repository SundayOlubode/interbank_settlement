package main

import (
	"encoding/json"
	"fmt"

	// "strings"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// PaymentStub is the lightweight public record stored in the world state.
type PaymentStub struct {
	ID        string `json:"id"`
	Hash      string `json:"hash"`
	PayerMSP  string `json:"payerMSP"`
	PayeeMSP  string `json:"payeeMSP"`
	Status    string `json:"status"` // PENDING, SETTLED, DISPUTED
	Timestamp string `json:"timestamp"`
}

// PaymentDetails is the sensitive payload stored in the bilateral PDC.
type PaymentDetails struct {
	PaymentStub
	PayerAcct string  `json:"payerAcct"`
	PayeeAcct string  `json:"payeeAcct"`
	Amount    float64 `json:"amount"`
	Reference string  `json:"reference"`
}

// PaymentsContract defines the chain‑code struct.
// Implements fabric‑contract‑api shim.

type PaymentsContract struct {
	contractapi.Contract
}

// helper: bilateral collection name by two MSP IDs (alphabetical)
func collectionName(mspA, mspB string) string {
	if mspA > mspB {
		mspA, mspB = mspB, mspA
	}
	return fmt.Sprintf("col-%s-%s", mspA, mspB)
}

// CreatePayment ‑ invoked by the payer bank.
// The private JSON is expected in transient map under key "payment".
func (c *PaymentsContract) CreatePayment(ctx contractapi.TransactionContextInterface, paymentID string) error {
	// 1. Pull MSP ID of invoker (payer)
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return err
	}

	// 2. Fetch transient map (private payload)
	transient, err := ctx.GetStub().GetTransient()
	if err != nil {
		return err
	}
	dataBytes, ok := transient["payment"]
	if !ok {
		return fmt.Errorf("missing payment transient field")
	}

	// 3. Unmarshal
	var details PaymentDetails
	if err := json.Unmarshal(dataBytes, &details); err != nil {
		return err
	}

	// 4. Basic sanity
	if details.Amount <= 0 {
		return fmt.Errorf("amount must be > 0")
	}

	// 5. Determine bilateral collection
	coll := collectionName(details.PayerMSP, details.PayeeMSP)

	// 6. Write private data
	if err := ctx.GetStub().PutPrivateData(coll, paymentID, dataBytes); err != nil {
		return err
	}

	// 7. Write public stub (hash is supplied by client to avoid re‑hash on chain, saving compute)
	stubJSON, _ := json.Marshal(PaymentStub{
		ID:        paymentID,
		Hash:      details.Hash,
		PayerMSP:  details.PayerMSP,
		PayeeMSP:  details.PayeeMSP,
		Status:    "PENDING",
		Timestamp: details.Timestamp,
	})

	ctx.GetStub().PutState(paymentID, stubJSON)

	evtBytes, _ := json.Marshal(stubJSON)
	return ctx.GetStub().SetEvent("PaymentPending", evtBytes) // Event name + payload
}

// SettlePayment ‑ invoked by payee bank after confirming receipt.
func (c *PaymentsContract) SettlePayment(ctx contractapi.TransactionContextInterface, paymentID string) error {
	// fetch stub
	stubBytes, err := ctx.GetStub().GetState(paymentID)
	if err != nil || stubBytes == nil {
		return fmt.Errorf("payment not found")
	}
	var stub PaymentStub
	_ = json.Unmarshal(stubBytes, &stub)
	stub.Status = "SETTLED"
	newBytes, _ := json.Marshal(stub)
	return ctx.GetStub().PutState(paymentID, newBytes)
}

// ReadPayment returns the public stub.
func (c *PaymentsContract) ReadPayment(ctx contractapi.TransactionContextInterface, paymentID string) (*PaymentStub, error) {
	bytes, err := ctx.GetStub().GetState(paymentID)
	if err != nil || bytes == nil {
		return nil, fmt.Errorf("not found")
	}
	var stub PaymentStub
	_ = json.Unmarshal(bytes, &stub)
	return &stub, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&PaymentsContract{})
	if err != nil {
		panic(err)
	}
	if err := chaincode.Start(); err != nil {
		panic(err)
	}
}
