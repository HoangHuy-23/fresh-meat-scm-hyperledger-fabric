// blockchain/chaincode/meatcc/meatcc.go

package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing meat assets.
type SmartContract struct {
	contractapi.Contract
}

// ===============================================================
// CÁC CẤU TRÚC DỮ LIỆU
// ===============================================================

// MediaPointer defines a reference to an off-chain media file (e.g., on S3).
type MediaPointer struct {
	S3Bucket string `json:"s3Bucket"`
	S3Key    string `json:"s3Key"`
	MimeType string `json:"mimeType"`
}

// FarmDetails captures information from the farming stage.
type FarmDetails struct {
	FarmOrgName  string         `json:"farmOrgName"`
	FacilityName string         `json:"facilityName"`
	SowingDate   string         `json:"sowingDate"`
	HarvestDate  string         `json:"harvestDate"`
	Fertilizers  []string       `json:"fertilizers"`
	Pesticides   []string       `json:"pesticides"`
	Certificates []MediaPointer `json:"certificates"`
}

// ProcessingStep defines a single step in the processing stage.
type ProcessingStep struct {
	Name      string `json:"name"`
	Technique string `json:"technique"`
	Timestamp string `json:"timestamp"`
}

// ProcessingDetails captures information from the processing stage.
type ProcessingDetails struct {
	ProcessorOrgName string           `json:"processorOrgName"`
	FacilityName     string           `json:"facilityName"`
	Steps            []ProcessingStep `json:"steps"`
	Certificates     []MediaPointer   `json:"certificates"`
}

// ShipmentTimeline defines a point in time during a shipment.
type ShipmentTimeline struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Location  string `json:"location,omitempty"`
}

// ShipmentDetails captures information about a shipment.
type ShipmentDetails struct {
	ShipmentID     string             `json:"shipmentID"`
	CarrierOrgName string             `json:"carrierOrgName"`
	DriverName     string             `json:"driverName"`
	VehiclePlate   string             `json:"vehiclePlate"`
	FromFacility   string             `json:"fromFacility"`
	ToFacility     string             `json:"toFacility"`
	Status         string             `json:"status"`
	Timeline       []ShipmentTimeline `json:"timeline"`
}

// StorageDetails captures information during storage (warehouse or retail).
type StorageDetails struct {
	OwnerOrgName    string `json:"ownerOrgName"`
	FacilityName    string `json:"facilityName"`
	LocationInStore string `json:"locationInStore,omitempty"`
	Temperature     string `json:"temperature,omitempty"`
	Note            string `json:"note"`
}

// SoldDetails captures information about the final sale.
type SoldDetails struct {
	RetailerOrgName string `json:"retailerOrgName"`
	FacilityName    string `json:"facilityName"`
	SaleTimestamp   string `json:"saleTimestamp"`
}

// Event captures a significant event in the asset's lifecycle.
type Event struct {
	Type      string      `json:"type"`
	ActorMSP  string      `json:"actorMSP"`
	Timestamp string      `json:"timestamp"`
	TxID      string      `json:"txID"`
	Details   interface{} `json:"details"`
}

// MeatAsset is the main object stored on the ledger.
type MeatAsset struct {
	ObjectType     string   `json:"docType"`
	AssetID        string   `json:"assetID"`
	ParentAssetIDs []string `json:"parentAssetIDs"`
	ProductName    string   `json:"productName"`
	Status         string   `json:"status"`
	History        []Event  `json:"history"`
}

// ChildAssetInput is used for the ProcessAndSplitBatch function.
type ChildAssetInput struct {
	AssetID     string `json:"assetID"`
	ProductName string `json:"productName"`
}

// FullAssetTrace: Cấu trúc dữ liệu hoàn chỉnh cho việc truy xuất.
type FullAssetTrace struct {
	AssetID        string   `json:"assetID"`
	ParentAssetIDs []string `json:"parentAssetIDs"`
	ProductName    string   `json:"productName"`
	Status         string   `json:"status"`
	FullHistory    []Event  `json:"fullHistory"`
}

// ===============================================================
// CÁC HÀM CHAINCODE (TRANSACTIONS)
// ===============================================================

