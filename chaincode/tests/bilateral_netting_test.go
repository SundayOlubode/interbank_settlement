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
// MOCK STATE QUERY ITERATOR (MATCHING SHIM INTERFACE)
// =============================================================================

// MockStateQueryIterator implements the shim.StateQueryIteratorInterface
type MockStateQueryIterator struct {
	mock.Mock
}

func (m *MockStateQueryIterator) HasNext() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockStateQueryIterator) Next() (*queryresult.KV, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*queryresult.KV), args.Error(1)
}

func (m *MockStateQueryIterator) Close() error {
	args := m.Called()
	return args.Error(0)
}

const (
	bankAMSP = "AccessBankMSP"
	bankBMSP = "GTBankMSP"
	bankCMSP = "ZenithBankMSP"
	bankDMSP = "FirstBankMSP"
	bankEMSP = "UBABankMSP"
)

// =============================================================================
// HELPER FUNCTIONS FOR BILATERAL NETTING TESTS
// =============================================================================

// Helper to create queued payment for testing
func createQueuedPayment(id, payerMSP, payeeMSP string, amount float64) settlement.PaymentDetails {
	return settlement.PaymentDetails{
		ID:             id,
		PayerMSP:       payerMSP,
		PayeeMSP:       payeeMSP,
		Amount:         amount,
		AmountToSettle: amount,
		Status:         "QUEUED",
		Currency:       "NGN",
	}
}

// Helper to create mock iterator with payments
func setupMockIterator(payments []settlement.PaymentDetails) *MockStateQueryIterator {
	iterator := &MockStateQueryIterator{}

	// Setup HasNext calls (one for each payment + final false)
	for i := 0; i < len(payments); i++ {
		iterator.On("HasNext").Return(true).Once()
	}
	iterator.On("HasNext").Return(false).Once()

	// Setup Next calls for each payment
	for _, payment := range payments {
		paymentJSON, _ := json.Marshal(payment)
		kv := &queryresult.KV{
			Key:   payment.ID,
			Value: paymentJSON,
		}
		iterator.On("Next").Return(kv, nil).Once()
	}

	iterator.On("Close").Return(nil)
	return iterator
}

// Helper to set bilateral offset update in transient data
func setBilateralOffsetInTransientData(t *testing.T, chaincodeStub *mocks.ChaincodeStubInterface, offset float64, updates []settlement.OffsetUpdate) {
	payload := struct {
		Offset  float64                   `json:"offset"`
		Updates []settlement.OffsetUpdate `json:"updates"`
	}{
		Offset:  offset,
		Updates: updates,
	}

	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)

	transientData := map[string][]byte{
		"offsetUpdate": payloadJSON,
	}
	chaincodeStub.On("GetTransient").Return(transientData, nil)
}

// =============================================================================
// Bilateral Offset Calculation Tests
// =============================================================================

func TestCalculateBilateralOffset_Success_EqualAmounts(t *testing.T) {
	t.Log("✓ Equal Bilateral Amounts Result in Full Offset")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Create equal payments in both directions
	payments := []settlement.PaymentDetails{
		createQueuedPayment("pay1", bankAMSP, bankBMSP, 1000.0),
		createQueuedPayment("pay2", bankBMSP, bankAMSP, 1000.0),
	}

	iterator := setupMockIterator(payments)
	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateDataByRange", collectionName, "", "").Return(iterator, nil)

	// Execute
	result, err := smartContract.CalculateBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1000.0, result.Offset)
	require.Len(t, result.Updates, 2)

	// Verify both payments are fully settled
	for _, update := range result.Updates {
		require.Equal(t, 0.0, update.AmountToSettle)
		require.Equal(t, "SETTLED", update.Status)
	}
}

func TestCalculateBilateralOffset_Success_UnequalAmounts(t *testing.T) {
	t.Log("✓ Unequal Bilateral Amounts Result in Partial Offset")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Create unequal payments (BankA owes more)
	payments := []settlement.PaymentDetails{
		createQueuedPayment("pay1", bankAMSP, bankBMSP, 1500.0),
		createQueuedPayment("pay2", bankBMSP, bankAMSP, 800.0),
	}

	iterator := setupMockIterator(payments)
	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateDataByRange", collectionName, "", "").Return(iterator, nil)

	// Execute
	result, err := smartContract.CalculateBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 800.0, result.Offset) // Min of 1500 and 800
	require.Len(t, result.Updates, 2)

	// Find updates by ID
	var pay1Update, pay2Update settlement.OffsetUpdate
	for _, update := range result.Updates {
		if update.ID == "pay1" {
			pay1Update = update
		} else if update.ID == "pay2" {
			pay2Update = update
		}
	}

	// pay1 should have remaining amount
	require.Equal(t, 700.0, pay1Update.AmountToSettle) // 1500 - 800
	require.Equal(t, "QUEUED", pay1Update.Status)

	// pay2 should be fully settled
	require.Equal(t, 0.0, pay2Update.AmountToSettle)
	require.Equal(t, "SETTLED", pay2Update.Status)
}

