package chaincode_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/SundayOlubode/interbank_settlement/chaincode/settlement"
	"github.com/SundayOlubode/interbank_settlement/chaincode/settlement/mocks"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MOCK STATE QUERY ITERATOR FOR MULTILATERAL TESTS
// =============================================================================

// MockMultilateralStateQueryIterator implements the StateQueryIteratorInterface
type MockMultilateralStateQueryIterator struct {
	mock.Mock
}

func (m *MockMultilateralStateQueryIterator) HasNext() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockMultilateralStateQueryIterator) Next() (*queryresult.KV, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*queryresult.KV), args.Error(1)
}

func (m *MockMultilateralStateQueryIterator) Close() error {
	args := m.Called()
	return args.Error(0)
}

// =============================================================================
// CONSTANTS AND HELPER FUNCTIONS
// =============================================================================

// Helper to setup comprehensive mocking for all authorized MSPs
func setupComprehensiveMocking(chaincodeStub *mocks.ChaincodeStubInterface, paymentsByCollection map[string][]settlement.PaymentDetails) {
	// All possible MSPs that might be in authorizedMSPs array
	allMSPs := []string{bankAMSP, bankBMSP, bankCMSP, bankDMSP, bankEMSP}

	// Mock every possible collection combination
	for _, a := range allMSPs {
		for _, b := range allMSPs {
			if a == b {
				continue
			}
			collectionName := getCollectionName(a, b)

			if payments, exists := paymentsByCollection[collectionName]; exists && len(payments) > 0 {
				// Collection with payments
				iterator := &MockMultilateralStateQueryIterator{}

				// Setup calls for this specific collection
				for i := 0; i < len(payments); i++ {
					iterator.On("HasNext").Return(true).Once()
				}
				iterator.On("HasNext").Return(false).Once()

				for _, payment := range payments {
					paymentJSON, _ := json.Marshal(payment)
					kv := &queryresult.KV{
						Key:   payment.ID,
						Value: paymentJSON,
					}
					iterator.On("Next").Return(kv, nil).Once()
				}

				iterator.On("Close").Return(nil).Once()
				chaincodeStub.On("GetPrivateDataByRange", collectionName, "", "").Return(iterator, nil).Once()
			} else {
				// Empty collection
				iterator := &MockMultilateralStateQueryIterator{}
				iterator.On("HasNext").Return(false).Once()
				iterator.On("Close").Return(nil).Once()
				chaincodeStub.On("GetPrivateDataByRange", collectionName, "", "").Return(iterator, nil).Once()
			}
		}
	}
}

// =============================================================================
// TEST CASES
// =============================================================================

func TestCalculateMultilateralOffset_Success_SimpleScenario(t *testing.T) {
	t.Log("✓ Simple Multilateral Offset Calculation")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Create a simple circular payment scenario:
	// AccessBank owes GTBank 1000
	// GTBank owes Zenith 800
	// Zenith owes AccessBank 600
	paymentsByCollection := map[string][]settlement.PaymentDetails{
		getCollectionName(bankAMSP, bankBMSP): {
			createQueuedPayment("pay1", bankAMSP, bankBMSP, 1000.0),
		},
		getCollectionName(bankBMSP, bankCMSP): {
			createQueuedPayment("pay2", bankBMSP, bankCMSP, 800.0),
		},
		getCollectionName(bankCMSP, bankAMSP): {
			createQueuedPayment("pay3", bankCMSP, bankAMSP, 600.0),
		},
	}

	setupComprehensiveMocking(chaincodeStub, paymentsByCollection)

	// Execute
	result, err := smartContract.CalculateMultilateralOffset(transactionContext)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)

	// Expected net positions (if each collection processed once):
	// AccessBank: receives 600 from Zenith, pays 1000 to GTBank = -400
	// GTBank: receives 1000 from AccessBank, pays 800 to Zenith = +200
	// Zenith: receives 800 from GTBank, pays 600 to AccessBank = +200
	expectedNetPositions := map[string]float64{
		bankAMSP: -400.0, // AccessBank net debtor
		bankBMSP: 200.0,  // GTBank net creditor
		bankCMSP: 200.0,  // Zenith net creditor
	}

	// Note: This test will fail until the algorithm is fixed to process each collection once
	// Current algorithm processes each collection twice due to (a,b) and (b,a) iteration
	// Actual results will be: AccessBank: -800, GTBank: +400, Zenith: +400
	require.Equal(t, expectedNetPositions, result.NetPositions)
	require.Len(t, result.Updates, 3) // Should be 3 payments, each processed once

	// Verify all updates mark payments as SETTLED with zero AmountToSettle
	paymentIDs := []string{"pay1", "pay2", "pay3"}
	for _, update := range result.Updates {
		require.Contains(t, paymentIDs, update.ID)
		require.Equal(t, 0.0, update.AmountToSettle)
		require.Equal(t, "SETTLED", update.Status)
	}
}

