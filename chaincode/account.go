package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing eNaira accounts and payments
type SmartContract struct {
	contractapi.Contract
}

// PaymentDetails holds sensitive payment fields (in Naira, decimals for kobo)
type PaymentDetails struct {
	ID        string  `json:"id"`
	PayerAcct string  `json:"payerAcct"`
	PayeeAcct string  `json:"payeeAcct"`
	Amount    float64 `json:"amount"`   // eNaira, e.g. 1234.56
	Currency  string  `json:"currency"` // "eNaira"
	BVN       string  `json:"bvn"`      // customer BVN
	PayerMSP  string  `json:"payerMSP"`
	PayeeMSP  string  `json:"payeeMSP"`
	Timestamp int64   `json:"timestamp"`
}

// PaymentStub is the public view of a payment
type PaymentStub struct {
	ID        string `json:"id"`
	Hash      string `json:"hash"`
	PayerMSP  string `json:"payerMSP"`
	PayeeMSP  string `json:"payeeMSP"`
	Status    string `json:"status"` // PENDING, SETTLED
	Timestamp int64  `json:"timestamp"`
}

// BankAccount stores on-ledger eNaira token balances per org in each org’s implicit collection
type BankAccount struct {
	MSP     string  `json:"msp"`
	Balance float64 `json:"balance"` // eNaira, decimals for kobo
}

