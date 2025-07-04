package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// initializeBVNRecords loads all BVN records into the col-BVN PDC
func (s *SmartContract) initializeBVNRecords(ctx contractapi.TransactionContextInterface) error {
	bvns := []BVNRecord{
		// Original 10 records
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

		// Additional 30 diverse records
		{BVN: "32344566789", Firstname: "Aminu", Lastname: "Abdullahi", Middlename: "Suleiman", Gender: "Male", Phone: "08103456789", Birthdate: "14-07-1992"},
		{BVN: "33455667890", Firstname: "Chinwendu", Lastname: "Okonkwo", Middlename: "Adaeze", Gender: "Female", Phone: "08114567890", Birthdate: "23-02-1988"},
		{BVN: "34566778901", Firstname: "Tolu", Lastname: "Adesanya", Middlename: "Folake", Gender: "Female", Phone: "08125678901", Birthdate: "11-05-1995"},
		{BVN: "35677889012", Firstname: "Bashir", Lastname: "Yusuf", Middlename: "Ahmad", Gender: "Male", Phone: "08136789012", Birthdate: "08-12-1984"},
		{BVN: "36788990123", Firstname: "Ijeoma", Lastname: "Uzoma", Middlename: "Chioma", Gender: "Female", Phone: "08147890123", Birthdate: "19-09-1991"},
		{BVN: "37899001234", Firstname: "Seun", Lastname: "Ogundipe", Middlename: "Adebayo", Gender: "Male", Phone: "08158901234", Birthdate: "06-03-1986"},
		{BVN: "38900112345", Firstname: "Hauwa", Lastname: "Mohammed", Middlename: "Zainab", Gender: "Female", Phone: "08169012345", Birthdate: "25-11-1993"},
		{BVN: "39011223456", Firstname: "Kemi", Lastname: "Adeleye", Middlename: "Bukola", Gender: "Female", Phone: "08170123456", Birthdate: "13-01-1989"},
		{BVN: "40122334567", Firstname: "Chinedu", Lastname: "Emeagwali", Middlename: "Ikechukwu", Gender: "Male", Phone: "08181234567", Birthdate: "27-06-1990"},
		{BVN: "41233445678", Firstname: "Adunni", Lastname: "Bakare", Middlename: "Omotola", Gender: "Female", Phone: "08192345678", Birthdate: "04-04-1987"},
		{BVN: "42344556789", Firstname: "Musa", Lastname: "Garba", Middlename: "Ibrahim", Gender: "Male", Phone: "08103456790", Birthdate: "18-08-1994"},
		{BVN: "43455667891", Firstname: "Ngozi", Lastname: "Ikwuemesi", Middlename: "Adanna", Gender: "Female", Phone: "08114567891", Birthdate: "02-10-1985"},
		{BVN: "44566778902", Firstname: "Gbenga", Lastname: "Olumide", Middlename: "Ayodeji", Gender: "Male", Phone: "08125678902", Birthdate: "21-12-1996"},
		{BVN: "45677889013", Firstname: "Aisha", Lastname: "Bello", Middlename: "Fatima", Gender: "Female", Phone: "08136789013", Birthdate: "16-07-1988"},
		{BVN: "46788990124", Firstname: "Emeka", Lastname: "Nnamdi", Middlename: "Chukwuma", Gender: "Male", Phone: "08147890124", Birthdate: "09-05-1992"},
		{BVN: "47899001235", Firstname: "Damilola", Lastname: "Adebisi", Middlename: "Temiloluwa", Gender: "Female", Phone: "08158901235", Birthdate: "30-01-1990"},
		{BVN: "48900112346", Firstname: "Yahaya", Lastname: "Aliyu", Middlename: "Usman", Gender: "Male", Phone: "08169012346", Birthdate: "12-11-1983"},
		{BVN: "49011223457", Firstname: "Blessing", Lastname: "Okoro", Middlename: "Chiamaka", Gender: "Female", Phone: "08170123457", Birthdate: "07-03-1994"},
		{BVN: "50122334568", Firstname: "Babatunde", Lastname: "Ajayi", Middlename: "Olumuyiwa", Gender: "Male", Phone: "08181234568", Birthdate: "24-09-1986"},
		{BVN: "51233445679", Firstname: "Safiya", Lastname: "Umar", Middlename: "Hadiza", Gender: "Female", Phone: "08192345679", Birthdate: "15-06-1991"},
		{BVN: "52344556780", Firstname: "Obinna", Lastname: "Ezechukwu", Middlename: "Kenechukwu", Gender: "Male", Phone: "08103456791", Birthdate: "03-02-1989"},
		{BVN: "53455667891", Firstname: "Funmi", Lastname: "Elegbede", Middlename: "Abisola", Gender: "Female", Phone: "08114567892", Birthdate: "28-04-1987"},
		{BVN: "54566778903", Firstname: "Abdullahi", Lastname: "Maikano", Middlename: "Nasiru", Gender: "Male", Phone: "08125678903", Birthdate: "20-10-1995"},
		{BVN: "55677889014", Firstname: "Chidinma", Lastname: "Okafor", Middlename: "Precious", Gender: "Female", Phone: "08136789014", Birthdate: "11-08-1993"},
		{BVN: "56788990125", Firstname: "Femi", Lastname: "Ogundimu", Middlename: "Oluwaseyi", Gender: "Male", Phone: "08147890125", Birthdate: "05-12-1984"},
		{BVN: "57899001236", Firstname: "Rukayat", Lastname: "Salami", Middlename: "Aminat", Gender: "Female", Phone: "08158901236", Birthdate: "17-07-1990"},
		{BVN: "58900112347", Firstname: "Kelechi", Lastname: "Anyanwu", Middlename: "Chukwuebuka", Gender: "Male", Phone: "08169012347", Birthdate: "22-01-1988"},
		{BVN: "59011223458", Firstname: "Yetunde", Lastname: "Oladele", Middlename: "Omolara", Gender: "Female", Phone: "08170123458", Birthdate: "08-05-1992"},
		{BVN: "60122334569", Firstname: "Sani", Lastname: "Danjuma", Middlename: "Yakubu", Gender: "Male", Phone: "08181234569", Birthdate: "14-03-1985"},
		{BVN: "61233445680", Firstname: "Nneka", Lastname: "Okoye", Middlename: "Chizoba", Gender: "Female", Phone: "08192345680", Birthdate: "26-11-1996"},
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