// CreateFarmingBatch creates a new batch at the farm.
func (s *SmartContract) CreateFarmingBatch(ctx contractapi.TransactionContextInterface, assetID string, productName string, farmDetailsJSON string) error {
	// TODO: Add access control logic here (e.g., check if caller's MSP is a Farm MSP).

	exists, err := s.assetExists(ctx, assetID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the asset %s already exists", assetID)
	}

	var farmDetails FarmDetails
	err = json.Unmarshal([]byte(farmDetailsJSON), &farmDetails)
	if err != nil {
		return fmt.Errorf("failed to unmarshal farmDetailsJSON: %v", err)
	}

	clientMSP, _ := ctx.GetClientIdentity().GetMSPID()
	txID := ctx.GetStub().GetTxID()
	timestamp, _ := ctx.GetStub().GetTxTimestamp()

	event := Event{
		Type:      "FARMING",
		ActorMSP:  clientMSP,
		Timestamp: time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).Format(time.RFC3339),
		TxID:      txID,
		Details:   farmDetails,
	}

	asset := MeatAsset{
		ObjectType:     "MeatAsset",
		AssetID:        assetID,
		ParentAssetIDs: []string{},
		ProductName:    productName,
		Status:         "HARVESTED_AT_FARM",
		History:        []Event{event},
	}

	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(assetID, assetJSON)
}

// ProcessAndSplitBatch processes a parent batch and creates multiple child batches.
func (s *SmartContract) ProcessAndSplitBatch(ctx contractapi.TransactionContextInterface, parentAssetID string, childAssetsJSON string, processingDetailsJSON string) error {
	// TODO: Add access control logic here (e.g., check for Processor MSP).

	parentAsset, err := s.readAsset(ctx, parentAssetID)
	if err != nil {
		return err
	}

	// IMPROVEMENT: More robust status check.
	// A batch can only be processed if it has been received.
	if parentAsset.Status != "RECEIVED_AT_PROCESSOR" {
		return fmt.Errorf("asset %s is in status '%s' and cannot be processed", parentAssetID, parentAsset.Status)
	}

	var processingDetails ProcessingDetails
	err = json.Unmarshal([]byte(processingDetailsJSON), &processingDetails)
	if err != nil {
		return fmt.Errorf("failed to unmarshal processingDetailsJSON: %v", err)
	}

	clientMSP, _ := ctx.GetClientIdentity().GetMSPID()
	txID := ctx.GetStub().GetTxID()
	timestamp, _ := ctx.GetStub().GetTxTimestamp()
	formattedTime := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).Format(time.RFC3339)

	processingEvent := Event{
		Type:      "PROCESSING",
		ActorMSP:  clientMSP,
		Timestamp: formattedTime,
		TxID:      txID,
		Details:   processingDetails,
	}

	parentAsset.History = append(parentAsset.History, processingEvent)
	parentAsset.Status = "PROCESSED_AND_SPLIT"

	parentAssetJSON, err := json.Marshal(parentAsset)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(parentAssetID, parentAssetJSON)
	if err != nil {
		return err
	}

	var childAssets []ChildAssetInput
	err = json.Unmarshal([]byte(childAssetsJSON), &childAssets)
	if err != nil {
		return fmt.Errorf("failed to unmarshal childAssetsJSON: %v", err)
	}

	for _, child := range childAssets {
		exists, err := s.assetExists(ctx, child.AssetID)
		if err != nil {
			return err
		}
		if exists {
			// IMPROVEMENT: Instead of failing the whole transaction, we could just log a warning and skip.
			// For now, failing is safer.
			return fmt.Errorf("child asset %s already exists", child.AssetID)
		}

		creationEvent := Event{
			Type:      "CREATED_FROM_PROCESSING",
			ActorMSP:  clientMSP,
			Timestamp: formattedTime,
			TxID:      txID,
			Details:   fmt.Sprintf("Created from parent batch %s", parentAssetID),
		}

		newChildAsset := MeatAsset{
			ObjectType:     "MeatAsset",
			AssetID:        child.AssetID,
			ParentAssetIDs: []string{parentAssetID},
			ProductName:    child.ProductName,
			Status:         "PACKAGED_AT_PROCESSOR",
			History:        []Event{creationEvent},
		}

		childAssetJSON, err := json.Marshal(newChildAsset)
		if err != nil {
			return err
		}
		err = ctx.GetStub().PutState(child.AssetID, childAssetJSON)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddEvent adds a generic event to an asset's history.
func (s *SmartContract) AddEvent(ctx contractapi.TransactionContextInterface, assetID string, eventType string, newStatus string, detailsJSON string) error {
	// TODO: Add access control logic here (e.g., check MSP based on eventType).

	asset, err := s.readAsset(ctx, assetID)
	if err != nil {
		return err
	}

	var details interface{}
	// IMPROVEMENT: Unmarshal into specific structs for data validation.
	switch eventType {
	case "SHIPPING":
		var d ShipmentDetails
		err = json.Unmarshal([]byte(detailsJSON), &d)
		details = d
	case "RECEIVING":
		var d StorageDetails // Re-using StorageDetails for receiving event
		err = json.Unmarshal([]byte(detailsJSON), &d)
		details = d
	case "STORAGE_UPDATE":
		var d StorageDetails
		err = json.Unmarshal([]byte(detailsJSON), &d)
		details = d
	case "SOLD":
		var d SoldDetails
		err = json.Unmarshal([]byte(detailsJSON), &d)
		details = d
	default:
		// For flexibility, allow generic JSON object for unknown types
		var genericDetails map[string]interface{}
		err = json.Unmarshal([]byte(detailsJSON), &genericDetails)
		details = genericDetails
	}
	if err != nil {
		return fmt.Errorf("failed to unmarshal detailsJSON for event type %s: %v", eventType, err)
	}

	clientMSP, _ := ctx.GetClientIdentity().GetMSPID()
	txID := ctx.GetStub().GetTxID()
	timestamp, _ := ctx.GetStub().GetTxTimestamp()

	event := Event{
		Type:      eventType,
		ActorMSP:  clientMSP,
		Timestamp: time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).Format(time.RFC3339),
		TxID:      txID,
		Details:   details,
	}

	asset.History = append(asset.History, event)
	asset.Status = newStatus

	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(assetID, assetJSON)
}