// MintRecord logs eNaira issuance by CentralBankMSP
type MintRecord struct {
	ID        string  `json:"id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	ToMSP     string  `json:"toMsp"`
	Timestamp int64   `json:"timestamp"`
}

// BVNRecord holds basic identity information for the BVN PDC
type BVNRecord struct {
	BVN        string `json:"bvn"`
	Firstname  string `json:"firstname"`
	Lastname   string `json:"lastname"`
	Middlename string `json:"middlename"`
	Gender     string `json:"gender"`
	Phone      string `json:"phone"`
	Birthdate  string `json:"birthdate"` // DD-MM-YYYY
}

// Init ;  Method for initializing smart contract
func (s *SmartContract) Init(ctx contractapi.TransactionContextInterface) error {
	return nil
}

// InitLedger seeds the invoking MSP with 15 billion eNaira and loads all BVN records into the col-BVN PDC.
// Call this in your instantiate/approve step via:
//
//	peer chaincode instantiate ... -c '{"Args":["InitLedger"]}'
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client MSP: %v", err)
	}

	// seed account balance
	starting := 15_000_000_000.0 // 15 billion eNaira
	acct := BankAccount{MSP: clientMSP, Balance: starting}
	acctBytes, err := json.Marshal(acct)
	if err != nil {
		return fmt.Errorf("marshal init account for %s: %v", clientMSP, err)
	}
	acctColl := fmt.Sprintf("_implicit_org_%s", clientMSP)
	if err := ctx.GetStub().PutPrivateData(acctColl, clientMSP, acctBytes); err != nil {
		return fmt.Errorf("init account for %s: %v", clientMSP, err)
	}

	// prepare BVN records
	bvns := []BVNRecord{
		{BVN: "22133455678", Firstname: "Oluwaseun", Lastname: "Adebanjo", Middlename: "Temitope", Gender: "Female", Phone: "08031234567", Birthdate: "15-04-1990"},
		{BVN: "23455677890", Firstname: "Emeka", Lastname: "Okafor", Middlename: "Chukwuemeka", Gender: "Male", Phone: "08134567890", Birthdate: "02-11-1985"},
		{BVN: "24566788901", Firstname: "Chiamaka", Lastname: "Nwankwo", Middlename: "Amarachi", Gender: "Female", Phone: "08046789012", Birthdate: "28-09-1993"},
		{BVN: "25677899012", Firstname: "Ibrahim", Lastname: "Muhammad", Middlename: "Abdulrahman", Gender: "Male", Phone: "08156789023", Birthdate: "10-01-1988"},
		{BVN: "26788900123", Firstname: "Fatima", Lastname: "Ahmad", Middlename: "Sadiku", Gender: "Female", Phone: "08067890123", Birthdate: "05-06-1992"},
		{BVN: "27899011234", Firstname: "Tunde", Lastname: "Oloyede", Middlename: "Oluwatobi", Gender: "Male", Phone: "08178901234", Birthdate: "22-12-1983"},
		{BVN: "28900122345", Firstname: "Amarachi", Lastname: "Eze", Middlename: "Chidera", Gender: "Female", Phone: "08089012345", Birthdate: "17-08-1995"},
		{BVN: "29011233456", Firstname: "Chukwuemeka", Lastname: "Okoro", Middlename: "Obinna", Gender: "Male", Phone: "08190123456", Birthdate: "30-03-1989"},
		{BVN: "30122344567", Firstname: "Aikevbiosa", Lastname: "Okunrola", Middlename: "Oluwasegun", Gender: "Male", Phone: "08091234567", Birthdate: "12-07-1991"},
		{BVN: "31233455678", Firstname: "Bolanle", Lastname: "Soniyi", Middlename: "Omowunmi", Gender: "Female", Phone: "08102345678", Birthdate: "09-10-1987"},
	}

	// load into private data collection
	for _, rec := range bvns {
		recBytes, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal BVN record %s: %v", rec.BVN, err)
		}
		if err := ctx.GetStub().PutPrivateData("col-BVN", rec.BVN, recBytes); err != nil {
			return fmt.Errorf("failed to put BVN record %s: %v", rec.BVN, err)
		}
	}

	return nil
}

// CreateBankAccount initializes a zero-balance account for the given MSP
func (s *SmartContract) CreateBankAccount(ctx contractapi.TransactionContextInterface, msp string) error {
	account := BankAccount{MSP: msp, Balance: 0}
	accJSON, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %v", err)
	}
	coll := fmt.Sprintf("_implicit_org_%s", msp)
	return ctx.GetStub().PutPrivateData(coll, msp, accJSON)
}

// MintTokens issues new eNaira tokens to the specified MSP (only CentralBankMSP allowed)
// func (s *SmartContract) MintTokens(ctx contractapi.TransactionContextInterface, id, toMSP string, amount float64) error {
// 	invoker, err := ctx.GetClientIdentity().GetMSPID()
// 	if err != nil {
// 		return fmt.Errorf("failed to get invoker MSP: %v", err)
// 	}
// 	if invoker != "CentralBankMSP" {
// 		return fmt.Errorf("only CentralBankMSP can mint tokens (invoker: %s)", invoker)
// 	}

// 	// update recipient balance
// 	if err := s.TransferTokens(ctx, "CentralBankMSP", toMSP, amount); err != nil {
// 		return err
// 	}

// 	// record mint event
// 	record := MintRecord{ID: id, Amount: amount, Currency: "eNaira", ToMSP: toMSP, Timestamp: time.Now().UnixMilli()}
// 	recJSON, err := json.Marshal(record)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal mint record: %v", err)
// 	}
// 	return ctx.GetStub().PutState(id, recJSON)
// }

// CreatePayment records a new payment, verifies BVN, transfers tokens, and writes a public stub
func (s *SmartContract) CreatePayment(ctx contractapi.TransactionContextInterface) error {
	// get transient payment details
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

	// verify BVN in central PDC
	bvnBytes, err := ctx.GetStub().GetPrivateData("col-BVN", details.BVN)
	if err != nil || bvnBytes == nil {
		return fmt.Errorf("BVN %s not registered", details.BVN)
	}

	// store full details in bilateral collection
	coll := getCollectionName(details.PayerMSP, details.PayeeMSP)
	if err := ctx.GetStub().PutPrivateData(coll, details.ID, paymentJSON); err != nil {
		return fmt.Errorf("failed to put private payment data: %v", err)
	}

	// transfer tokens from payer to payee
	if err := s.TransferTokens(ctx, details.PayerMSP, details.PayeeMSP, details.Amount); err != nil {
		return err
	}

	// create and store public stub
	hash := computeHash(paymentJSON)
	stub := PaymentStub{ID: details.ID,
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

	// emit event
	if err := ctx.GetStub().SetEvent("PaymentPending", []byte(details.ID)); err != nil {
		return fmt.Errorf("failed to set PaymentPending event: %v", err)
	}

	return nil
}

// GetPrivatePayment retrieves the private payment details given a stub ID
func (s *SmartContract) GetPrivatePayment(ctx contractapi.TransactionContextInterface, id string) (*PaymentDetails, error) {
	// lookup stub to derive collection
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

// TransferTokens debits fromMSP and credits toMSP an amount in their implicit collections
func (s *SmartContract) TransferTokens(ctx contractapi.TransactionContextInterface, fromMSP, toMSP string, amount float64) error {
	// debit payer
	fromColl := fmt.Sprintf("_implicit_org_%s", fromMSP)
	fromBytes, err := ctx.GetStub().GetPrivateData(fromColl, fromMSP)
	if err != nil {
		return fmt.Errorf("failed to read payer account: %v", err)
	}
	var fromAcct BankAccount

	if err := json.Unmarshal(fromBytes, &fromAcct); err != nil {
		return fmt.Errorf("failed to unmarshal payer account: %v", err)
	}

	if fromAcct.Balance < amount {
		return fmt.Errorf("insufficient funds: available %.2f, required %.2f", fromAcct.Balance, amount)
	}
	fromAcct.Balance -= amount
	updated, _ := json.Marshal(fromAcct)
	if err := ctx.GetStub().PutPrivateData(fromColl, fromMSP, updated); err != nil {
		return fmt.Errorf("failed to update payer account: %v", err)
	}

	// credit payee
	toColl := fmt.Sprintf("_implicit_org_%s", toMSP)
	toBytes, err := ctx.GetStub().GetPrivateData(toColl, toMSP)
	var toAcct BankAccount
	if err != nil {
		return fmt.Errorf("failed to read payee account: %v", err)
	}

	if err := json.Unmarshal(toBytes, &toAcct); err != nil {
		return fmt.Errorf("failed to unmarshal payee account: %v", err)
	}

	toAcct.Balance += amount
	cred, _ := json.Marshal(toAcct)
	if err := ctx.GetStub().PutPrivateData(toColl, toMSP, cred); err != nil {
		return fmt.Errorf("failed to update payee account: %v", err)
	}

	return nil
}

// SettlePayment marks a payment stub as SETTLED and emits a settlement event
func (s *SmartContract) SettlePayment(ctx contractapi.TransactionContextInterface, id string) error {
	stubBytes, err := ctx.GetStub().GetState(id)
	if err != nil || stubBytes == nil {
		return fmt.Errorf("payment stub %s not found", id)
	}
	var stub PaymentStub
	if err := json.Unmarshal(stubBytes, &stub); err != nil {
		return fmt.Errorf("failed to unmarshal stub: %v", err)
	}
	stub.Status = "SETTLED"
	stub.Timestamp = time.Now().UnixMilli()
	updated, _ := json.Marshal(stub)
	if err := ctx.GetStub().PutState(id, updated); err != nil {
		return fmt.Errorf("failed to update stub status: %v", err)
	}
	if err := ctx.GetStub().SetEvent("PaymentSettled", []byte(id)); err != nil {
		return fmt.Errorf("failed to set PaymentSettled event: %v", err)
	}
	return nil
}

// ProcessLSM settles all pending payments; can be extended with netting logic
// func (s *SmartContract) ProcessLSM(ctx contractapi.TransactionContextInterface) error {
// 	iter, err := ctx.GetStub().GetStateByRange("", "")
// 	if err != nil {
// 		return fmt.Errorf("failed to get state by range: %v", err)
// 	}
// 	defer iter.Close()

// 	for iter.HasNext() {
// 		res, err := iter.Next()
// 		if err != nil {
// 			return err
// 		}
// 		var stub PaymentStub
// 		if err := json.Unmarshal(res.Value, &stub); err != nil {
// 			continue
// 		}
// 		if stub.Status == "PENDING" {
// 			if err := s.SettlePayment(ctx, stub.ID); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

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

func main() {
	chaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		panic(fmt.Sprintf("Error creating chaincode: %v", err))
	}
	if err := chaincode.Start(); err != nil {
		panic(fmt.Sprintf("Error starting chaincode: %v", err))
	}
}
