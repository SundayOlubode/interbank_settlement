package settlement

//go:generate mockery --name=TransactionContextInterface --srcpkg=github.com/hyperledger/fabric-contract-api-go/contractapi --output=mocks --outpkg=mocks
//go:generate mockery --name=ChaincodeStubInterface --srcpkg=github.com/hyperledger/fabric-chaincode-go/shim --output=mocks --outpkg=mocks
