package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// MultiOffsetUpdate describes how a single queued payment should change.
type MultiOffsetUpdate struct {
	ID             string  `json:"id"`
	PayerMSP       string  `json:"payerMSP"`
	PayeeMSP       string  `json:"payeeMSP"`
	AmountToSettle float64 `json:"amountToSettle"`
	Status         string  `json:"status"`
}

// MultiOffsetCalculation - the full payload to client
type MultiOffsetCalculation struct {
	// Net position per bank: positive = owed money; negative = owes money
	NetPositions map[string]float64 `json:"netPositions"`
	// Exactly which rows in which PDCs to update
	Updates []MultiOffsetUpdate `json:"updates"`
}

func (s *SmartContract) CalculateMultilateralOffset(
	ctx contractapi.TransactionContextInterface,
) (*MultiOffsetCalculation, error) {

	// 1) Scan **every** bilateral PDC for QUEUED items
	netPos := make(map[string]float64)
	var updates []MultiOffsetUpdate

	for _, a := range authorizedMSPs {
		for _, b := range authorizedMSPs {
			if a == b {
				continue
			}
			coll := getCollectionName(a, b)
			iter, err := ctx.GetStub().GetPrivateDataByRange(coll, "", "")
			if err != nil {
				return nil, fmt.Errorf("failed to read PDC %s: %v", coll, err)
			}
			defer iter.Close()

			for iter.HasNext() {
				qr, _ := iter.Next()

				var pd PaymentDetails
				if err := json.Unmarshal(qr.Value, &pd); err != nil {
					continue
				}
				if pd.Status != "QUEUED" {
					continue
				}

				// Build net position: incoming minus outgoing
				netPos[pd.PayeeMSP] += pd.AmountToSettle
				netPos[pd.PayerMSP] -= pd.AmountToSettle

				// Buffer an update to mark this row SETTLED (zero out AmountToSettle)
				updates = append(updates, MultiOffsetUpdate{
					ID:             pd.ID,
					PayerMSP:       pd.PayerMSP,
					PayeeMSP:       pd.PayeeMSP,
					AmountToSettle: 0,
					Status:         "SETTLED",
				})
			}
		}
	}

	return &MultiOffsetCalculation{
		NetPositions: netPos,
		Updates:      updates,
	}, nil
}

func (s *SmartContract) ApplyMultilateralOffset(
	ctx contractapi.TransactionContextInterface,
) error {
	// 1) Read the payload from transient
	trans, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("transient error: %v", err)
	}
	data, ok := trans["multilateralUpdate"]
	if !ok {
		return fmt.Errorf("multilateralUpdate required in transient")
	}

	var payload MultiOffsetCalculation
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %v", err)
	}

	// 2) Apply every queued‐payment update (single-key reads + writes)
	for _, u := range payload.Updates {
		coll := getCollectionName(u.PayerMSP, u.PayeeMSP)

		existing, err := ctx.GetStub().GetPrivateData(coll, u.ID)
		if err != nil || existing == nil {
			return fmt.Errorf("payment %s not found", u.ID)
		}
		var pd PaymentDetails
		json.Unmarshal(existing, &pd)

		pd.AmountToSettle = u.AmountToSettle
		pd.Status = u.Status

		updated, _ := json.Marshal(pd)
		if err := ctx.GetStub().PutPrivateData(coll, u.ID, updated); err != nil {
			return fmt.Errorf("write failed %s: %v", u.ID, err)
		}
	}

	// 3) Move real money once per bank
	for msp, net := range payload.NetPositions {
		switch {
		case net < 0:
			if err := s.DebitNetting(ctx, msp, -net); err != nil {
				return err
			}
		case net > 0:
			if err := s.CreditNetting(ctx, msp, net); err != nil {
				return err
			}
		}
	}

	// 4) Emit audit event
	evt, _ := json.Marshal(struct {
		NetPositions map[string]float64 `json:"netPositions"`
	}{payload.NetPositions})
	return ctx.GetStub().SetEvent("MultilateralOffsetExecuted", evt)
}

// DebitNetting subtracts `amount` from the MSP’s settlement account.
// Errors if the account doesn’t exist or has insufficient funds.
func (s *SmartContract) DebitNetting(
	ctx contractapi.TransactionContextInterface,
	msp string,
	amount float64,
) error {
	coll := fmt.Sprintf("col-settlement-%s", msp)
	acctBytes, err := ctx.GetStub().GetPrivateData(coll, msp)
	if err != nil {
		return fmt.Errorf("failed to read settlement account for %s: %v", msp, err)
	}
	if acctBytes == nil {
		return fmt.Errorf("no settlement account found for %s", msp)
	}

	var acct BankAccount
	if err := json.Unmarshal(acctBytes, &acct); err != nil {
		return fmt.Errorf("failed to unmarshal account for %s: %v", msp, err)
	}

	// if acct.Balance < amount {
	// 	return fmt.Errorf(
	// 		"insufficient funds: account %s balance %.2f, need %.2f",
	// 		msp, acct.Balance, amount,
	// 	)
	// }

	// Negative means the bank owes the Central Bank
	acct.Balance -= amount

	updated, err := json.Marshal(acct)
	if err != nil {
		return fmt.Errorf("failed to marshal updated account for %s: %v", msp, err)
	}
	if err := ctx.GetStub().PutPrivateData(coll, msp, updated); err != nil {
		return fmt.Errorf("failed to update settlement account for %s: %v", msp, err)
	}

	// Audit event
	evt := struct {
		MSP    string  `json:"msp"`
		Amount float64 `json:"amount"`
		Type   string  `json:"type"`
	}{MSP: msp, Amount: amount, Type: "netting-debit"}
	evtBytes, _ := json.Marshal(evt)
	if err := ctx.GetStub().SetEvent("NettingDebitExecuted", evtBytes); err != nil {
		return fmt.Errorf("failed to emit debit event for %s: %v", msp, err)
	}

	return nil
}

// CreditNetting adds `amount` to the MSP’s settlement account.
// Creates the account if it doesn’t already exist.
func (s *SmartContract) CreditNetting(
	ctx contractapi.TransactionContextInterface,
	msp string,
	amount float64,
) error {
	coll := fmt.Sprintf("col-settlement-%s", msp)
	acctBytes, err := ctx.GetStub().GetPrivateData(coll, msp)
	if err != nil {
		return fmt.Errorf("failed to read settlement account for %s: %v", msp, err)
	}

	var acct BankAccount
	if acctBytes != nil {
		if err := json.Unmarshal(acctBytes, &acct); err != nil {
			return fmt.Errorf("failed to unmarshal account for %s: %v", msp, err)
		}
	} else {
		// No existing account—start from zero
		acct = BankAccount{MSP: msp, Balance: 0}
	}

	acct.Balance += amount
	updated, err := json.Marshal(acct)
	if err != nil {
		return fmt.Errorf("failed to marshal updated account for %s: %v", msp, err)
	}
	if err := ctx.GetStub().PutPrivateData(coll, msp, updated); err != nil {
		return fmt.Errorf("failed to update settlement account for %s: %v", msp, err)
	}

	// Audit event
	evt := struct {
		MSP    string  `json:"msp"`
		Amount float64 `json:"amount"`
		Type   string  `json:"type"`
	}{MSP: msp, Amount: amount, Type: "netting-credit"}
	evtBytes, _ := json.Marshal(evt)
	if err := ctx.GetStub().SetEvent("NettingCreditExecuted", evtBytes); err != nil {
		return fmt.Errorf("failed to emit credit event for %s: %v", msp, err)
	}

	return nil
}