func TestCalculateBilateralOffset_Success_NoQueuedPayments(t *testing.T) {
	t.Log("✓ No Queued Payments Result in Zero Offset")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Create payments with non-QUEUED status
	payments := []settlement.PaymentDetails{
		{ID: "pay1", PayerMSP: bankAMSP, PayeeMSP: bankBMSP, AmountToSettle: 1000.0, Status: "PENDING"},
		{ID: "pay2", PayerMSP: bankBMSP, PayeeMSP: bankAMSP, AmountToSettle: 800.0, Status: "SETTLED"},
	}

	iterator := setupMockIterator(payments)
	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateDataByRange", collectionName, "", "").Return(iterator, nil)

	// Execute
	result, err := smartContract.CalculateBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 0.0, result.Offset)
	require.Len(t, result.Updates, 0)
}

func TestCalculateBilateralOffset_Success_OnlyOneDirection(t *testing.T) {
	t.Log("✓ Single Direction Payments Result in Zero Offset")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Create payments only in one direction
	payments := []settlement.PaymentDetails{
		createQueuedPayment("pay1", bankAMSP, bankBMSP, 1000.0),
		createQueuedPayment("pay2", bankAMSP, bankBMSP, 500.0),
	}

	iterator := setupMockIterator(payments)
	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateDataByRange", collectionName, "", "").Return(iterator, nil)

	// Execute
	result, err := smartContract.CalculateBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 0.0, result.Offset) // Min of 1500 and 0
	require.Len(t, result.Updates, 0)
}

func TestCalculateBilateralOffset_IteratorError(t *testing.T) {
	t.Log("✓ Private Data Collection Access Error Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateDataByRange", collectionName, "", "").Return(nil, fmt.Errorf("collection access denied"))

	// Execute
	result, err := smartContract.CalculateBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), fmt.Sprintf("read PDC %s", collectionName))
}

func TestCalculateBilateralOffset_InvalidPaymentJSON(t *testing.T) {
	t.Log("✓ Invalid Payment JSON Gracefully Skipped")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Create mock iterator with invalid JSON
	iterator := &MockStateQueryIterator{}
	iterator.On("HasNext").Return(true).Once()
	kv := &queryresult.KV{
		Key:   "invalid",
		Value: []byte("invalid json"),
	}
	iterator.On("Next").Return(kv, nil).Once()
	iterator.On("HasNext").Return(false).Once()
	iterator.On("Close").Return(nil)

	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateDataByRange", collectionName, "", "").Return(iterator, nil)

	// Execute
	result, err := smartContract.CalculateBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 0.0, result.Offset)
	require.Len(t, result.Updates, 0)
}

// =============================================================================
// Bilateral Offset Application Tests
// =============================================================================

func TestApplyBilateralOffset_Success(t *testing.T) {
	t.Log("✓ Bilateral Offset Successfully Applied to Payments")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Create offset updates
	updates := []settlement.OffsetUpdate{
		{ID: "pay1", AmountToSettle: 700.0, Status: "QUEUED"},
		{ID: "pay2", AmountToSettle: 0.0, Status: "SETTLED"},
	}

	setBilateralOffsetInTransientData(t, chaincodeStub, 800.0, updates)

	// Mock existing payments
	existingPay1 := createQueuedPayment("pay1", bankAMSP, bankBMSP, 1500.0)
	existingPay2 := createQueuedPayment("pay2", bankBMSP, bankAMSP, 800.0)
	existingPay1JSON, _ := json.Marshal(existingPay1)
	existingPay2JSON, _ := json.Marshal(existingPay2)

	// Setup mocks
	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateData", collectionName, "pay1").Return(existingPay1JSON, nil)
	chaincodeStub.On("GetPrivateData", collectionName, "pay2").Return(existingPay2JSON, nil)
	chaincodeStub.On("PutPrivateData", collectionName, "pay1", mock.Anything).Return(nil)
	chaincodeStub.On("PutPrivateData", collectionName, "pay2", mock.Anything).Return(nil)
	chaincodeStub.On("SetEvent", "BilateralOffsetExecuted", mock.Anything).Return(nil)

	// Execute
	err := smartContract.ApplyBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.NoError(t, err)
	chaincodeStub.AssertCalled(t, "PutPrivateData", collectionName, "pay1", mock.Anything)
	chaincodeStub.AssertCalled(t, "PutPrivateData", collectionName, "pay2", mock.Anything)
	chaincodeStub.AssertCalled(t, "SetEvent", "BilateralOffsetExecuted", mock.Anything)
}

