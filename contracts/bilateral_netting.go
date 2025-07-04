package main

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// OffsetUpdate describes how a single PaymentDetails record should change.
type OffsetUpdate struct {
	ID             string  `json:"id"`
	AmountToSettle float64 `json:"amountToSettle"`
	Status         string  `json:"status"`
}

// OffsetCalculation - full payload to client
type OffsetCalculation struct {
	Offset  float64        `json:"offset"`
	Updates []OffsetUpdate `json:"updates"`
}

func (s *SmartContract) CalculateBilateralOffset(
	ctx contractapi.TransactionContextInterface,
	mspA, mspB string,
) (*OffsetCalculation, error) {
	coll := getCollectionName(mspA, mspB)
	iter, err := ctx.GetStub().GetPrivateDataByRange(coll, "", "")
	if err != nil {
		return nil, fmt.Errorf("read PDC %s: %v", coll, err)
	}
	defer iter.Close()

	var queueAB, queueBA []*PaymentDetails
	var totalAB, totalBA float64
	for iter.HasNext() {
		qr, _ := iter.Next()
		var pd PaymentDetails
		if err := json.Unmarshal(qr.Value, &pd); err != nil {
			continue
		}
		if pd.Status != "QUEUED" {
			continue
		}
		switch {
		case pd.PayerMSP == mspA && pd.PayeeMSP == mspB:
			queueAB = append(queueAB, &pd)
			totalAB += pd.AmountToSettle
		case pd.PayerMSP == mspB && pd.PayeeMSP == mspA:
			queueBA = append(queueBA, &pd)
			totalBA += pd.AmountToSettle
		}
	}

	offset := math.Min(totalAB, totalBA)
	// build updates
	updates := make([]OffsetUpdate, 0)
	apply := func(list []*PaymentDetails, rem float64) float64 {
		for _, pd := range list {
			if rem == 0 {
				break
			}
			deduct := math.Min(pd.AmountToSettle, rem)
			pd.AmountToSettle -= deduct
			rem -= deduct
			status := "QUEUED"
			if pd.AmountToSettle == 0 {
				status = "SETTLED"
			}
			updates = append(updates, OffsetUpdate{
				ID:             pd.ID,
				AmountToSettle: pd.AmountToSettle,
				Status:         status,
			})
		}
		return rem
	}
	apply(queueAB, offset)
	apply(queueBA, offset)

	// return a struct instead of JSON string
	return &OffsetCalculation{
		Offset:  offset,
		Updates: updates,
	}, nil
}

func (s *SmartContract) ApplyBilateralOffset(
	ctx contractapi.TransactionContextInterface,
	mspA, mspB string,
) error {
	trans, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("transient error: %v", err)
	}
	data, ok := trans["offsetUpdate"]
	if !ok {
		return fmt.Errorf("offsetUpdate required in transient")
	}

	type Update struct {
		ID             string  `json:"id"`
		AmountToSettle float64 `json:"amountToSettle"`
		Status         string  `json:"status"`
	}
	var payload struct {
		Offset  float64  `json:"offset"`
		Updates []Update `json:"updates"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %v", err)
	}

	coll := getCollectionName(mspA, mspB)

	for _, u := range payload.Updates {
		// optional: read-single-key to validate version
		existingBytes, err := ctx.GetStub().GetPrivateData(coll, u.ID)
		if err != nil || existingBytes == nil {
			return fmt.Errorf("payment %s not found", u.ID)
		}

		// reconstruct PaymentDetails with new fields
		var pd PaymentDetails
		json.Unmarshal(existingBytes, &pd)
		pd.AmountToSettle = u.AmountToSettle
		pd.Status = u.Status

		updated, _ := json.Marshal(pd)
		if err := ctx.GetStub().PutPrivateData(coll, u.ID, updated); err != nil {
			return fmt.Errorf("write failed for %s: %v", u.ID, err)
		}
	}

	evt := struct {
		MSPA   string  `json:"mspA"`
		MSPB   string  `json:"mspB"`
		Offset float64 `json:"offset"`
	}{mspA, mspB, payload.Offset}
	evtBytes, _ := json.Marshal(evt)
	ctx.GetStub().SetEvent("BilateralOffsetExecuted", evtBytes)

	return nil
}
