package main

import (
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	assetChaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		fmt.Printf("Error creating meatcc chaincode: %v", err)
		return
	}
	if err := assetChaincode.Start(); err != nil {
		fmt.Printf("Error starting meatcc chaincode: %v", err)
	}
}