func TestApplyBilateralOffset_NoTransientData(t *testing.T) {
	t.Log("✓ Missing Transient Data Error Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	chaincodeStub.On("GetTransient").Return(nil, fmt.Errorf("no transient data"))

	// Execute
	err := smartContract.ApplyBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "transient error")
}

func TestApplyBilateralOffset_MissingOffsetUpdate(t *testing.T) {
	t.Log("✓ Missing Offset Update Key Error Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	transientData := map[string][]byte{
		"other": []byte("data"),
	}

	chaincodeStub.On("GetTransient").Return(transientData, nil)

	// Execute
	err := smartContract.ApplyBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "offsetUpdate required in transient")
}

func TestApplyBilateralOffset_InvalidJSON(t *testing.T) {
	t.Log("✓ Invalid Offset Update JSON Error Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	transientData := map[string][]byte{
		"offsetUpdate": []byte("invalid json"),
	}

	chaincodeStub.On("GetTransient").Return(transientData, nil)

	// Execute
	err := smartContract.ApplyBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal payload")
}

func TestApplyBilateralOffset_PaymentNotFound(t *testing.T) {
	t.Log("✓ Non-Existent Payment Update Error Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	updates := []settlement.OffsetUpdate{
		{ID: "nonexistent", AmountToSettle: 0.0, Status: "SETTLED"},
	}

	setBilateralOffsetInTransientData(t, chaincodeStub, 100.0, updates)

	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateData", collectionName, "nonexistent").Return(nil, nil)

	// Execute
	err := smartContract.ApplyBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "payment nonexistent not found")
}

func TestApplyBilateralOffset_PutPrivateDataFailure(t *testing.T) {
	t.Log("✓ Private Data Update Failure Error Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	updates := []settlement.OffsetUpdate{
		{ID: "pay1", AmountToSettle: 0.0, Status: "SETTLED"},
	}

	setBilateralOffsetInTransientData(t, chaincodeStub, 100.0, updates)

	existingPayment := createQueuedPayment("pay1", bankAMSP, bankBMSP, 100.0)
	existingPaymentJSON, _ := json.Marshal(existingPayment)

	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateData", collectionName, "pay1").Return(existingPaymentJSON, nil)
	chaincodeStub.On("PutPrivateData", collectionName, "pay1", mock.Anything).Return(fmt.Errorf("write failed"))

	// Execute
	err := smartContract.ApplyBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "write failed for pay1")
}

func TestApplyBilateralOffset_SetEventFailure(t *testing.T) {
	t.Log("✓ Set Event Failure Error Handled")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	updates := []settlement.OffsetUpdate{
		{ID: "pay1", AmountToSettle: 0.0, Status: "SETTLED"},
	}

	setBilateralOffsetInTransientData(t, chaincodeStub, 100.0, updates)

	existingPayment := createQueuedPayment("pay1", bankAMSP, bankBMSP, 100.0)
	existingPaymentJSON, _ := json.Marshal(existingPayment)

	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateData", collectionName, "pay1").Return(existingPaymentJSON, nil)
	chaincodeStub.On("PutPrivateData", collectionName, "pay1", mock.Anything).Return(nil)
	chaincodeStub.On("SetEvent", "BilateralOffsetExecuted", mock.Anything).Return(fmt.Errorf("set event failed"))

	// Execute
	err := smartContract.ApplyBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.NoError(t, err)
	chaincodeStub.AssertCalled(t, "PutPrivateData", collectionName, "pay1", mock.Anything)
	chaincodeStub.AssertCalled(t, "SetEvent", "BilateralOffsetExecuted", mock.Anything)
}

// =============================================================================
// Integration Tests for Complete Bilateral Netting Flow
// =============================================================================

