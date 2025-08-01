package chaincode_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/SundayOlubode/interbank_settlement/chaincode/settlement"
	"github.com/SundayOlubode/interbank_settlement/chaincode/settlement/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	myOrg1Clientid = "AccessBankMSP"
	myOrg2Clientid = "GTBankMSP"
)

// Helper function to prepare mocks
func prepMocks() (*mocks.TransactionContextInterface, *mocks.ChaincodeStubInterface) {
	chaincodeStub := &mocks.ChaincodeStubInterface{}
	transactionContext := &mocks.TransactionContextInterface{}
	transactionContext.On("GetStub").Return(chaincodeStub)
	transactionContext.On("GetClientIdentity").Return(nil).Maybe()
	return transactionContext, chaincodeStub
}

// Helper function to create sample payment data
func createTestPaymentDetails() *settlement.PaymentDetails {
	return &settlement.PaymentDetails{
		ID:             "payment-123",
		PayerAcct:      "1234567890",
		PayeeAcct:      "0987654321",
		Amount:         1000.50,
		AmountToSettle: 1000.50,
		Currency:       "NGN",
		BVN:            "23455677890",
		PayerMSP:       myOrg1Clientid,
		PayeeMSP:       myOrg2Clientid,
		Status:         "PENDING",
		Timestamp:      time.Now().Unix(),
		User: settlement.BankUser{
			BVN:       "23455677890",
			Firstname: "Emeka",
			Lastname:  "Okafor",
			Birthdate: "02-11-1985",
			Gender:    "Male",
		},
	}
}

// Helper function to set payment in transient data
func setPaymentInTransientData(t *testing.T, chaincodeStub *mocks.ChaincodeStubInterface, payment *settlement.PaymentDetails) {
	paymentJSON, err := json.Marshal(payment)
	require.NoError(t, err)

	transientData := map[string][]byte{
		"payment": paymentJSON,
	}
	chaincodeStub.On("GetTransient").Return(transientData, nil)

	// Mock BVN verification - return valid BVN data
	bvnRecord := settlement.BVNRecord{
		BVN:       payment.User.BVN,
		Firstname: payment.User.Firstname,
		Lastname:  payment.User.Lastname,
		Gender:    payment.User.Gender,
		Birthdate: payment.User.Birthdate,
	}
	bvnJSON, err := json.Marshal(bvnRecord)
	require.NoError(t, err)

	// Mock the BVN collection call
	chaincodeStub.On("GetPrivateData", "col-BVN", payment.User.BVN).Return(bvnJSON, nil)
}

func TestInitiatePayment_Success(t *testing.T) {
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Test payment does not exist yet
	testPayment := createTestPaymentDetails()
	setPaymentInTransientData(t, chaincodeStub, testPayment)

	// Mock successful operations
	chaincodeStub.On("PutPrivateData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	chaincodeStub.On("PutState", mock.Anything, mock.Anything).Return(nil)
	chaincodeStub.On("SetEvent", mock.Anything, mock.Anything).Return(nil)

	// Execute CreatePayment
	err := smartContract.CreatePayment(transactionContext)
	require.NoError(t, err)

	// Verify the calls were made
	chaincodeStub.AssertCalled(t, "GetPrivateData", "col-BVN", testPayment.User.BVN)
	chaincodeStub.AssertCalled(t, "PutPrivateData", mock.Anything, mock.Anything, mock.Anything)
	chaincodeStub.AssertCalled(t, "PutState", mock.Anything, mock.Anything)
	chaincodeStub.AssertCalled(t, "SetEvent", mock.Anything, mock.Anything)
}

func TestInitiatePayment_NoTransientData(t *testing.T) {
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Return error when getting transient data
	chaincodeStub.On("GetTransient").Return(nil, fmt.Errorf("no transient data"))

	err := smartContract.CreatePayment(transactionContext)
	require.EqualError(t, err, "error getting transient data: no transient data")
}

func TestInitiatePayment_MissingPaymentKey(t *testing.T) {
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Return transient data without "payment" key
	transientData := map[string][]byte{
		"other": []byte("data"),
	}
	chaincodeStub.On("GetTransient").Return(transientData, nil)

	err := smartContract.CreatePayment(transactionContext)
	require.EqualError(t, err, "payment details must be provided in transient data under 'payment'")
}

func TestInitiatePayment_BVNVerificationFailure(t *testing.T) {
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	testPayment := createTestPaymentDetails()
	paymentJSON, err := json.Marshal(testPayment)
	require.NoError(t, err)

	transientData := map[string][]byte{
		"payment": paymentJSON,
	}
	chaincodeStub.On("GetTransient").Return(transientData, nil)

	// Mock BVN not found
	chaincodeStub.On("GetPrivateData", "col-BVN", testPayment.User.BVN).Return(nil, nil)

	err = smartContract.CreatePayment(transactionContext)
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("BVN %s not registered", testPayment.User.BVN))
}

