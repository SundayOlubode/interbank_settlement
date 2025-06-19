package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

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

var authorizedMSPs = []string{
	"AccessBankMSP",
	"GTBankMSP",
	"ZenithBankMSP",
	"FirstBankMSP",
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

// Get all queued transactions for the calling MSP with all other MSPs
func (s *SmartContract) GetAllQueuedTransactions(ctx contractapi.TransactionContextInterface) (*AllQueuedSummary, error) {
	// Get the caller's MSP
	clientIdentity := ctx.GetClientIdentity()
	callerMSP, err := clientIdentity.GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("failed to get caller MSP ID: %v", err)
	}

	// Validate caller is authorized
	if !s.isAuthorizedMSP(callerMSP) {
		return nil, fmt.Errorf("unauthorized MSP: %s", callerMSP)
	}

	var bilateralSummaries []QueuedTransactionSummary
	var grandTotalAmount float64
	var grandTotalCount int

	// Iterate through all other MSPs
	for _, otherMSP := range authorizedMSPs {
		if otherMSP == callerMSP {
			continue // Skip self
		}

		// Get the bilateral collection name
		collectionName := getCollectionName(callerMSP, otherMSP)

		// Get queued transactions for this MSP pair
		queuedCount, queuedAmount, err := s.getQueuedTransactionsFromCollection(ctx, collectionName)
		if err != nil {
			// Log error but continue with other MSPs
			fmt.Printf("Error getting queued transactions from collection %s: %v\n", collectionName, err)
			continue
		}

		// Add to bilateral summary
		summary := QueuedTransactionSummary{
			MSP1:              callerMSP,
			MSP2:              otherMSP,
			CollectionName:    collectionName,
			QueuedCount:       queuedCount,
			QueuedTotalAmount: queuedAmount,
		}

		bilateralSummaries = append(bilateralSummaries, summary)

		// Add to grand totals
		grandTotalAmount += queuedAmount
		grandTotalCount += queuedCount
	}

	return &AllQueuedSummary{
		CallerMSP:          callerMSP,
		BilateralSummaries: bilateralSummaries,
		GrandTotalAmount:   grandTotalAmount,
		GrandTotalCount:    grandTotalCount,
	}, nil
}

// Helper function to get queued transactions from a specific collection
func (s *SmartContract) getQueuedTransactionsFromCollection(ctx contractapi.TransactionContextInterface, collectionName string) (int, float64, error) {
	// Get all records from the private data collection
	resultsIterator, err := ctx.GetStub().GetPrivateDataByRange(collectionName, "", "")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get private data from collection %s: %v", collectionName, err)
	}
	defer resultsIterator.Close()

	var queuedCount int
	var queuedTotalAmount float64

	// Iterate through all records in the collection
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return 0, 0, fmt.Errorf("failed to get next result: %v", err)
		}

		var payment PaymentDetails
		err = json.Unmarshal(queryResponse.Value, &payment)
		if err != nil {
			// Log error but continue processing
			fmt.Printf("Error unmarshaling payment record %s: %v\n", queryResponse.Key, err)
			continue
		}

		// Check if status is QUEUED
		if payment.Status == "QUEUED" {
			queuedCount++
			queuedTotalAmount += payment.Amount
		}
	}

	return queuedCount, queuedTotalAmount, nil
}

// Get queued transactions for a specific MSP pair
func (s *SmartContract) GetQueuedTransactionsForMSPPair(ctx contractapi.TransactionContextInterface, otherMSP string) (*QueuedTransactionSummary, error) {
	// Get the caller's MSP
	clientIdentity := ctx.GetClientIdentity()
	callerMSP, err := clientIdentity.GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("failed to get caller MSP ID: %v", err)
	}

	// Validate both MSPs are authorized
	if !s.isAuthorizedMSP(callerMSP) {
		return nil, fmt.Errorf("unauthorized caller MSP: %s", callerMSP)
	}
	if !s.isAuthorizedMSP(otherMSP) {
		return nil, fmt.Errorf("unauthorized target MSP: %s", otherMSP)
	}

	// Get the bilateral collection name
	collectionName := getCollectionName(callerMSP, otherMSP)

	// Get queued transactions
	queuedCount, queuedAmount, err := s.getQueuedTransactionsFromCollection(ctx, collectionName)
	if err != nil {
		return nil, err
	}

	return &QueuedTransactionSummary{
		MSP1:              callerMSP,
		MSP2:              otherMSP,
		CollectionName:    collectionName,
		QueuedCount:       queuedCount,
		QueuedTotalAmount: queuedAmount,
	}, nil
}