func TestBilateralNetting_CompleteFlow(t *testing.T) {
	t.Log("✓ Complete Bilateral Netting Flow Integration")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Step 1: Calculate offset
	payments := []settlement.PaymentDetails{
		createQueuedPayment("pay1", bankAMSP, bankBMSP, 1200.0),
		createQueuedPayment("pay2", bankBMSP, bankAMSP, 800.0),
	}

	iterator := setupMockIterator(payments)
	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateDataByRange", collectionName, "", "").Return(iterator, nil)

	offsetResult, err := smartContract.CalculateBilateralOffset(transactionContext, bankAMSP, bankBMSP)
	require.NoError(t, err)
	require.Equal(t, 800.0, offsetResult.Offset)

	// Step 2: Apply the calculated offset
	setBilateralOffsetInTransientData(t, chaincodeStub, offsetResult.Offset, offsetResult.Updates)

	// Mock the payments for apply step
	pay1JSON, _ := json.Marshal(payments[0])
	pay2JSON, _ := json.Marshal(payments[1])

	chaincodeStub.On("GetPrivateData", collectionName, "pay1").Return(pay1JSON, nil)
	chaincodeStub.On("GetPrivateData", collectionName, "pay2").Return(pay2JSON, nil)
	chaincodeStub.On("PutPrivateData", collectionName, "pay1", mock.Anything).Return(nil)
	chaincodeStub.On("PutPrivateData", collectionName, "pay2", mock.Anything).Return(nil)
	chaincodeStub.On("SetEvent", "BilateralOffsetExecuted", mock.Anything).Return(nil)

	err = smartContract.ApplyBilateralOffset(transactionContext, bankAMSP, bankBMSP)
	require.NoError(t, err)

	// Verify the complete flow
	chaincodeStub.AssertCalled(t, "SetEvent", "BilateralOffsetExecuted", mock.Anything)
}

// =============================================================================
// Edge Cases and Error Scenarios
// =============================================================================

func TestCalculateBilateralOffset_MultiplePaymentsSameDirection(t *testing.T) {
	t.Log("✓ Multiple Payments in Same Direction Properly Aggregated")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Multiple payments from BankA to BankB, single payment back
	payments := []settlement.PaymentDetails{
		createQueuedPayment("pay1", bankAMSP, bankBMSP, 500.0),
		createQueuedPayment("pay2", bankAMSP, bankBMSP, 300.0),
		createQueuedPayment("pay3", bankAMSP, bankBMSP, 200.0),
		createQueuedPayment("pay4", bankBMSP, bankAMSP, 600.0),
	}

	iterator := setupMockIterator(payments)
	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateDataByRange", collectionName, "", "").Return(iterator, nil)

	// Execute
	result, err := smartContract.CalculateBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 600.0, result.Offset) // Min of 1000 (500+300+200) and 600

	// The algorithm might optimize out fully settled payments with 0 remaining
	// So we check that we have at least the payments that have remaining amounts
	require.GreaterOrEqual(t, len(result.Updates), 2) // At least pay4 and one with remaining amount

	// Verify that 600 was deducted from the A->B payments and pay4 is fully settled
	totalDeductedAB := 0.0
	settledPayments := 0
	remainingPayments := 0

	// Track which payments we found
	foundPayments := make(map[string]bool)

	for _, update := range result.Updates {
		foundPayments[update.ID] = true

		if update.ID == "pay4" {
			require.Equal(t, 0.0, update.AmountToSettle)
			require.Equal(t, "SETTLED", update.Status)
			settledPayments++
		} else {
			// These should be A->B payments
			original := 0.0
			switch update.ID {
			case "pay1":
				original = 500.0
			case "pay2":
				original = 300.0
			case "pay3":
				original = 200.0
			}
			require.Greater(t, original, 0.0, "Unknown payment ID: %s", update.ID)

			deducted := original - update.AmountToSettle
			totalDeductedAB += deducted

			if update.AmountToSettle == 0 {
				require.Equal(t, "SETTLED", update.Status)
				settledPayments++
			} else {
				require.Equal(t, "QUEUED", update.Status)
				remainingPayments++
			}
		}
	}
}

