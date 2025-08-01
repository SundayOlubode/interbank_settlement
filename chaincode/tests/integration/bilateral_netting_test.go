package chaincode_test

import (
	"encoding/json"
	"testing"

	"github.com/SundayOlubode/interbank_settlement/chaincode/settlement"
	"github.com/SundayOlubode/interbank_settlement/chaincode/settlement/mocks"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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

// Helper function to prepare mocks
func prepMocks() (*mocks.TransactionContextInterface, *mocks.ChaincodeStubInterface) {
	chaincodeStub := &mocks.ChaincodeStubInterface{}
	transactionContext := &mocks.TransactionContextInterface{}
	transactionContext.On("GetStub").Return(chaincodeStub)
	transactionContext.On("GetClientIdentity").Return(nil).Maybe()
	return transactionContext, chaincodeStub
}

// Bank MSPs for testing
const (
	bankAMSP = "AccessBankMSP"
	bankBMSP = "GTBankMSP"
	bankCMSP = "ZenithBankMSP"
	bankDMSP = "FirstBankMSP"
	bankEMSP = "UBABankMSP"
)

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

// =============================================================================
// Integration Tests for Complete Bilateral Netting Flow
// =============================================================================
func TestBilateralNetting_CompleteFlow(t *testing.T) {
	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Calculate offset
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
	logBlue(t, "✓ Bilateral Offset Calculation phase completed")

	// Apply the calculated offset
	setBilateralOffsetInTransientData(t, chaincodeStub, offsetResult.Offset, offsetResult.Updates)

	// Mock the payments for apply step
	pay1JSON, _ := json.Marshal(payments[0])
	pay2JSON, _ := json.Marshal(payments[1])

	chaincodeStub.On("GetPrivateData", collectionName, "pay1").Return(pay1JSON, nil)
	chaincodeStub.On("GetPrivateData", collectionName, "pay2").Return(pay2JSON, nil)
	chaincodeStub.On("PutPrivateData", collectionName, "pay1", mock.Anything).Return(nil)
	chaincodeStub.On("PutPrivateData", collectionName, "pay2", mock.Anything).Return(nil)
	chaincodeStub.On("SetEvent", "BilateralOffsetExecuted", mock.Anything).Return(nil)
	logGreen(t, " ✓ Offset Calculation operations verified")

	err = smartContract.ApplyBilateralOffset(transactionContext, bankAMSP, bankBMSP)
	require.NoError(t, err)
	logBlue(t, "✓ Bilateral Offset Apply phase completed")
	logGreen(t, " ✓ Offset applied without errors")

	// Verify the complete flow
	chaincodeStub.AssertCalled(t, "SetEvent", "BilateralOffsetExecuted", mock.Anything)
	logYellow(t, "✓ Complete Bilateral Netting Flow Integration")
}