// Get detailed queued transactions from a specific collection
func (s *SmartContract) GetQueuedTransactionDetails(ctx contractapi.TransactionContextInterface, otherMSP string) ([]*PaymentDetails, error) {
	// Get the caller's MSP
	clientIdentity := ctx.GetClientIdentity()
	callerMSP, err := clientIdentity.GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("failed to get caller MSP ID: %v", err)
	}

	// Validate both MSPs are authorized
	if !s.isAuthorizedMSP(callerMSP) {
		return nil, fmt.Errorf("unauthorized caller MSP: %s", callerMSP)
	}
	if !s.isAuthorizedMSP(otherMSP) {
		return nil, fmt.Errorf("unauthorized target MSP: %s", otherMSP)
	}

	// Get the bilateral collection name
	collectionName := getCollectionName(callerMSP, otherMSP)

	// Get all records from the private data collection
	resultsIterator, err := ctx.GetStub().GetPrivateDataByRange(collectionName, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get private data from collection %s: %v", collectionName, err)
	}
	defer resultsIterator.Close()

	var queuedTransactions []*PaymentDetails

	// Iterate through all records in the collection
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next result: %v", err)
		}

		var payment PaymentDetails
		err = json.Unmarshal(queryResponse.Value, &payment)
		if err != nil {
			// Log error but continue processing
			fmt.Printf("Error unmarshaling payment record %s: %v\n", queryResponse.Key, err)
			continue
		}

		// Only include QUEUED transactions
		if payment.Status == "QUEUED" {
			queuedTransactions = append(queuedTransactions, &payment)
		}
	}

	return queuedTransactions, nil
}

func (s *SmartContract) GetBankAccountBalance(ctx contractapi.TransactionContextInterface) (*BankAccount, error) {
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("failed to get client MSP: %v", err)
	}

	coll := fmt.Sprintf("col-settlement-%s", clientMSP)
	accountBytes, err := ctx.GetStub().GetPrivateData(coll, clientMSP)
	if err != nil {
		return nil, fmt.Errorf("failed to get account data for %s: %v", clientMSP, err)
	}
	if accountBytes == nil {
		return nil, fmt.Errorf("no account found for %s", clientMSP)
	}

	var account BankAccount
	if err := json.Unmarshal(accountBytes, &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account data: %v", err)
	}

	return &account, nil
}

func (s *SmartContract) GetBankAccountBalanceByMSP(ctx contractapi.TransactionContextInterface, msp string) (*BankAccount, error) {
	if msp == "" {
		return nil, fmt.Errorf("please provide target bank MSP")
	}

	coll := fmt.Sprintf("col-settlement-%s", msp)
	accountBytes, err := ctx.GetStub().GetPrivateData(coll, msp)
	if err != nil {
		return nil, fmt.Errorf("failed to get account data for %s: %v", msp, err)
	}
	if accountBytes == nil {
		return nil, fmt.Errorf("no account found for %s", msp)
	}

	var account BankAccount
	if err := json.Unmarshal(accountBytes, &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account data: %v", err)
	}

	return &account, nil
}

// GetBankingOverview returns combined bank account balance and queued transaction summary
func (s *SmartContract) GetBankingOverview(ctx contractapi.TransactionContextInterface) (*CombinedBankingData, error) {
	// Get bank account balance
	bankAccount, err := s.GetBankAccountBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bank account balance: %v", err)
	}

	// Get all queued transactions summary
	queuedSummary, err := s.GetAllQueuedTransactions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get queued transactions: %v", err)
	}

	return &CombinedBankingData{
		BankAccount:   bankAccount,
		QueuedSummary: queuedSummary,
	}, nil
}

