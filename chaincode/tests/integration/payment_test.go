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

// =============================================================================
// INTEGRATION TESTS FOR PAYMENT.GO
// =============================================================================

const (
	accessBankMSP = "AccessBankMSP"
	gtBankMSP     = "GTBankMSP"
	zenithBankMSP = "ZenithBankMSP"
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorBlue   = "\033[34m"
)

// Mock implementation of getCollectionName for testing
func getCollectionName(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return fmt.Sprintf("col-%s-%s", a, b)
}

// Helper function to colorize log messages
func logYellow(t *testing.T, message string) {
	t.Logf("%s%s%s", colorYellow, message, colorReset)
}

func logGreen(t *testing.T, message string) {
	t.Logf("%s%s%s", colorGreen, message, colorReset)
}

func logBlue(t *testing.T, message string) {
	t.Logf("%s%s%s", colorBlue, message, colorReset)
}

// Helper function to prepare mocks
func prepPaymentMocks() (*mocks.TransactionContextInterface, *mocks.ChaincodeStubInterface) {
	chaincodeStub := &mocks.ChaincodeStubInterface{}
	transactionContext := &mocks.TransactionContextInterface{}
	transactionContext.On("GetStub").Return(chaincodeStub)
	transactionContext.On("GetClientIdentity").Return(nil).Maybe()
	return transactionContext, chaincodeStub
}

// Helper function to create test payment details
func createTestPaymentDetails() settlement.PaymentDetails {
	return settlement.PaymentDetails{
		ID:             "payment-001",
		PayerAcct:      "1234567890",
		PayeeAcct:      "0987654321",
		Amount:         1500.75,
		AmountToSettle: 1500.75,
		Currency:       "NGN",
		BVN:            "12345678901",
		PayerMSP:       accessBankMSP,
		PayeeMSP:       gtBankMSP,
		Status:         "PENDING",
		Timestamp:      time.Now().Unix(),
		User: settlement.BankUser{
			BVN:       "12345678901",
			Firstname: "John",
			Lastname:  "Doe",
			Birthdate: "15-08-1990",
			Gender:    "Male",
		},
	}
}

// Helper function to create BVN record
func createBVNRecord(user settlement.BankUser) settlement.BVNRecord {
	return settlement.BVNRecord{
		BVN:       user.BVN,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
		Gender:    user.Gender,
		Birthdate: user.Birthdate,
	}
}

// Helper function to set payment in transient data
func setPaymentInTransientData(t *testing.T, chaincodeStub *mocks.ChaincodeStubInterface, payment settlement.PaymentDetails) {
	paymentJSON, err := json.Marshal(payment)
	require.NoError(t, err)

	transientData := map[string][]byte{
		"payment": paymentJSON,
	}
	chaincodeStub.On("GetTransient").Return(transientData, nil)
}

// Helper function to setup BVN verification mock
func setupBVNVerificationMock(t *testing.T, chaincodeStub *mocks.ChaincodeStubInterface, user settlement.BankUser, shouldExist bool) {
	if shouldExist {
		bvnRecord := createBVNRecord(user)
		bvnJSON, err := json.Marshal(bvnRecord)
		require.NoError(t, err)
		chaincodeStub.On("GetPrivateData", "col-BVN", user.BVN).Return(bvnJSON, nil)
	} else {
		chaincodeStub.On("GetPrivateData", "col-BVN", user.BVN).Return(nil, nil)
	}
}