// ===============================================================
// CÁC HÀM QUERY (READ-ONLY)
// ===============================================================

// GetAssetWithFullHistory: Lấy thông tin chi tiết của một tài sản
// và toàn bộ lịch sử của nó, bao gồm cả lịch sử của các lô cha.
func (s *SmartContract) GetAssetWithFullHistory(ctx contractapi.TransactionContextInterface, assetID string) (*FullAssetTrace, error) {
    // 1. Lấy thông tin của chính tài sản được yêu cầu
    asset, err := s.readAsset(ctx, assetID)
    if err != nil {
        return nil, err
    }

    // 2. Lấy toàn bộ lịch sử đệ quy
    fullHistory, err := s.getAssetHistoryRecursive(ctx, assetID) // Gọi hàm private
    if err != nil {
        return nil, err
    }

    // 3. Tạo đối tượng trả về hoàn chỉnh
    traceResult := FullAssetTrace{
        AssetID:        asset.AssetID,
        ParentAssetIDs: asset.ParentAssetIDs,
        ProductName:    asset.ProductName,
        Status:         asset.Status,
        FullHistory:    fullHistory,
    }

    return &traceResult, nil
}

// ===============================================================
// CÁC HÀM TIỆN ÍCH (PRIVATE)
// ===============================================================


// GetAssetHistoryRecursive retrieves the full history of an asset, including its parents.
// This function is read-only and does not submit a transaction.
func (s *SmartContract) getAssetHistoryRecursive(ctx contractapi.TransactionContextInterface, assetID string) ([]Event, error) {
	var fullHistory []Event
	queue := []string{assetID}
	processedIDs := make(map[string]bool)

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		if processedIDs[currentID] {
			continue
		}

		asset, err := s.readAsset(ctx, currentID)
		if err != nil {
			return nil, fmt.Errorf("failed to read asset %s: %v", currentID, err)
		}

		fullHistory = append(fullHistory, asset.History...)

		for _, parentID := range asset.ParentAssetIDs {
			if !processedIDs[parentID] {
				queue = append(queue, parentID)
			}
		}
		processedIDs[currentID] = true
	}

	sort.Slice(fullHistory, func(i, j int) bool {
		return fullHistory[i].Timestamp < fullHistory[j].Timestamp
	})

	return fullHistory, nil
}


// readAsset is a private helper function to read an asset from the ledger.
func (s *SmartContract) readAsset(ctx contractapi.TransactionContextInterface, assetID string) (*MeatAsset, error) {
	assetJSON, err := ctx.GetStub().GetState(assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		return nil, fmt.Errorf("the asset %s does not exist", assetID)
	}

	var asset MeatAsset
	err = json.Unmarshal(assetJSON, &asset)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

// assetExists is a private helper function to check if an asset exists.
func (s *SmartContract) assetExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return assetJSON != nil, nil
}

// ===============================================================
// HÀM MAIN
// ===============================================================

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