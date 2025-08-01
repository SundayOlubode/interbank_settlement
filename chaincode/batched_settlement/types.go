// types.go
package settlement

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
	ID             string   `json:"id"`
	PayerAcct      string   `json:"payerAcct"`
	PayeeAcct      string   `json:"payeeAcct"`
	Amount         float64  `json:"amount"`         // original transaction value
	AmountToSettle float64  `json:"amountToSettle"` // remaining for settlement/netting
	Currency       string   `json:"currency"`
	BVN            string   `json:"bvn"`
	PayerMSP       string   `json:"payerMSP"`
	PayeeMSP       string   `json:"payeeMSP"`
	Status         string   `json:"status"` // PENDING, ACKNOWLEDGED, BATCHED, QUEUED, DEBITED, SETTLED...
	Timestamp      int64    `json:"timestamp"`
	BatchWindow    int64    `json:"batchWindow"` // Which 2-minute window this payment belongs to
	User           BankUser `json:"user"`
}

// PaymentEventDetails for events
type PaymentEventDetails struct {
	ID          string `json:"id"`
	PayeeMSP    string `json:"payeeMSP"`
	PayerMSP    string `json:"payerMSP"`
	BatchWindow int64  `json:"batchWindow,omitempty"`
}

// PaymentStub is the public view of a payment
type PaymentStub struct {
	ID          string `json:"id"`
	Hash        string `json:"hash"`
	PayerMSP    string `json:"payerMSP"`
	PayeeMSP    string `json:"payeeMSP"`
	Status      string `json:"status"` // PENDING, ACKNOWLEDGED, BATCHED, SETTLED, QUEUED
	Timestamp   int64  `json:"timestamp"`
	BatchWindow int64  `json:"batchWindow"` // Which 2-minute window this payment belongs to
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
	Batched   TransactionStats `json:"batched"` // New status for batch processing
}

// TransactionStats holds count and volume for each status
type TransactionStats struct {
	Count  int     `json:"count"`
	Volume float64 `json:"volume"`
}

// CounterpartyStats holds the summary for one counterparty MSP.
type CounterpartyStats struct {
	BankMSP           string  `json:"bankMSP"`
	TransactionCount  int     `json:"transactionCount"`
	TransactionVolume float64 `json:"transactionVolume"`
	NetPosition       float64 `json:"netPosition"`
}

// QueuedTransactionSummary holds the summary for each MSP pair
type QueuedTransactionSummary struct {
	MSP1              string  `json:"msp1"`
	MSP2              string  `json:"msp2"`
	CollectionName    string  `json:"collectionName"`
	QueuedCount       int     `json:"queuedCount"`
	QueuedTotalAmount float64 `json:"queuedTotalAmount"`
}

// AllQueuedSummary holds the complete summary
type AllQueuedSummary struct {
	CallerMSP          string                     `json:"callerMSP"`
	BilateralSummaries []QueuedTransactionSummary `json:"bilateralSummaries"`
	GrandTotalAmount   float64                    `json:"grandTotalAmount"`
	GrandTotalCount    int                        `json:"grandTotalCount"`
}

type TransactionHistoryEntry struct {
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	PayeeMSP    string  `json:"payeeMSP"`
	PayerAcct   string  `json:"payerAcct"`
	PayerMSP    string  `json:"payerMSP"`
	PaymentId   string  `json:"paymentId"`
	SettledAt   string  `json:"settledAt"` // nil unless SETTLED
	Status      string  `json:"status"`
	Timestamp   string  `json:"timestamp"`   // RFC3339
	BatchWindow int64   `json:"batchWindow"` // Which batch window
}

// OffsetUpdate describes how a single PaymentDetails record should change.
type OffsetUpdate struct {
	ID             string  `json:"id"`
	AmountToSettle float64 `json:"amountToSettle"`
	Status         string  `json:"status"`
}

// OffsetCalculation - full payload to client
type OffsetCalculation struct {
	Offset  float64        `json:"offset"`
	Updates []OffsetUpdate `json:"updates"`
}

// MultiOffsetUpdate describes how a single queued payment should change.
type MultiOffsetUpdate struct {
	ID             string  `json:"id"`
	PayerMSP       string  `json:"payerMSP"`
	PayeeMSP       string  `json:"payeeMSP"`
	AmountToSettle float64 `json:"amountToSettle"`
	Status         string  `json:"status"`
}