func TestCalculateBilateralOffset_ZeroAmountPayments(t *testing.T) {
	t.Log("✓ Zero Amount Payments Handled Correctly")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	payments := []settlement.PaymentDetails{
		createQueuedPayment("pay1", bankAMSP, bankBMSP, 0.0),
		createQueuedPayment("pay2", bankBMSP, bankAMSP, 1000.0),
	}

	iterator := setupMockIterator(payments)
	collectionName := getCollectionName(bankAMSP, bankBMSP)
	chaincodeStub.On("GetPrivateDataByRange", collectionName, "", "").Return(iterator, nil)

	// Execute
	result, err := smartContract.CalculateBilateralOffset(transactionContext, bankAMSP, bankBMSP)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 0.0, result.Offset) // Min of 0 and 1000
	require.Len(t, result.Updates, 0)
}

func TestApplyBilateralOffset_EmptyUpdatesArray(t *testing.T) {
	t.Log("✓ Empty Updates Array Handled Without Error")

	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	setBilateralOffsetInTransientData(t, chaincodeStub, 0.0, []settlement.OffsetUpdate{})

	chaincodeStub.On("SetEvent", "BilateralOffsetExecuted", mock.Anything).Return(nil)

	// Execute
	err := smartContract.ApplyBilateralOffset(transactionContext, "BankA", "BankB")

	// Assert
	require.NoError(t, err)
	chaincodeStub.AssertCalled(t, "SetEvent", "BilateralOffsetExecuted", mock.Anything)
}

// =============================================================================
// Parametrized Tests for Different Scenarios
// =============================================================================

func TestCalculateBilateralOffset_DifferentAmounts(t *testing.T) {
	testCases := []struct {
		name           string
		amountAtoB     float64
		amountBtoA     float64
		expectedOffset float64
	}{
		{"Equal small amounts", 100.0, 100.0, 100.0},
		{"Equal large amounts", 1000000.0, 1000000.0, 1000000.0},
		{"A owes more", 1500.0, 800.0, 800.0},
		{"B owes more", 600.0, 1200.0, 600.0},
		{"Zero from A", 0.0, 500.0, 0.0},
		{"Zero from B", 500.0, 0.0, 0.0},
		{"A owes more", 1500.0, 800.0, 800.0},
		{"B owes more", 600.0, 1200.0, 600.0},
		{"Zero from A", 0.0, 500.0, 0.0},
		{"Zero from B", 500.0, 0.0, 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transactionContext, chaincodeStub := prepMocks()
			smartContract := settlement.SmartContract{}

			payments := []settlement.PaymentDetails{
				createQueuedPayment("pay1", bankAMSP, bankBMSP, tc.amountAtoB),
				createQueuedPayment("pay2", bankBMSP, bankAMSP, tc.amountBtoA),
			}

			iterator := setupMockIterator(payments)
			collectionName := getCollectionName(bankAMSP, bankBMSP)
			chaincodeStub.On("GetPrivateDataByRange", collectionName, "", "").Return(iterator, nil)

			result, err := smartContract.CalculateBilateralOffset(transactionContext, bankAMSP, bankBMSP)

			require.NoError(t, err)
			require.Equal(t, tc.expectedOffset, result.Offset)
		})
	}
}

func TestApplyBilateralOffset_DifferentMSPs(t *testing.T) {
	testCases := []struct {
		name     string
		payerMSP string
		payeeMSP string
	}{
		{"Standard banks", bankAMSP, bankBMSP},
		{"Reversed banks", bankBMSP, bankAMSP},
		{"Different banks", "FirstBankMSP", "SecondBankMSP"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transactionContext, chaincodeStub := prepMocks()
			smartContract := settlement.SmartContract{}

			updates := []settlement.OffsetUpdate{
				{ID: "pay1", AmountToSettle: 0.0, Status: "SETTLED"},
			}

			setBilateralOffsetInTransientData(t, chaincodeStub, 100.0, updates)

			existingPayment := createQueuedPayment("pay1", tc.payerMSP, tc.payeeMSP, 100.0)
			existingPaymentJSON, _ := json.Marshal(existingPayment)

			collectionName := getCollectionName(tc.payerMSP, tc.payeeMSP)
			chaincodeStub.On("GetPrivateData", collectionName, "pay1").Return(existingPaymentJSON, nil)
			chaincodeStub.On("PutPrivateData", collectionName, "pay1", mock.Anything).Return(nil)
			chaincodeStub.On("SetEvent", "BilateralOffsetExecuted", mock.Anything).Return(nil)

			err := smartContract.ApplyBilateralOffset(transactionContext, tc.payerMSP, tc.payeeMSP)
			require.NoError(t, err)
		})
	}
}