func TestCalculateMultilateralOffset_Success_ComplexScenario(t *testing.T) {
	t.Log("✓ Complex Multilateral Offset with Multiple Payments")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Create a complex scenario with payments across different collections:
	// Collection 1: AccessBank ↔ GTBank
	//   - AccessBank → GTBank: 500 + 300 = 800
	//   - GTBank → AccessBank: 400
	// Collection 2: GTBank → Zenith: 600
	// Collection 3: Zenith → FirstBank: 200
	// Collection 4: FirstBank → AccessBank: 150
	paymentsByCollection := map[string][]settlement.PaymentDetails{
		// All AccessBank ↔ GTBank payments in one collection (alphabetically sorted)
		getCollectionName(bankAMSP, bankBMSP): {
			createQueuedPayment("pay1", bankAMSP, bankBMSP, 500.0), // AccessBank → GTBank
			createQueuedPayment("pay2", bankAMSP, bankBMSP, 300.0), // AccessBank → GTBank
			createQueuedPayment("pay3", bankBMSP, bankAMSP, 400.0), // GTBank → AccessBank
		},
		// GTBank → Zenith payments
		getCollectionName(bankBMSP, bankCMSP): {
			createQueuedPayment("pay4", bankBMSP, bankCMSP, 600.0),
		},
		// Zenith → FirstBank payments
		getCollectionName(bankCMSP, bankDMSP): {
			createQueuedPayment("pay5", bankCMSP, bankDMSP, 200.0),
		},
		// FirstBank → AccessBank payments
		getCollectionName(bankDMSP, bankAMSP): {
			createQueuedPayment("pay6", bankDMSP, bankAMSP, 150.0),
		},
	}

	setupComprehensiveMocking(chaincodeStub, paymentsByCollection)

	// Execute
	result, err := smartContract.CalculateMultilateralOffset(transactionContext)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)

	// Expected net positions:
	// AccessBank: pays 800 to GTBank, receives 400 from GTBank + 150 from FirstBank = -250
	// GTBank: receives 800 from AccessBank, pays 400 to AccessBank + 600 to Zenith = -200
	// Zenith: receives 600 from GTBank, pays 200 to FirstBank = +400
	// FirstBank: receives 200 from Zenith, pays 150 to AccessBank = +50
	expectedNetPositions := map[string]float64{
		bankAMSP: -250.0, // AccessBank: -800 + 400 + 150 = -250
		bankBMSP: -200.0, // GTBank: +800 - 400 - 600 = -200
		bankCMSP: 400.0,  // Zenith: +600 - 200 = +400
		bankDMSP: 50.0,   // FirstBank: +200 - 150 = +50
	}

	require.Equal(t, expectedNetPositions, result.NetPositions)
	require.Len(t, result.Updates, 6) // Should be 6 payments, each processed once

	// Verify all updates mark payments as SETTLED with zero AmountToSettle
	paymentIDs := []string{"pay1", "pay2", "pay3", "pay4", "pay5", "pay6"}
	for _, update := range result.Updates {
		require.Contains(t, paymentIDs, update.ID)
		require.Equal(t, 0.0, update.AmountToSettle)
		require.Equal(t, "SETTLED", update.Status)
	}
}

