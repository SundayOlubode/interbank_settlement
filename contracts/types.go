package main

// BankUser represents a bank customer
type BankUser struct {
	BVN       string `json:"bvn"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Birthdate string `json:"birthdate"` // DD-MM-YYYY
	Gender    string `json:"gender"`
}

// PaymentDetails holds sensitive payment fields (in Naira, decimals for kobo)
type PaymentDetails struct {
	ID        string   `json:"id"`
	PayerAcct string   `json:"payerAcct"`
	PayeeAcct string   `json:"payeeAcct"`
	Amount    float64  `json:"amount"`   // eNaira, e.g. 1234.56
	Currency  string   `json:"currency"` // "eNaira"
	BVN       string   `json:"bvn"`      // customer BVN
	PayerMSP  string   `json:"payerMSP"`
	PayeeMSP  string   `json:"payeeMSP"`
	Status    string   `json:"status"` // PENDING, SETTLED, QUEUED
	Timestamp int64    `json:"timestamp"`
	User      BankUser `json:"user"` // optional, can be used to store user details
}

// PaymentEventDetails for events
type PaymentEventDetails struct {
	ID       string `json:"id"`
	PayeeMSP string `json:"payeeMSP"`
	PayerMSP string `json:"payerMSP"`
}

// PaymentStub is the public view of a payment
type PaymentStub struct {
	ID        string `json:"id"`
	Hash      string `json:"hash"`
	PayerMSP  string `json:"payerMSP"`
	PayeeMSP  string `json:"payeeMSP"`
	Status    string `json:"status"` // PENDING, SETTLED, QUEUED
	Timestamp int64  `json:"timestamp"`
}

// BankAccount stores on-ledger eNaira token balances per org in each org's implicit collection
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

// PaymentSummary represents a summary of payments between two MSPs
type PaymentSummary struct {
	MSP1           string             `json:"msp1"`
	MSP2           string             `json:"msp2"`
	CollectionName string             `json:"collectionName"`
	TotalPayments  int                `json:"totalPayments"`
	TotalAmount    float64            `json:"totalAmount"`
	StatusCounts   map[string]int     `json:"statusCounts"`
	TotalAmounts   map[string]float64 `json:"totalAmounts"`
}

// CombinedBankingData holds both balance and queued transaction information
type CombinedBankingData struct {
	BankAccount   *BankAccount      `json:"bankAccount"`
	QueuedSummary *AllQueuedSummary `json:"queuedSummary"`
}

// TransactionAnalytics holds the aggregated transaction data
type TransactionAnalytics struct {
    Completed TransactionStats `json:"completed"`
    Queued    TransactionStats `json:"queued"`
    Pending   TransactionStats `json:"pending"`
}

// TransactionStats holds count and volume for each status
type TransactionStats struct {
    Count  int     `json:"count"`
    Volume float64 `json:"volume"`
}