// GetAllTransactionAnalytics extracts all transactions from all PDCs and aggregates by status
func (s *SmartContract) GetAllTransactionAnalytics(ctx contractapi.TransactionContextInterface) (*TransactionAnalytics, error) {
	// Get the caller's MSP
	clientIdentity := ctx.GetClientIdentity()
	callerMSP, err := clientIdentity.GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("failed to get caller MSP ID: %v", err)
	}

	// Validate caller is authorized
	if !s.isAuthorizedMSP(callerMSP) {
		return nil, fmt.Errorf("unauthorized MSP: %s", callerMSP)
	}

	// Initialize analytics
	analytics := &TransactionAnalytics{
		Completed: TransactionStats{Count: 0, Volume: 0.0},
		Queued:    TransactionStats{Count: 0, Volume: 0.0},
		Pending:   TransactionStats{Count: 0, Volume: 0.0},
	}

	// Get all possible PDC collection names for the caller
	collectionNames := s.getAllBilateralCollections(callerMSP)

	// Process each collection
	for _, collectionName := range collectionNames {
		err := s.processCollectionTransactions(ctx, collectionName, analytics)
		if err != nil {
			// Log error but continue with other collections
			fmt.Printf("Error processing collection %s: %v\n", collectionName, err)
			continue
		}
	}

	return analytics, nil
}

// Helper function to get all bilateral collection names for a given MSP
func (s *SmartContract) getAllBilateralCollections(callerMSP string) []string {
	var collections []string

	for _, otherMSP := range authorizedMSPs {
		if otherMSP == callerMSP {
			continue // Skip self
		}

		// Generate bilateral collection name
		collectionName := getCollectionName(callerMSP, otherMSP)
		collections = append(collections, collectionName)
	}

	return collections
}

// Helper function to process transactions from a single collection
func (s *SmartContract) processCollectionTransactions(ctx contractapi.TransactionContextInterface, collectionName string, analytics *TransactionAnalytics) error {
	// Get all records from the private data collection
	resultsIterator, err := ctx.GetStub().GetPrivateDataByRange(collectionName, "", "")
	if err != nil {
		return fmt.Errorf("failed to get private data from collection %s: %v", collectionName, err)
	}
	defer resultsIterator.Close()

	// Process each transaction record
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return fmt.Errorf("failed to get next result: %v", err)
		}

		var payment PaymentDetails
		err = json.Unmarshal(queryResponse.Value, &payment)
		if err != nil {
			// Log error but continue processing other records
			fmt.Printf("Error unmarshaling payment record %s in collection %s: %v\n",
				queryResponse.Key, collectionName, err)
			continue
		}

		// Aggregate based on status
		switch strings.ToUpper(payment.Status) {
		case "COMPLETED", "SETTLED":
			analytics.Completed.Count++
			analytics.Completed.Volume += payment.Amount
		case "QUEUED":
			analytics.Queued.Count++
			analytics.Queued.Volume += payment.Amount
		case "PENDING":
			analytics.Pending.Count++
			analytics.Pending.Volume += payment.Amount
		default:
			// Log unknown status but don't fail
			fmt.Printf("Unknown transaction status %s for transaction %s\n",
				payment.Status, payment.ID)
		}
	}

	return nil
}

func (s *SmartContract) GetAllBankTransactions(ctx contractapi.TransactionContextInterface) ([]*PaymentDetails, error) {
	// Get the caller's MSP
	clientIdentity := ctx.GetClientIdentity()
	callerMSP, err := clientIdentity.GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("failed to get caller MSP ID: %v", err)
	}

	// Validate caller is authorized
	if !s.isAuthorizedMSP(callerMSP) {
		return nil, fmt.Errorf("unauthorized MSP: %s", callerMSP)
	}

	var allTransactions []*PaymentDetails

	// Get all possible PDC collection names for the caller
	collectionNames := s.getAllBilateralCollections(callerMSP)

	// Process each collection
	for _, collectionName := range collectionNames {
		resultsIterator, err := ctx.GetStub().GetPrivateDataByRange(collectionName, "", "")
		if err != nil {
			return nil, fmt.Errorf("failed to get private data from collection %s: %v", collectionName, err)
		}
		defer resultsIterator.Close()

		for resultsIterator.HasNext() {
			queryResponse, err := resultsIterator.Next()
			if err != nil {
				return nil, fmt.Errorf("failed to get next result: %v", err)
			}

			var payment PaymentDetails
			err = json.Unmarshal(queryResponse.Value, &payment)
			if err != nil {
				fmt.Printf("Error unmarshaling payment record %s in collection %s: %v\n",
					queryResponse.Key, collectionName, err)
				continue
			}

			allTransactions = append(allTransactions, &payment)
		}
	}

	return allTransactions, nil
}