func TestCalculateMultilateralOffset_Success_NoQueuedPayments(t *testing.T) {
	t.Log("✓ No Queued Payments Results in Empty Calculation")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// All collections are empty (no payments at all)
	paymentsByCollection := map[string][]settlement.PaymentDetails{}

	setupComprehensiveMocking(chaincodeStub, paymentsByCollection)

	// Execute
	result, err := smartContract.CalculateMultilateralOffset(transactionContext)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.NetPositions)
	require.Empty(t, result.Updates)
}

func TestCalculateMultilateralOffset_Success_NonQueuedPaymentsIgnored(t *testing.T) {
	t.Log("✓ Non-Queued Payments Are Ignored")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Create payments with different statuses - only QUEUED should be processed
	paymentsByCollection := map[string][]settlement.PaymentDetails{
		getCollectionName(bankAMSP, bankBMSP): {
			// These should be ignored (non-QUEUED status)
			{ID: "pay1", PayerMSP: bankAMSP, PayeeMSP: bankBMSP, AmountToSettle: 1000.0, Status: "PENDING", Currency: "NGN"},
			{ID: "pay2", PayerMSP: bankAMSP, PayeeMSP: bankBMSP, AmountToSettle: 500.0, Status: "SETTLED", Currency: "NGN"},
			{ID: "pay3", PayerMSP: bankAMSP, PayeeMSP: bankBMSP, AmountToSettle: 200.0, Status: "FAILED", Currency: "NGN"},

			// Only this should be processed (QUEUED status)
			createQueuedPayment("pay4", bankAMSP, bankBMSP, 300.0),
		},
		getCollectionName(bankBMSP, bankCMSP): {
			// This should also be processed (QUEUED status)
			createQueuedPayment("pay5", bankBMSP, bankCMSP, 150.0),
		},
	}

	setupComprehensiveMocking(chaincodeStub, paymentsByCollection)

	// Execute
	result, err := smartContract.CalculateMultilateralOffset(transactionContext)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)

	// Only the QUEUED payments should affect net positions
	// AccessBank pays 300 to GTBank, GTBank pays 150 to Zenith
	expectedNetPositions := map[string]float64{
		bankAMSP: -300.0, // AccessBank pays 300 to GTBank
		bankBMSP: 150.0,  // GTBank receives 300 from AccessBank, pays 150 to Zenith
		bankCMSP: 150.0,  // Zenith receives 150 from GTBank
	}

	require.Equal(t, expectedNetPositions, result.NetPositions)
	require.Len(t, result.Updates, 2) // Only pay4 and pay5 should be updated

	// Verify only QUEUED payments are in updates
	updateIDs := make([]string, len(result.Updates))
	for i, update := range result.Updates {
		updateIDs[i] = update.ID
		require.Equal(t, 0.0, update.AmountToSettle)
		require.Equal(t, "SETTLED", update.Status)
	}
	require.Contains(t, updateIDs, "pay4")
	require.Contains(t, updateIDs, "pay5")
	require.NotContains(t, updateIDs, "pay1") // PENDING - should be ignored
	require.NotContains(t, updateIDs, "pay2") // SETTLED - should be ignored
	require.NotContains(t, updateIDs, "pay3") // FAILED - should be ignored
}

func TestCalculateMultilateralOffset_CollectionReadError(t *testing.T) {
	t.Log("✓ Collection Read Error Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Mock error for the first collection that will be accessed
	// Since we iterate with i < j, the first collection will be between the first two MSPs alphabetically
	firstMSP := bankAMSP  // AccessBankMSP
	secondMSP := bankBMSP // GTBankMSP (next in alphabetical order)
	errorCollectionName := getCollectionName(firstMSP, secondMSP)

	chaincodeStub.On("GetPrivateDataByRange", errorCollectionName, "", "").Return(nil, fmt.Errorf("collection access denied"))

	// Execute
	result, err := smartContract.CalculateMultilateralOffset(transactionContext)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), fmt.Sprintf("failed to read PDC %s", errorCollectionName))
	require.Contains(t, err.Error(), "collection access denied")
}

