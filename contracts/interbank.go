package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

type Wallet struct {
	ID      string `json:"id"`
	Org     string `json:"org"`
	Balance string `json:"balance"`
}

func (s *SmartContract) CreateWallet(ctx contractapi.TransactionContextInterface, initialBalance string) error {
	orgID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client MSP ID: %v", err)
	}

	walletID := "wallet_" + orgID

	exists, err := s.WalletExists(ctx, walletID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("wallet for org %s already exists", orgID)
	}

	wallet := Wallet{
		ID:      walletID,
		Org:     orgID,
		Balance: initialBalance,
	}

	walletJSON, err := json.Marshal(wallet)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(walletID, walletJSON)
}

func (s *SmartContract) ReadWallet(ctx contractapi.TransactionContextInterface) (*Wallet, error) {
	orgID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("failed to get client MSP ID: %v", err)
	}

	walletID := "wallet_" + orgID
	walletJSON, err := ctx.GetStub().GetState(walletID)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet: %v", err)
	}
	if walletJSON == nil {
		return nil, fmt.Errorf("wallet for org %s does not exist", orgID)
	}

	var wallet Wallet
	err = json.Unmarshal(walletJSON, &wallet)
	if err != nil {
		return nil, err
	}

	return &wallet, nil
}

func (s *SmartContract) WalletExists(ctx contractapi.TransactionContextInterface, walletID string) (bool, error) {
	walletJSON, err := ctx.GetStub().GetState(walletID)
	if err != nil {
		return false, err
	}

	txid, err := ctx.GetStub().GetTxID()

	fmt.Printf("Transaction ID: %s\n", txid)

	return walletJSON != nil, nil

}

func main() {

	fmt.Println("Creating & Starting Interbank Smart Contract...")

	chaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		panic(fmt.Sprintf("Error creating chaincode: %v", err))
	}

	if err := chaincode.Start(); err != nil {
		panic(fmt.Sprintf("Error starting chaincode: %v", err))
	}
}
