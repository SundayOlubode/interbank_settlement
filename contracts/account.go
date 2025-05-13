package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

type Account struct {
	ID      string `json:"id"`
	Org     string `json:"org"`
	Balance string `json:"balance"`
}

func (s *SmartContract) CreateAccount(ctx contractapi.TransactionContextInterface, initialBalance string) error {
	orgID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client MSP ID: %v", err)
	}

	accountID := "account_" + orgID

	exists, err := s.AccountExists(ctx, accountID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("account for org %s already exists", orgID)
	}

	account := Account{
		ID:      accountID,
		Org:     orgID,
		Balance: initialBalance,
	}

	accountJSON, err := json.Marshal(account)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(accountID, accountJSON)
}

func (s *SmartContract) ReadAccount(ctx contractapi.TransactionContextInterface) (*Account, error) {
	orgID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("failed to get client MSP ID: %v", err)
	}

	accountID := "account_" + orgID
	accountJSON, err := ctx.GetStub().GetState(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to read account: %v", err)
	}
	if accountJSON == nil {
		return nil, fmt.Errorf("account for org %s does not exist", orgID)
	}

	var account Account
	err = json.Unmarshal(accountJSON, &account)
	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (s *SmartContract) AccountExists(ctx contractapi.TransactionContextInterface, accountID string) (bool, error) {
	accountJSON, err := ctx.GetStub().GetState(accountID)
	if err != nil {
		return false, err
	}
	return accountJSON != nil, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		panic(fmt.Sprintf("Error creating account chaincode: %v", err))
	}

	if err := chaincode.Start(); err != nil {
		panic(fmt.Sprintf("Error starting account chaincode: %v", err))
	}
}