// MultiOffsetCalculation - the full payload to client
type MultiOffsetCalculation struct {
	// Net position per bank: positive = owed money; negative = owes money
	NetPositions map[string]float64 `json:"netPositions"`
	// Exactly which rows in which PDCs to update
	Updates []MultiOffsetUpdate `json:"updates"`
}

// BatchSettlementRequest represents a batch settlement operation
type BatchSettlementRequest struct {
	BatchWindow int64    `json:"batchWindow"`
	PaymentIDs  []string `json:"paymentIds"`
	RequestedBy string   `json:"requestedBy"`
	Timestamp   int64    `json:"timestamp"`
}

// BatchSettlementResult represents the result of a batch settlement
type BatchSettlementResult struct {
	BatchWindow     int64           `json:"batchWindow"`
	ProcessedCount  int             `json:"processedCount"`
	SuccessfulCount int             `json:"successfulCount"`
	QueuedCount     int             `json:"queuedCount"`
	TotalAmount     float64         `json:"totalAmount"`
	Results         []PaymentResult `json:"results"`
	Timestamp       int64           `json:"timestamp"`
}

// PaymentResult represents the result of processing a single payment
type PaymentResult struct {
	PaymentID string `json:"paymentId"`
	Status    string `json:"status"` // SUCCESS, QUEUED, ERROR
	Message   string `json:"message,omitempty"`
}

// SettlementWindow represents a 2-minute settlement window
type SettlementWindow struct {
	WindowID     int64  `json:"windowId"`
	StartTime    int64  `json:"startTime"`
	EndTime      int64  `json:"endTime"`
	Status       string `json:"status"` // COLLECTING, PROCESSING, COMPLETED
	PaymentCount int    `json:"paymentCount"`
}

// BatchProcessingEvent represents events emitted during batch processing
type BatchProcessingEvent struct {
	EventType   string `json:"eventType"` // BATCH_STARTED, BATCH_COMPLETED, PAYMENT_PROCESSED
	BatchWindow int64  `json:"batchWindow"`
	MSP         string `json:"msp"`
	PaymentID   string `json:"paymentId,omitempty"`
	Status      string `json:"status,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}

// NettingSettlementResult represents the result of netting-based settlement
type NettingSettlementResult struct {
	TotalPayments   int                    `json:"totalPayments"`
	SettledPayments int                    `json:"settledPayments"`
	NetPositions    map[string]float64     `json:"netPositions"`
	SettledBanks    map[string]float64     `json:"settledBanks"`
	FailedBanks     []FailedBankSettlement `json:"failedBanks"`
	TotalNetAmount  float64                `json:"totalNetAmount"`
	Timestamp       int64                  `json:"timestamp"`
}

// FailedBankSettlement represents a bank that failed during net settlement
type FailedBankSettlement struct {
	BankMSP   string  `json:"bankMSP"`
	NetAmount float64 `json:"netAmount"`
	Error     string  `json:"error"`
}

// SettlementStatistics represents system-wide settlement statistics
type SettlementStatistics struct {
	TotalPayments int                `json:"totalPayments"`
	TotalAmount   float64            `json:"totalAmount"`
	StatusCounts  map[string]int     `json:"statusCounts"`
	StatusAmounts map[string]float64 `json:"statusAmounts"`
	BankBalances  map[string]float64 `json:"bankBalances"`
	LastUpdated   int64              `json:"lastUpdated"`
}

// NettingCalculationResult represents the calculated netting offsets
type NettingCalculationResult struct {
	NetPositions   map[string]float64 `json:"netPositions"`
	PaymentUpdates []PaymentUpdate    `json:"paymentUpdates"`
	TotalPayments  int                `json:"totalPayments"`
	TotalNetAmount float64            `json:"totalNetAmount"`
	Timestamp      int64              `json:"timestamp"`
}

// PaymentUpdate represents a payment status update
type PaymentUpdate struct {
	ID             string  `json:"id"`
	PayerMSP       string  `json:"payerMSP"`
	PayeeMSP       string  `json:"payeeMSP"`
	Status         string  `json:"status"`
	AmountToSettle float64 `json:"amountToSettle"`
}

// NettingApplicationResult represents the result of applying netting offsets
type NettingApplicationResult struct {
	SettledBanks    map[string]float64     `json:"settledBanks"`
	FailedBanks     []FailedBankSettlement `json:"failedBanks"`
	SettledPayments int                    `json:"settledPayments"`
	FailedPayments  int                    `json:"failedPayments"`
	TotalNetAmount  float64                `json:"totalNetAmount"`
	Timestamp       int64                  `json:"timestamp"`
}
