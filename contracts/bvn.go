package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// initializeBVNRecords loads all BVN records into the col-BVN PDC
func (s *SmartContract) initializeBVNRecords(ctx contractapi.TransactionContextInterface) error {
	bvns := []BVNRecord{
		{BVN: "22133455678", Firstname: "Oluwaseun", Lastname: "Adebanjo", Middlename: "Temitope", Gender: "Female", Phone: "08031234567", Birthdate: "15-04-1990"},
		{BVN: "23455677890", Firstname: "Emeka", Lastname: "Okafor", Middlename: "Chukwuemeka", Gender: "Male", Phone: "08134567890", Birthdate: "02-11-1985"},
		{BVN: "24566788901", Firstname: "Chiamaka", Lastname: "Nwankwo", Middlename: "Amarachi", Gender: "Female", Phone: "08046789012", Birthdate: "28-09-1993"},
		{BVN: "25677899012", Firstname: "Ibrahim", Lastname: "Muhammad", Middlename: "Abdulrahman", Gender: "Male", Phone: "08156789023", Birthdate: "10-01-1988"},
		{BVN: "26788900123", Firstname: "Fatima", Lastname: "Ahmad", Middlename: "Sadiku", Gender: "Female", Phone: "08067890123", Birthdate: "05-06-1992"},
		{BVN: "27899011234", Firstname: "Tunde", Lastname: "Oloyede", Middlename: "Oluwatobi", Gender: "Male", Phone: "08178901234", Birthdate: "22-12-1983"},
		{BVN: "28900122345", Firstname: "Amarachi", Lastname: "Eze", Middlename: "Chidera", Gender: "Female", Phone: "08089012345", Birthdate: "17-08-1995"},
		{BVN: "29011233456", Firstname: "Chukwuemeka", Lastname: "Okoro", Middlename: "Obinna", Gender: "Male", Phone: "08190123456", Birthdate: "30-03-1989"},
		{BVN: "30122344567", Firstname: "Aikevbiosa", Lastname: "Okunrola", Middlename: "Oluwasegun", Gender: "Male", Phone: "08091234567", Birthdate: "12-07-1991"},
		{BVN: "31233455678", Firstname: "Bolanle", Lastname: "Soniyi", Middlename: "Omowunmi", Gender: "Female", Phone: "08102345678", Birthdate: "09-10-1987"},
	}

	for _, rec := range bvns {
		recBytes, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal BVN record %s: %v", rec.BVN, err)
		}
		if err := ctx.GetStub().PutPrivateData("col-BVN", rec.BVN, recBytes); err != nil {
			return fmt.Errorf("failed to put BVN record %s: %v", rec.BVN, err)
		}
	}

	return nil
}

// verifyBVN verifies BVN details against the central PDC
func (s *SmartContract) verifyBVN(ctx contractapi.TransactionContextInterface, user BankUser) error {
	bvnBytes, err := ctx.GetStub().GetPrivateData("col-BVN", user.BVN)
	if err != nil || bvnBytes == nil {
		return fmt.Errorf("BVN %s not registered", user.BVN)
	}

	var bvnRecord BVNRecord
	if err := json.Unmarshal(bvnBytes, &bvnRecord); err != nil {
		return fmt.Errorf("failed to unmarshal BVN record: %v", err)
	}

	if bvnRecord.Lastname != user.Lastname ||
		bvnRecord.Firstname != user.Firstname ||
		bvnRecord.Birthdate != user.Birthdate ||
		bvnRecord.Gender != user.Gender {
		return fmt.Errorf("BVN details do not match user's details")
	}

	return nil
}

// GetBVNRecord retrieves a BVN record from the BVN PDC
func (s *SmartContract) GetBVNRecord(ctx contractapi.TransactionContextInterface, bvn string) (*BVNRecord, error) {
	bvnBytes, err := ctx.GetStub().GetPrivateData("col-BVN", bvn)
	if err != nil {
		return nil, fmt.Errorf("failed to get BVN record %s: %v", bvn, err)
	}
	if bvnBytes == nil {
		return nil, fmt.Errorf("BVN record %s not found", bvn)
	}

	var bvnRecord BVNRecord
	if err := json.Unmarshal(bvnBytes, &bvnRecord); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BVN record: %v", err)
	}

	return &bvnRecord, nil
}