func TestCalculateMultilateralOffset_InvalidPaymentJSON(t *testing.T) {
	t.Log("✓ Invalid Payment JSON Gracefully Skipped")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Create one collection with invalid JSON and one with valid payment
	paymentsByCollection := map[string][]settlement.PaymentDetails{
		getCollectionName(bankAMSP, bankBMSP): {
			createQueuedPayment("pay1", bankAMSP, bankBMSP, 500.0), // Valid payment
		},
	}

	// Setup most collections normally first
	setupComprehensiveMocking(chaincodeStub, paymentsByCollection)

	// Override one specific collection with invalid JSON mixed with valid data
	iterator := &MockMultilateralStateQueryIterator{}

	// Setup for: 1 valid payment + 1 invalid JSON + HasNext=false
	iterator.On("HasNext").Return(true).Once()  // For valid payment
	iterator.On("HasNext").Return(true).Once()  // For invalid JSON
	iterator.On("HasNext").Return(false).Once() // End iteration

	// Valid payment first
	validPayment := createQueuedPayment("pay1", bankAMSP, bankBMSP, 500.0)
	validPaymentJSON, _ := json.Marshal(validPayment)
	validKV := &queryresult.KV{
		Key:   "pay1",
		Value: validPaymentJSON,
	}
	iterator.On("Next").Return(validKV, nil).Once()

	// Invalid JSON second
	invalidKV := &queryresult.KV{
		Key:   "invalid",
		Value: []byte("invalid json data"),
	}
	iterator.On("Next").Return(invalidKV, nil).Once()

	iterator.On("Close").Return(nil).Once()

	// Replace the mock for this specific collection
	collectionName := getCollectionName(bankAMSP, bankBMSP)

	// Clear previous expectations for this collection and set new one
	chaincodeStub.ExpectedCalls = nil

	// Setup all other collections as empty
	allMSPs := []string{bankAMSP, bankBMSP, bankCMSP, bankDMSP, bankEMSP}
	for i, a := range allMSPs {
		for j := i + 1; j < len(allMSPs); j++ {
			b := allMSPs[j]
			coll := getCollectionName(a, b)

			if coll == collectionName {
				// Use our special iterator with invalid JSON
				chaincodeStub.On("GetPrivateDataByRange", coll, "", "").Return(iterator, nil).Once()
			} else {
				// Empty collection
				emptyIterator := &MockMultilateralStateQueryIterator{}
				emptyIterator.On("HasNext").Return(false).Once()
				emptyIterator.On("Close").Return(nil).Once()
				chaincodeStub.On("GetPrivateDataByRange", coll, "", "").Return(emptyIterator, nil).Once()
			}
		}
	}

	// Execute
	result, err := smartContract.CalculateMultilateralOffset(transactionContext)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should only process the valid payment, invalid JSON should be skipped
	expectedNetPositions := map[string]float64{
		bankAMSP: -500.0, // AccessBank pays 500 to GTBank
		bankBMSP: 500.0,  // GTBank receives 500 from AccessBank
	}

	require.Equal(t, expectedNetPositions, result.NetPositions)
	require.Len(t, result.Updates, 1) // Only the valid payment should create an update
	require.Equal(t, "pay1", result.Updates[0].ID)
	require.Equal(t, 0.0, result.Updates[0].AmountToSettle)
	require.Equal(t, "SETTLED", result.Updates[0].Status)
}

// =============================================================================
// Multilateral Offset Application Tests
// =============================================================================

// Helper to set multilateral offset update in transient data
func setMultilateralOffsetInTransientData(t *testing.T, chaincodeStub *mocks.ChaincodeStubInterface, netPositions map[string]float64, updates []settlement.MultiOffsetUpdate) {
	payload := settlement.MultiOffsetCalculation{
		NetPositions: netPositions,
		Updates:      updates,
	}

	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)

	transientData := map[string][]byte{
		"multilateralUpdate": payloadJSON,
	}
	chaincodeStub.On("GetTransient").Return(transientData, nil)
}