func TestInitiatePayment_InvalidJSON(t *testing.T) {
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	// Return invalid JSON in transient data
	transientData := map[string][]byte{
		"payment": []byte("invalid json"),
	}
	chaincodeStub.On("GetTransient").Return(transientData, nil)

	err := smartContract.CreatePayment(transactionContext)
	require.Contains(t, err.Error(), "failed to unmarshal payment details")
}

func TestInitiatePayment_PutPrivateDataFailure(t *testing.T) {
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	testPayment := createTestPaymentDetails()
	setPaymentInTransientData(t, chaincodeStub, testPayment)

	// Mock PutPrivateData failure
	chaincodeStub.On("PutPrivateData", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("put private data failed"))

	err := smartContract.CreatePayment(transactionContext)
	require.EqualError(t, err, "failed to put private payment data: put private data failed")
}

func TestInitiatePayment_PutStateFailure(t *testing.T) {
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	testPayment := createTestPaymentDetails()
	setPaymentInTransientData(t, chaincodeStub, testPayment)

	// Mock successful PutPrivateData but failed PutState
	chaincodeStub.On("PutPrivateData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	chaincodeStub.On("PutState", mock.Anything, mock.Anything).Return(fmt.Errorf("put state failed"))

	err := smartContract.CreatePayment(transactionContext)
	require.EqualError(t, err, "failed to put payment stub: put state failed")
}

func TestInitiatePayment_SetEventFailure(t *testing.T) {
	transactionContext, chaincodeStub := prepMocks()
	smartContract := settlement.SmartContract{}

	testPayment := createTestPaymentDetails()
	setPaymentInTransientData(t, chaincodeStub, testPayment)

	// Mock successful PutPrivateData and PutState but failed SetEvent
	chaincodeStub.On("PutPrivateData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	chaincodeStub.On("PutState", mock.Anything, mock.Anything).Return(nil)
	chaincodeStub.On("SetEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("set event failed"))

	err := smartContract.CreatePayment(transactionContext)
	require.Contains(t, err.Error(), "set event failed")
}

func TestInitiatePayment_DifferentAmounts(t *testing.T) {
	testCases := []struct {
		name   string
		amount float64
	}{
		{"Small amount", 10.50},
		{"Large amount", 1000000.99},
		{"Zero amount", 0.0},
		{"Fractional kobo", 1234.56},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transactionContext, chaincodeStub := prepMocks()
			smartContract := settlement.SmartContract{}

			testPayment := createTestPaymentDetails()
			testPayment.Amount = tc.amount
			testPayment.AmountToSettle = tc.amount
			setPaymentInTransientData(t, chaincodeStub, testPayment)

			// Mock successful operations
			chaincodeStub.On("PutPrivateData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			chaincodeStub.On("PutState", mock.Anything, mock.Anything).Return(nil)
			chaincodeStub.On("SetEvent", mock.Anything, mock.Anything).Return(nil)

			err := smartContract.CreatePayment(transactionContext)
			require.NoError(t, err)
		})
	}
}

func TestInitiatePayment_DifferentMSPs(t *testing.T) {
	testCases := []struct {
		name     string
		payerMSP string
		payeeMSP string
	}{
		{"Standard banks", "AccessBankMSP", "GTBankMSP"},
		{"Reversed banks", "GTBankMSP", "AccessBankMSP"},
		{"Different banks", "FirstBankMSP", "SecondBankMSP"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transactionContext, chaincodeStub := prepMocks()
			smartContract := settlement.SmartContract{}

			testPayment := createTestPaymentDetails()
			testPayment.PayerMSP = tc.payerMSP
			testPayment.PayeeMSP = tc.payeeMSP
			setPaymentInTransientData(t, chaincodeStub, testPayment)

			// Mock successful operations
			chaincodeStub.On("PutPrivateData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			chaincodeStub.On("PutState", mock.Anything, mock.Anything).Return(nil)
			chaincodeStub.On("SetEvent", mock.Anything, mock.Anything).Return(nil)

			err := smartContract.CreatePayment(transactionContext)
			require.NoError(t, err)
		})
	}
}