func TestCreatePaymentAndRetrieve_Success_EndToEndFlow(t *testing.T) {
	// Setup
	transactionContext, chaincodeStub := prepPaymentMocks()
	smartContract := settlement.SmartContract{}

	// Create test payment
	testPayment := createTestPaymentDetails()

	// Setup transient data for creation
	setPaymentInTransientData(t, chaincodeStub, testPayment)

	// Setup BVN verification (valid BVN exists)
	setupBVNVerificationMock(t, chaincodeStub, testPayment.User, true)

	// Expected collection name
	expectedCollection := getCollectionName(testPayment.PayerMSP, testPayment.PayeeMSP)

	// Mock successful creation operations
	chaincodeStub.On("PutPrivateData", expectedCollection, testPayment.ID, mock.Anything).Return(nil)
	chaincodeStub.On("PutState", testPayment.ID, mock.Anything).Return(nil)
	chaincodeStub.On("SetEvent", "PaymentPending", mock.Anything).Return(nil)

	// STEP 1: Create Payment
	err := smartContract.CreatePayment(transactionContext)
	require.NoError(t, err)
	logBlue(t, "✓ Payment creation phase completed")

	// Capture the stored data for retrieval simulation
	var storedPaymentData []byte
	var storedStubData []byte

	// Extract stored payment data from mock calls
	for _, call := range chaincodeStub.Calls {
		if call.Method == "PutPrivateData" && len(call.Arguments) >= 3 {
			if call.Arguments[0] == expectedCollection && call.Arguments[1] == testPayment.ID {
				storedPaymentData = call.Arguments[2].([]byte)
			}
		}
		if call.Method == "PutState" && len(call.Arguments) >= 2 {
			if call.Arguments[0] == testPayment.ID {
				storedStubData = call.Arguments[1].([]byte)
			}
		}
	}

	require.NotNil(t, storedPaymentData, "Payment data should be captured from creation")
	require.NotNil(t, storedStubData, "Stub data should be captured from creation")
	logGreen(t, " ✓ Created payment data captured for retrieval phase")

	// Clear previous mock expectations and set up for retrieval
	chaincodeStub.ExpectedCalls = nil
	chaincodeStub.Calls = nil

	// Mock retrieval operations using the stored data
	chaincodeStub.On("GetState", testPayment.ID).Return(storedStubData, nil)
	chaincodeStub.On("GetPrivateData", expectedCollection, testPayment.ID).Return(storedPaymentData, nil)

	// STEP 2: Retrieve Payment
	retrievedPayment, err := smartContract.GetIncomingPayment(transactionContext, testPayment.ID)
	require.NoError(t, err)
	require.NotNil(t, retrievedPayment)
	logBlue(t, "✓ Payment retrieval phase completed")

	// Verify retrieval operations
	chaincodeStub.AssertCalled(t, "GetState", testPayment.ID)
	chaincodeStub.AssertCalled(t, "GetPrivateData", expectedCollection, testPayment.ID)
	logGreen(t, " ✓ Retrieval operations verified")

	// STEP 3: Verify End-to-End Data Integrity
	require.Equal(t, testPayment.ID, retrievedPayment.ID)
	require.Equal(t, testPayment.PayerMSP, retrievedPayment.PayerMSP)
	require.Equal(t, testPayment.PayeeMSP, retrievedPayment.PayeeMSP)
	require.Equal(t, testPayment.Amount, retrievedPayment.Amount)
	require.Equal(t, testPayment.Amount, retrievedPayment.AmountToSettle) // Should be set during creation
	require.Equal(t, testPayment.Currency, retrievedPayment.Currency)
	require.Equal(t, testPayment.PayerAcct, retrievedPayment.PayerAcct)
	require.Equal(t, testPayment.PayeeAcct, retrievedPayment.PayeeAcct)
	require.Equal(t, testPayment.BVN, retrievedPayment.BVN)
	logGreen(t, " ✓ End-to-end financial data integrity verified")

	// Verify user data integrity
	require.Equal(t, testPayment.User.BVN, retrievedPayment.User.BVN)
	require.Equal(t, testPayment.User.Firstname, retrievedPayment.User.Firstname)
	require.Equal(t, testPayment.User.Lastname, retrievedPayment.User.Lastname)
	require.Equal(t, testPayment.User.Birthdate, retrievedPayment.User.Birthdate)
	require.Equal(t, testPayment.User.Gender, retrievedPayment.User.Gender)
	logGreen(t, " ✓ End-to-end user data integrity verified")

	// Verify that AmountToSettle was properly set during creation
	require.Equal(t, testPayment.Amount, retrievedPayment.AmountToSettle,
		"AmountToSettle should equal original Amount after creation")
	logGreen(t, " ✓ AmountToSettle assignment logic verified")

	// Verify the complete workflow
	// CREATE: Transient → BVN Verify → Private Store → Public Stub → Event
	// RETRIEVE: Public Stub → Collection Derivation → Private Retrieve
	// INTEGRITY: Original Data = Retrieved Data
	logYellow(t, "✓ End-to-End Payment Creation and Retrieval Flow Integration")
}