// Helper to create bank account
func createBankAccount(msp string, balance float64) settlement.BankAccount {
	return settlement.BankAccount{
		MSP:     msp,
		Balance: balance,
	}
}

func TestApplyMultilateralOffset_Success(t *testing.T) {
	t.Log("✓ Multilateral Offset Successfully Applied")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Create net positions and updates from a typical calculation
	netPositions := map[string]float64{
		bankAMSP: -400.0, // AccessBank net debtor
		bankBMSP: 200.0,  // GTBank net creditor
		bankCMSP: 200.0,  // Zenith net creditor
	}

	updates := []settlement.MultiOffsetUpdate{
		{ID: "pay1", PayerMSP: bankAMSP, PayeeMSP: bankBMSP, AmountToSettle: 0.0, Status: "SETTLED"},
		{ID: "pay2", PayerMSP: bankBMSP, PayeeMSP: bankCMSP, AmountToSettle: 0.0, Status: "SETTLED"},
		{ID: "pay3", PayerMSP: bankCMSP, PayeeMSP: bankAMSP, AmountToSettle: 0.0, Status: "SETTLED"},
	}

	setMultilateralOffsetInTransientData(t, chaincodeStub, netPositions, updates)

	// Mock existing payments for updates
	existingPay1 := createQueuedPayment("pay1", bankAMSP, bankBMSP, 1000.0)
	existingPay2 := createQueuedPayment("pay2", bankBMSP, bankCMSP, 800.0)
	existingPay3 := createQueuedPayment("pay3", bankCMSP, bankAMSP, 600.0)

	existingPay1JSON, _ := json.Marshal(existingPay1)
	existingPay2JSON, _ := json.Marshal(existingPay2)
	existingPay3JSON, _ := json.Marshal(existingPay3)

	coll1 := getCollectionName(bankAMSP, bankBMSP)
	coll2 := getCollectionName(bankBMSP, bankCMSP)
	coll3 := getCollectionName(bankCMSP, bankAMSP)

	chaincodeStub.On("GetPrivateData", coll1, "pay1").Return(existingPay1JSON, nil)
	chaincodeStub.On("GetPrivateData", coll2, "pay2").Return(existingPay2JSON, nil)
	chaincodeStub.On("GetPrivateData", coll3, "pay3").Return(existingPay3JSON, nil)
	chaincodeStub.On("PutPrivateData", coll1, "pay1", mock.Anything).Return(nil)
	chaincodeStub.On("PutPrivateData", coll2, "pay2", mock.Anything).Return(nil)
	chaincodeStub.On("PutPrivateData", coll3, "pay3", mock.Anything).Return(nil)

	// Mock settlement accounts
	accessBankAccount := createBankAccount(bankAMSP, 1000.0)
	gtBankAccount := createBankAccount(bankBMSP, 500.0)
	zenithAccount := createBankAccount(bankCMSP, 300.0)

	accessBankJSON, _ := json.Marshal(accessBankAccount)
	gtBankJSON, _ := json.Marshal(gtBankAccount)
	zenithJSON, _ := json.Marshal(zenithAccount)

	chaincodeStub.On("GetPrivateData", fmt.Sprintf("col-settlement-%s", bankAMSP), bankAMSP).Return(accessBankJSON, nil)
	chaincodeStub.On("GetPrivateData", fmt.Sprintf("col-settlement-%s", bankBMSP), bankBMSP).Return(gtBankJSON, nil)
	chaincodeStub.On("GetPrivateData", fmt.Sprintf("col-settlement-%s", bankCMSP), bankCMSP).Return(zenithJSON, nil)

	chaincodeStub.On("PutPrivateData", fmt.Sprintf("col-settlement-%s", bankAMSP), bankAMSP, mock.Anything).Return(nil)
	chaincodeStub.On("PutPrivateData", fmt.Sprintf("col-settlement-%s", bankBMSP), bankBMSP, mock.Anything).Return(nil)
	chaincodeStub.On("PutPrivateData", fmt.Sprintf("col-settlement-%s", bankCMSP), bankCMSP, mock.Anything).Return(nil)

	chaincodeStub.On("SetEvent", "NettingDebitExecuted", mock.Anything).Return(nil)
	chaincodeStub.On("SetEvent", "NettingCreditExecuted", mock.Anything).Return(nil)
	chaincodeStub.On("SetEvent", "MultilateralOffsetExecuted", mock.Anything).Return(nil)

	// Execute
	err := smartContract.ApplyMultilateralOffset(transactionContext)

	// Assert
	require.NoError(t, err)

	// Verify payment updates
	chaincodeStub.AssertCalled(t, "PutPrivateData", coll1, "pay1", mock.Anything)
	chaincodeStub.AssertCalled(t, "PutPrivateData", coll2, "pay2", mock.Anything)
	chaincodeStub.AssertCalled(t, "PutPrivateData", coll3, "pay3", mock.Anything)

	// Verify settlement account updates
	chaincodeStub.AssertCalled(t, "PutPrivateData", fmt.Sprintf("col-settlement-%s", bankAMSP), bankAMSP, mock.Anything)
	chaincodeStub.AssertCalled(t, "PutPrivateData", fmt.Sprintf("col-settlement-%s", bankBMSP), bankBMSP, mock.Anything)
	chaincodeStub.AssertCalled(t, "PutPrivateData", fmt.Sprintf("col-settlement-%s", bankCMSP), bankCMSP, mock.Anything)

	// Verify events
	chaincodeStub.AssertCalled(t, "SetEvent", "NettingDebitExecuted", mock.Anything)  // AccessBank debited
	chaincodeStub.AssertCalled(t, "SetEvent", "NettingCreditExecuted", mock.Anything) // GTBank & Zenith credited
	chaincodeStub.AssertCalled(t, "SetEvent", "MultilateralOffsetExecuted", mock.Anything)
}

