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

// Helper to create bank account
func createBankAccount(msp string, balance float64) settlement.BankAccount {
	return settlement.BankAccount{
		MSP:     msp,
		Balance: balance,
	}
}

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

// =============================================================================
// Integration Tests for Complete Multilateral Netting Flow
// =============================================================================

func TestMultilateralNetting_CompleteFlow(t *testing.T) {
	// Setup
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Calculate multilateral offset
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

	offsetResult, err := smartContract.CalculateMultilateralOffset(transactionContext)
	require.NoError(t, err)
	logBlue(t, "✓ Multilateral offset calculation phase completed without errors")

	// Verify calculation results
	expectedNetPositions := map[string]float64{
		bankAMSP: -400.0, // AccessBank: receives 600, pays 1000 = -400
		bankBMSP: 200.0,  // GTBank: receives 1000, pays 800 = +200
		bankCMSP: 200.0,  // Zenith: receives 800, pays 600 = +200
	}
	require.Equal(t, expectedNetPositions, offsetResult.NetPositions)
	require.Len(t, offsetResult.Updates, 3)
	logGreen(t, " ✓ Net positions calculated correctly")
	logGreen(t, " ✓ Updates for payments matches correctly")

	// Apply the calculated offset
	// Clear previous mock expectations and set up for application
	chaincodeStub.ExpectedCalls = nil

	setMultilateralOffsetInTransientData(t, chaincodeStub, offsetResult.NetPositions, offsetResult.Updates)

	// Mock payment updates
	for _, update := range offsetResult.Updates {
		var payment settlement.PaymentDetails
		switch update.ID {
		case "pay1":
			payment = createQueuedPayment("pay1", bankAMSP, bankBMSP, 1000.0)
		case "pay2":
			payment = createQueuedPayment("pay2", bankBMSP, bankCMSP, 800.0)
		case "pay3":
			payment = createQueuedPayment("pay3", bankCMSP, bankAMSP, 600.0)
		}
		paymentJSON, _ := json.Marshal(payment)
		collectionName := getCollectionName(update.PayerMSP, update.PayeeMSP)
		chaincodeStub.On("GetPrivateData", collectionName, update.ID).Return(paymentJSON, nil)
		chaincodeStub.On("PutPrivateData", collectionName, update.ID, mock.Anything).Return(nil)
	}

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

	err = smartContract.ApplyMultilateralOffset(transactionContext)
	require.NoError(t, err)
	logBlue(t, "✓ Multilateral Offset Apply phase completed without errors")

	// Verify the complete flow
	chaincodeStub.AssertCalled(t, "SetEvent", "MultilateralOffsetExecuted", mock.Anything)
	chaincodeStub.AssertCalled(t, "SetEvent", "NettingDebitExecuted", mock.Anything)  // AccessBank debited
	chaincodeStub.AssertCalled(t, "SetEvent", "NettingCreditExecuted", mock.Anything) // GTBank & Zenith credited
	logGreen(t, " ✓ Netting events emitted successfully")

	// Verify all payments were updated
	chaincodeStub.AssertNumberOfCalls(t, "PutPrivateData", 6) // 3 payments + 3 settlement accounts
	logGreen(t, " ✓ All payment updates applied successfully")

	logYellow(t, "✓ Complete Multilateral Netting Flow Integration")
}
