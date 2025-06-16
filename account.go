// package main

// import (
// 	"encoding/json"
// 	"fmt"
// 	"strings"

// 	"github.com/hyperledger/fabric-contract-api-go/contractapi"
// )

// // PaymentStub is the lightweight public record stored in the world state.
// type PaymentStub struct {
// 	ID        string `json:"id"`
// 	Hash      string `json:"hash"`
// 	PayerMSP  string `json:"payerMSP"`
// 	PayeeMSP  string `json:"payeeMSP"`
// 	Status    string `json:"status"` // PENDING, SETTLED, DISPUTED
// 	Timestamp string `json:"timestamp"`
// }

// // PaymentDetails is the sensitive payload stored in the bilateral PDC.
// type PaymentDetails struct {
// 	PaymentStub
// 	PayerAcct string  `json:"payerAcct"`
// 	PayeeAcct string  `json:"payeeAcct"`
// 	Amount    float64 `json:"amount"`
// 	Reference string  `json:"reference"`
// }

// // PaymentsContract defines the chain‑code struct.
// // Implements fabric‑contract‑api shim.

// type PaymentsContract struct {
// 	contractapi.Contract
// }

// // helper: bilateral collection name by two MSP IDs (alphabetical)
// func collectionName(mspA, mspB string) string {
// 	if mspA > mspB {
// 		mspA, mspB = mspB, mspA
// 	}
// 	return fmt.Sprintf("col-%s-%s", mspA, mspB)
// }

// // CreatePayment ‑ invoked by the payer bank.
// // The private JSON is expected in transient map under key "payment".
// func (c *PaymentsContract) CreatePayment(ctx contractapi.TransactionContextInterface, paymentID string) error {
// 	// Pull MSP ID of invoker (payer)
// 	_, err := ctx.GetClientIdentity().GetMSPID()
// 	if err != nil {
// 		return err
// 	}

// 	// Fetch transient map (private payload)
// 	transient, err := ctx.GetStub().GetTransient()
// 	if err != nil {
// 		return err
// 	}
// 	dataBytes, ok := transient["payment"]
// 	if !ok {
// 		return fmt.Errorf("missing payment transient field")
// 	}

// 	// Unmarshal
// 	var details PaymentDetails
// 	if err := json.Unmarshal(dataBytes, &details); err != nil {
// 		return err
// 	}

// 	if details.Amount <= 0 {
// 		return fmt.Errorf("amount must be > 0")
// 	}

// 	// Determine bilateral collection
// 	coll := collectionName(details.PayerMSP, details.PayeeMSP)

// 	// Write private data
// 	if err := ctx.GetStub().PutPrivateData(coll, paymentID, dataBytes); err != nil {
// 		return err
// 	}

// 	// Write public stub
// 	stub := PaymentStub{
// 		ID:        paymentID,
// 		Hash:      details.Hash,
// 		PayerMSP:  details.PayerMSP,
// 		PayeeMSP:  details.PayeeMSP,
// 		Status:    "PENDING",
// 		Timestamp: details.Timestamp,
// 	}
// 	stubBytes, _ := json.Marshal(stub)
// 	ctx.GetStub().PutState(paymentID, stubBytes)

// 	// MINIMAL event
// 	evt := struct {
// 		ID       string `json:"id"`
// 		PayeeMSP string `json:"payeeMSP"`
// 	}{
// 		ID:       paymentID,
// 		PayeeMSP: details.PayeeMSP,
// 	}
// 	evtBytes, _ := json.Marshal(evt)
// 	return ctx.GetStub().SetEvent("PaymentPending", evtBytes) // Event name + payload
// }

// // SettlePayment ‑ invoked by payee bank after confirming receipt.
// func (c *PaymentsContract) SettlePayment(ctx contractapi.TransactionContextInterface, paymentID string) error {
// 	// fetch stub
// 	stubBytes, err := ctx.GetStub().GetState(paymentID)
// 	if err != nil || stubBytes == nil {
// 		return fmt.Errorf("payment not found")
// 	}
// 	var stub PaymentStub
// 	_ = json.Unmarshal(stubBytes, &stub)
// 	stub.Status = "SETTLED"
// 	newBytes, _ := json.Marshal(stub)
// 	return ctx.GetStub().PutState(paymentID, newBytes)
// }

// // GetPrivatePayment returns the full private-payload for a payment.
// // Access is automatically restricted to organisations that are
// // members of the bilateral collection (peers outside the policy
// // will receive a collection-access error).
// func (c *PaymentsContract) GetPrivatePayment(
// 	ctx contractapi.TransactionContextInterface,
// 	paymentID string) (string, error) { // Change []byte to string

// 	// Look up the public stub so we know payer/payee MSPs
// 	stubBytes, err := ctx.GetStub().GetState(paymentID)
// 	if err != nil || stubBytes == nil {
// 		return "", fmt.Errorf("payment %s not found", paymentID)
// 	}

// 	var stub PaymentStub
// 	if err := json.Unmarshal(stubBytes, &stub); err != nil {
// 		return "", err
// 	}

// 	// Derive the bilateral collection name
// 	coll := collectionName(stub.PayerMSP, stub.PayeeMSP)

// 	// Fetch the private payload
// 	payBytes, err := ctx.GetStub().GetPrivateData(coll, paymentID)
// 	if err != nil {
// 		return "", err
// 	}
// 	if payBytes == nil {
// 		return "", fmt.Errorf("no private data in %s for %s", coll, paymentID)
// 	}

// 	return string(payBytes), nil // Convert to string
// }

// // GetAllPrivateData returns all private data in a given collection.
// func (c *PaymentsContract) GetAllPrivateData(ctx contractapi.TransactionContextInterface, collection string) (string, error) {
// 	// Simple range query with empty start/end keys
// 	resultsIterator, err := ctx.GetStub().GetPrivateDataByRange(collection, "", "")
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resultsIterator.Close()

// 	var results []string

// 	for resultsIterator.HasNext() {
// 		queryResult, err := resultsIterator.Next()
// 		if err != nil {
// 			return "", err
// 		}

// 		record := fmt.Sprintf(`{"key":"%s","value":%s}`, queryResult.Key, string(queryResult.Value))
// 		results = append(results, record)
// 	}

// 	// Return JSON array
// 	return fmt.Sprintf("[%s]", strings.Join(results, ",")), nil
// }

// // ReadPayment returns the public stub.
// func (c *PaymentsContract) ReadPayment(ctx contractapi.TransactionContextInterface, paymentID string) (*PaymentStub, error) {
// 	bytes, err := ctx.GetStub().GetState(paymentID)
// 	if err != nil || bytes == nil {
// 		return nil, fmt.Errorf("not found")
// 	}
// 	var stub PaymentStub
// 	_ = json.Unmarshal(bytes, &stub)
// 	return &stub, nil
// }

// func main() {
// 	chaincode, err := contractapi.NewChaincode(&PaymentsContract{})
// 	if err != nil {
// 		panic(err)
// 	}
// 	if err := chaincode.Start(); err != nil {
// 		panic(err)
// 	}
// }