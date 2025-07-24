package settlement

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing eNaira accounts and payments
type SmartContract struct {
	contractapi.Contract
}

// Init - Method for initializing smart contract
func (s *SmartContract) Init(ctx contractapi.TransactionContextInterface) error {
	return nil
}

// InitLedger seeds the invoking MSP with 15 billion eNaira and loads all BVN records into the col-BVN PDC.
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client MSP: %v", err)
	}

	// seed account balance
	starting := 15_000_000.0 // 15 billion eNaira
	acct := BankAccount{MSP: clientMSP, Balance: starting}
	acctBytes, err := json.Marshal(acct)
	if err != nil {
		return fmt.Errorf("marshal init account for %s: %v", clientMSP, err)
	}
	acctColl := fmt.Sprintf("col-settlement-%s", clientMSP)
	if err := ctx.GetStub().PutPrivateData(acctColl, clientMSP, acctBytes); err != nil {
		return fmt.Errorf("init account for %s: %v", clientMSP, err)
	}

	// Initialize BVN records
	return s.initializeBVNRecords(ctx)
}