// Test Apply Multilateral Offset without transient data
func TestApplyMultilateralOffset_NoTransientData(t *testing.T) {
	t.Log("✓ Missing Transient Data Error Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	chaincodeStub.On("GetTransient").Return(nil, fmt.Errorf("no transient data"))

	// Execute
	err := smartContract.ApplyMultilateralOffset(transactionContext)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "transient error")
	require.Contains(t, err.Error(), "no transient data")
}

// Test Apply Multilateral Offset with missing multilateral update key
// This simulates a scenario where the multilateral update key is not present in the transient data
// This should return an error indicating the key is required
func TestApplyMultilateralOffset_MissingMultilateralUpdate(t *testing.T) {
	t.Log("✓ Missing Multilateral Update Key Error Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Transient data without "multilateralUpdate" key
	transientData := map[string][]byte{
		"other": []byte("some other data"),
		"wrong": []byte("wrong key"),
	}

	chaincodeStub.On("GetTransient").Return(transientData, nil)

	// Execute
	err := smartContract.ApplyMultilateralOffset(transactionContext)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "multilateralUpdate required in transient")
}

// Test Apply Multilateral Offset with invalid JSON in transient data
// This simulates a scenario where the multilateral update data is not valid JSON
// The function should return an error indicating the payload cannot be unmarshalled
func TestApplyMultilateralOffset_InvalidJSON(t *testing.T) {
	t.Log("✓ Invalid Multilateral Update JSON Error Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Invalid JSON in transient data
	transientData := map[string][]byte{
		"multilateralUpdate": []byte("invalid json data {broken"),
	}

	chaincodeStub.On("GetTransient").Return(transientData, nil)

	// Execute
	err := smartContract.ApplyMultilateralOffset(transactionContext)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal payload")
}

func TestApplyMultilateralOffset_PutPrivateDataFailure(t *testing.T) {
	t.Log("✓ Payment Update Failure Error Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Create valid update data
	netPositions := map[string]float64{bankAMSP: -100.0, bankBMSP: 100.0}
	updates := []settlement.MultiOffsetUpdate{
		{ID: "pay1", PayerMSP: bankAMSP, PayeeMSP: bankBMSP, AmountToSettle: 0.0, Status: "SETTLED"},
	}

	setMultilateralOffsetInTransientData(t, chaincodeStub, netPositions, updates)

	// Mock existing payment
	existingPayment := createQueuedPayment("pay1", bankAMSP, bankBMSP, 100.0)
	existingPaymentJSON, _ := json.Marshal(existingPayment)

	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateData", collectionName, "pay1").Return(existingPaymentJSON, nil)

	// Mock PutPrivateData failure
	chaincodeStub.On("PutPrivateData", collectionName, "pay1", mock.Anything).Return(fmt.Errorf("write failed"))

	// Execute
	err := smartContract.ApplyMultilateralOffset(transactionContext)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "write failed pay1")
}

// =============================================================================
// Settlement Account Tests (DebitNetting & CreditNetting)
// =============================================================================
func TestDebitNetting_Success(t *testing.T) {
	t.Log("✓ Debit Netting Successfully Applied")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Mock existing settlement account with sufficient balance
	existingAccount := createBankAccount(bankAMSP, 1000.0)
	accountJSON, _ := json.Marshal(existingAccount)

	collectionName := fmt.Sprintf("col-settlement-%s", bankAMSP)
	chaincodeStub.On("GetPrivateData", collectionName, bankAMSP).Return(accountJSON, nil)
	chaincodeStub.On("PutPrivateData", collectionName, bankAMSP, mock.Anything).Return(nil)
	chaincodeStub.On("SetEvent", "NettingDebitExecuted", mock.Anything).Return(nil)

	// Execute - debit 300 from account
	err := smartContract.DebitNetting(transactionContext, bankAMSP, 300.0)

	// Assert
	require.NoError(t, err)
	chaincodeStub.AssertCalled(t, "GetPrivateData", collectionName, bankAMSP)
	chaincodeStub.AssertCalled(t, "PutPrivateData", collectionName, bankAMSP, mock.Anything)
	chaincodeStub.AssertCalled(t, "SetEvent", "NettingDebitExecuted", mock.Anything)
}

func TestCreditNetting_Success_ExistingAccount(t *testing.T) {
	t.Log("✓ Credit Netting Successfully Applied to Existing Account")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Mock existing settlement account
	existingAccount := createBankAccount(bankBMSP, 500.0)
	accountJSON, _ := json.Marshal(existingAccount)

	collectionName := fmt.Sprintf("col-settlement-%s", bankBMSP)
	chaincodeStub.On("GetPrivateData", collectionName, bankBMSP).Return(accountJSON, nil)
	chaincodeStub.On("PutPrivateData", collectionName, bankBMSP, mock.Anything).Return(nil)
	chaincodeStub.On("SetEvent", "NettingCreditExecuted", mock.Anything).Return(nil)

	// Execute - credit 200 to account
	err := smartContract.CreditNetting(transactionContext, bankBMSP, 200.0)

	// Assert
	require.NoError(t, err)
	chaincodeStub.AssertCalled(t, "GetPrivateData", collectionName, bankBMSP)
	chaincodeStub.AssertCalled(t, "PutPrivateData", collectionName, bankBMSP, mock.Anything)
	chaincodeStub.AssertCalled(t, "SetEvent", "NettingCreditExecuted", mock.Anything)
}

func TestDebitNetting_GetPrivateDataFailure(t *testing.T) {
	t.Log("✓ Debit Netting Get Private Data Failure Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Mock GetPrivateData failure (network/permission error)
	collectionName := fmt.Sprintf("col-settlement-%s", bankAMSP)
	chaincodeStub.On("GetPrivateData", collectionName, bankAMSP).Return(nil, fmt.Errorf("network timeout"))

	// Execute
	err := smartContract.DebitNetting(transactionContext, bankAMSP, 300.0)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("failed to read settlement account for %s", bankAMSP))
	require.Contains(t, err.Error(), "network timeout")

	// Verify no update operations were attempted
	chaincodeStub.AssertNotCalled(t, "PutPrivateData", collectionName, bankAMSP, mock.Anything)
	chaincodeStub.AssertNotCalled(t, "SetEvent", "NettingDebitExecuted", mock.Anything)
}

func TestCreditNetting_PutPrivateDataFailure(t *testing.T) {
	t.Log("✓ Credit Netting Put Private Data Failure Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Mock existing account
	existingAccount := createBankAccount(bankBMSP, 500.0)
	accountJSON, _ := json.Marshal(existingAccount)

	collectionName := fmt.Sprintf("col-settlement-%s", bankBMSP)
	chaincodeStub.On("GetPrivateData", collectionName, bankBMSP).Return(accountJSON, nil)

	// Mock PutPrivateData failure (storage/permission error)
	chaincodeStub.On("PutPrivateData", collectionName, bankBMSP, mock.Anything).Return(fmt.Errorf("permission denied"))

	// Execute
	err := smartContract.CreditNetting(transactionContext, bankBMSP, 200.0)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("failed to update settlement account for %s", bankBMSP))
	require.Contains(t, err.Error(), "permission denied")

	// Verify account was read but event was not emitted due to failure
	chaincodeStub.AssertCalled(t, "GetPrivateData", collectionName, bankBMSP)
	chaincodeStub.AssertCalled(t, "PutPrivateData", collectionName, bankBMSP, mock.Anything)
	chaincodeStub.AssertNotCalled(t, "SetEvent", "NettingCreditExecuted", mock.Anything)
}

func TestApplyMultilateralOffset_EmptyUpdatesArray(t *testing.T) {
	t.Log("✓ Empty Updates Array Handled Without Error")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Create scenario with net positions but no payment updates (unusual but possible)
	netPositions := map[string]float64{
		bankAMSP: -100.0,
		bankBMSP: 100.0,
	}
	updates := []settlement.MultiOffsetUpdate{} // Empty updates array

	setMultilateralOffsetInTransientData(t, chaincodeStub, netPositions, updates)

	// Mock settlement accounts for the net position processing
	accessBankAccount := createBankAccount(bankAMSP, 1000.0)
	gtBankAccount := createBankAccount(bankBMSP, 500.0)
	accessBankJSON, _ := json.Marshal(accessBankAccount)
	gtBankJSON, _ := json.Marshal(gtBankAccount)

	chaincodeStub.On("GetPrivateData", fmt.Sprintf("col-settlement-%s", bankAMSP), bankAMSP).Return(accessBankJSON, nil)
	chaincodeStub.On("GetPrivateData", fmt.Sprintf("col-settlement-%s", bankBMSP), bankBMSP).Return(gtBankJSON, nil)
	chaincodeStub.On("PutPrivateData", fmt.Sprintf("col-settlement-%s", bankAMSP), bankAMSP, mock.Anything).Return(nil)
	chaincodeStub.On("PutPrivateData", fmt.Sprintf("col-settlement-%s", bankBMSP), bankBMSP, mock.Anything).Return(nil)

	chaincodeStub.On("SetEvent", "NettingDebitExecuted", mock.Anything).Return(nil)
	chaincodeStub.On("SetEvent", "NettingCreditExecuted", mock.Anything).Return(nil)
	chaincodeStub.On("SetEvent", "MultilateralOffsetExecuted", mock.Anything).Return(nil)

	// Execute
	err := smartContract.ApplyMultilateralOffset(transactionContext)

	// Assert
	require.NoError(t, err)

	// Verify no payment updates (since updates array was empty)
	// But settlement account operations should still happen
	chaincodeStub.AssertCalled(t, "PutPrivateData", fmt.Sprintf("col-settlement-%s", bankAMSP), bankAMSP, mock.Anything)
	chaincodeStub.AssertCalled(t, "PutPrivateData", fmt.Sprintf("col-settlement-%s", bankBMSP), bankBMSP, mock.Anything)
	chaincodeStub.AssertCalled(t, "SetEvent", "MultilateralOffsetExecuted", mock.Anything)
}
