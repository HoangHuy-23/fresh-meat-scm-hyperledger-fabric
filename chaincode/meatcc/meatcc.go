package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing meat assets and shipments.
type SmartContract struct {
	contractapi.Contract
}

// ===============================================================
// CÁC CẤU TRÚC DỮ LIỆU
// ===============================================================

type Quantity struct {
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
}

type MediaPointer struct {
	S3Bucket string `json:"s3Bucket"`
	S3Key    string `json:"s3Key"`
	MimeType string `json:"mimeType"`
}

type FarmDetails struct {
	FarmOrgName  string         `json:"farmOrgName"`
	FacilityName string         `json:"facilityName"`
	SowingDate   string         `json:"sowingDate"`
	HarvestDate  string         `json:"harvestDate"`
	Fertilizers  []string       `json:"fertilizers"`
	Pesticides   []string       `json:"pesticides"`
	Certificates []MediaPointer `json:"certificates"`
}

type ProcessingStep struct {
	Name      string `json:"name"`
	Technique string `json:"technique"`
	Timestamp string `json:"timestamp"`
}

type ProcessingDetails struct {
	ProcessorOrgName string           `json:"processorOrgName"`
	FacilityName     string           `json:"facilityName"`
	Steps            []ProcessingStep `json:"steps"`
	Certificates     []MediaPointer   `json:"certificates"`
}

type ShipmentTimeline struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Location  string `json:"location,omitempty"`
}

type StorageDetails struct {
	OwnerOrgName    string `json:"ownerOrgName"`
	FacilityName    string `json:"facilityName"`
	LocationInStore string `json:"locationInStore,omitempty"`
	Temperature     string `json:"temperature,omitempty"`
	Note            string `json:"note"`
}

type SoldDetails struct {
	RetailerOrgName string `json:"retailerOrgName"`
	FacilityName    string `json:"facilityName"`
	SaleTimestamp   string `json:"saleTimestamp"`
}

type Event struct {
	Type      string      `json:"type"`
	ActorMSP  string      `json:"actorMSP"`
	Timestamp string      `json:"timestamp"`
	TxID      string      `json:"txID"`
	Details   interface{} `json:"details"`
}

type MeatAsset struct {
	ObjectType       string   `json:"docType"`
	AssetID          string   `json:"assetID"`
	ParentAssetIDs   []string `json:"parentAssetIDs"`
	ProductName      string   `json:"productName"`
	Status           string   `json:"status"`
	OriginalQuantity Quantity `json:"originalQuantity"`
	CurrentQuantity  Quantity `json:"currentQuantity"`
	History          []Event  `json:"history"`
}

type ItemInShipment struct {
	AssetID      string   `json:"assetID"`
	Quantity     Quantity `json:"quantity"`
	ToFacilityID string   `json:"toFacilityID"`
	Status       string   `json:"status"`
}

type ShipmentAsset struct {
	ObjectType     string             `json:"docType"`
	ShipmentID     string             `json:"shipmentID"`
	CarrierOrgName string             `json:"carrierOrgName"`
	DriverName     string             `json:"driverName"`
	VehiclePlate   string             `json:"vehiclePlate"`
	FromFacilityID string             `json:"fromFacilityID"`
	Status         string             `json:"status"`
	Items          []ItemInShipment   `json:"items"`
	Timeline       []ShipmentTimeline `json:"timeline"`
	History        []Event            `json:"history"`
}

type ChildAssetInput struct {
	AssetID     string   `json:"assetID"`
	ProductName string   `json:"productName"`
	Quantity    Quantity `json:"quantity"`
}

type FullAssetTrace struct {
	AssetID          string   `json:"assetID"`
	ParentAssetIDs   []string `json:"parentAssetIDs"`
	ProductName      string   `json:"productName"`
	Status           string   `json:"status"`
	OriginalQuantity Quantity `json:"originalQuantity"`
	CurrentQuantity  Quantity `json:"currentQuantity"`
	FullHistory      []Event  `json:"fullHistory"`
}

// ===============================================================
// CÁC HÀM CHAINCODE (TRANSACTIONS)
// ===============================================================

func (s *SmartContract) CreateFarmingBatch(ctx contractapi.TransactionContextInterface, assetID string, productName string, quantityJSON string, farmDetailsJSON string) error {
	exists, err := s.assetExists(ctx, assetID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("asset %s already exists", assetID)
	}

	var quantity Quantity
	if err := json.Unmarshal([]byte(quantityJSON), &quantity); err != nil {
		return fmt.Errorf("failed to unmarshal quantityJSON: %v", err)
	}

	var farmDetails FarmDetails
	if err := json.Unmarshal([]byte(farmDetailsJSON), &farmDetails); err != nil {
		return fmt.Errorf("failed to unmarshal farmDetailsJSON: %v", err)
	}

	event, err := s.createEvent(ctx, "FARMING", farmDetails)
	if err != nil {
		return err
	}

	asset := MeatAsset{
		ObjectType:       "MeatAsset",
		AssetID:          assetID,
		ParentAssetIDs:   []string{},
		ProductName:      productName,
		Status:           "AT_FARM",
		OriginalQuantity: quantity,
		CurrentQuantity:  quantity,
		History:          []Event{*event},
	}

	return s.updateAsset(ctx, &asset)
}

func (s *SmartContract) CreateShipment(ctx contractapi.TransactionContextInterface, shipmentID string, carrierOrgName, driverName, vehiclePlate, fromFacilityID string) error {
	exists, err := s.assetExists(ctx, shipmentID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("shipment %s already exists", shipmentID)
	}

	event, err := s.createEvent(ctx, "SHIPMENT_CREATED", "Shipment created and pending item loading.")
	if err != nil {
		return err
	}

	shipment := ShipmentAsset{
		ObjectType:     "ShipmentAsset",
		ShipmentID:     shipmentID,
		CarrierOrgName: carrierOrgName,
		DriverName:     driverName,
		VehiclePlate:   vehiclePlate,
		FromFacilityID: fromFacilityID,
		Status:         "PENDING",
		Items:          []ItemInShipment{},
		History:        []Event{*event},
	}

	return s.updateShipment(ctx, &shipment)
}

// LoadItemToShipment loads a specific quantity of an asset onto a shipment.
// This function can be called multiple times for a shipment in PENDING state.
func (s *SmartContract) LoadItemToShipment(ctx contractapi.TransactionContextInterface, shipmentID string, assetID string, quantityJSON string, toFacilityID string) error {
	// TODO: Add access control logic (e.g., only the shipment's carrier can load items).

	shipment, err := s.readShipmentAsset(ctx, shipmentID)
	if err != nil {
		return err
	}
	// Logic kiểm tra cốt lõi: Chỉ có thể thêm hàng khi chuyến xe đang chờ.
	if shipment.Status != "PENDING" {
		return fmt.Errorf("shipment %s is not in PENDING state, cannot load more items", shipmentID)
	}

	asset, err := s.readAsset(ctx, assetID)
	if err != nil {
		return err
	}

	var quantityToLoad Quantity
	if err := json.Unmarshal([]byte(quantityJSON), &quantityToLoad); err != nil {
		return fmt.Errorf("failed to unmarshal quantityJSON: %v", err)
	}

	if asset.CurrentQuantity.Unit != quantityToLoad.Unit {
		return fmt.Errorf("quantity unit mismatch for asset %s", assetID)
	}
	if asset.CurrentQuantity.Value < quantityToLoad.Value {
		return fmt.Errorf("insufficient quantity for asset %s. Available: %f, Requested: %f", assetID, asset.CurrentQuantity.Value, quantityToLoad.Value)
	}

	// Trừ số lượng khỏi lô gốc
	asset.CurrentQuantity.Value -= quantityToLoad.Value

	// Chỉ ghi lại một sự kiện "đã được phân bổ", không thay đổi trạng thái chính của lô hàng
	eventDetails := map[string]interface{}{
		"shipmentID": shipmentID,
		"quantity":   quantityToLoad,
	}
	// Trạng thái của asset vẫn giữ nguyên cho đến khi chuyến xe thực sự bắt đầu
	err = s.addEvent(ctx, asset, "ALLOCATED_FOR_SHIPMENT", asset.Status, eventDetails)
	if err != nil {
		return err
	}

	// Thêm item vào chuyến xe
	item := ItemInShipment{
		AssetID:      assetID,
		Quantity:     quantityToLoad,
		ToFacilityID: toFacilityID,
		Status:       "LOADED",
	}
	shipment.Items = append(shipment.Items, item)

	// Lưu lại ShipmentAsset, LƯU Ý: KHÔNG THAY ĐỔI TRẠNG THÁI CỦA NÓ
	return s.updateShipment(ctx, shipment)
}

// StartShipment: Tài xế xác nhận bắt đầu chuyến đi.
func (s *SmartContract) StartShipment(ctx contractapi.TransactionContextInterface, shipmentID string) error {
	// TODO: Phân quyền (chỉ tài xế của chuyến hàng)
	shipment, err := s.readShipmentAsset(ctx, shipmentID)
	if err != nil {
		return err
	}
	if shipment.Status != "PENDING" {
		return fmt.Errorf("shipment %s has already started or is completed", shipmentID)
	}
	if len(shipment.Items) == 0 {
		return fmt.Errorf("cannot start an empty shipment %s", shipmentID)
	}

	// Cập nhật trạng thái của tất cả các MeatAsset trong chuyến hàng
	for _, item := range shipment.Items {
		asset, err := s.readAsset(ctx, item.AssetID)
		if err != nil {
			// Ghi log và bỏ qua nếu không tìm thấy asset, để không làm hỏng toàn bộ giao dịch
			fmt.Printf("Warning: could not read asset %s while starting shipment %s: %v\n", item.AssetID, shipmentID, err)
			continue
		}

		var newStatus string
		if asset.CurrentQuantity.Value > 0 {
			newStatus = "PARTIALLY_SHIPPED"
		} else {
			newStatus = "SHIPPED_FULL"
		}

		eventDetails := map[string]string{"shipmentID": shipmentID}
		// Thêm sự kiện và cập nhật trạng thái cho MeatAsset
		s.addEvent(ctx, asset, "SHIPPING_STARTED", newStatus, eventDetails)
	}

	// Cập nhật trạng thái và timeline của chuyến xe
	shipment.Status = "IN_TRANSIT"
	timelineEvent := ShipmentTimeline{
		Type:      "pickup",
		Timestamp: s.getTxTimestamp(ctx),
	}
	shipment.Timeline = append(shipment.Timeline, timelineEvent)

	return s.updateShipment(ctx, shipment)
}

// ConfirmShipmentDelivery: Xác nhận giao hàng và tạo lô con với trạng thái được chỉ định.
func (s *SmartContract) ConfirmShipmentDelivery(ctx contractapi.TransactionContextInterface, shipmentID string, facilityID string, newAssetID string, newStatus string) error {
    // TODO: Phân quyền

    shipment, err := s.readShipmentAsset(ctx, shipmentID)
    if err != nil { return err }
    if shipment.Status != "IN_TRANSIT" {
        return fmt.Errorf("shipment %s is not in transit", shipmentID)
    }

    exists, err := s.assetExists(ctx, newAssetID)
    if err != nil { return err }
    if exists { return fmt.Errorf("new asset ID %s already exists", newAssetID) }

    // ... (logic tìm items for this facility và parentAssetID giữ nguyên) ...
    var itemsForThisFacility []ItemInShipment
    var parentAssetID string
    for i, item := range shipment.Items {
        if item.ToFacilityID == facilityID && item.Status != "DELIVERED" {
            itemsForThisFacility = append(itemsForThisFacility, item)
            shipment.Items[i].Status = "DELIVERED"
            if parentAssetID == "" { parentAssetID = item.AssetID }
        }
    }
    if len(itemsForThisFacility) == 0 {
        return fmt.Errorf("no undelivered items found for facility %s in shipment %s", facilityID, shipmentID)
    }

    parentAsset, err := s.readAsset(ctx, parentAssetID)
    if err != nil { return err }

    totalReceivedQuantity := Quantity{Unit: itemsForThisFacility[0].Quantity.Unit, Value: 0}
    for _, item := range itemsForThisFacility {
        totalReceivedQuantity.Value += item.Quantity.Value
    }

    // === SỬA LỖI LOGIC: Không còn đoán trạng thái, nhận trực tiếp từ tham số ===
    // Kiểm tra xem newStatus có hợp lệ không (tùy chọn nhưng nên có)
    validStatuses := map[string]bool{
        "RECEIVED_AT_PROCESSOR": true,
        "STORED_AT_WAREHOUSE":   true,
        "AT_RETAILER":           true,
    }
    if !validStatuses[newStatus] {
        return fmt.Errorf("invalid newStatus provided: %s", newStatus)
    }

    receivingEventDetails := map[string]interface{}{
        "shipmentID":       shipmentID,
        "receivedFacility": facilityID,
        "quantityReceived": totalReceivedQuantity,
    }
    event, err := s.createEvent(ctx, "RECEIVING", receivingEventDetails)
    if err != nil { return err }

    newAsset := MeatAsset{
        ObjectType:       "MeatAsset",
        AssetID:          newAssetID,
        ParentAssetIDs:   []string{parentAssetID},
        ProductName:      parentAsset.ProductName,
        Status:           newStatus, // Sử dụng trạng thái do client cung cấp
        OriginalQuantity: totalReceivedQuantity,
        CurrentQuantity:  totalReceivedQuantity,
        History:          []Event{*event},
    }
    err = s.updateAsset(ctx, &newAsset)
    if err != nil { return err }

    // ... (logic cập nhật trạng thái shipment thành COMPLETED giữ nguyên) ...
    allDelivered := true
    for _, item := range shipment.Items {
        if item.Status != "DELIVERED" {
            allDelivered = false
            break
        }
    }
    if allDelivered {
        shipment.Status = "COMPLETED"
    }

    return s.updateShipment(ctx, shipment)
}

func (s *SmartContract) ProcessAndSplitBatch(ctx contractapi.TransactionContextInterface, parentAssetID string, childAssetsJSON string, processingDetailsJSON string) error {
	parentAsset, err := s.readAsset(ctx, parentAssetID)
	if err != nil {
		return err
	}

	var processingDetails ProcessingDetails
	if err := json.Unmarshal([]byte(processingDetailsJSON), &processingDetails); err != nil {
		return fmt.Errorf("failed to unmarshal processingDetailsJSON: %v", err)
	}

	err = s.addEvent(ctx, parentAsset, "PROCESSING", "PROCESSED_AND_SPLIT", processingDetails)
	if err != nil {
		return err
	}

	var childAssets []ChildAssetInput
	if err := json.Unmarshal([]byte(childAssetsJSON), &childAssets); err != nil {
		return fmt.Errorf("failed to unmarshal childAssetsJSON: %v", err)
	}

	clientMSP, _ := ctx.GetClientIdentity().GetMSPID()
	txID := ctx.GetStub().GetTxID()
	timestamp, _ := ctx.GetStub().GetTxTimestamp()
	formattedTime := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).Format(time.RFC3339)

	for _, child := range childAssets {
		exists, err := s.assetExists(ctx, child.AssetID)
		if err != nil {
			return err
		}
		if exists {
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
			ObjectType:       "MeatAsset",
			AssetID:          child.AssetID,
			ParentAssetIDs:   []string{parentAssetID},
			ProductName:      child.ProductName,
			Status:           "PACKAGED",
			OriginalQuantity: child.Quantity,
			CurrentQuantity:  child.Quantity,
			History:          []Event{creationEvent},
		}
		err = s.updateAsset(ctx, &newChildAsset)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateFarmingDetails allows a farm to add or update details of a batch before it's shipped.
func (s *SmartContract) UpdateFarmingDetails(ctx contractapi.TransactionContextInterface, assetID string, updatedFarmDetailsJSON string) error {
    // TODO: Add access control logic (only the farm that owns the asset can update).

    asset, err := s.readAsset(ctx, assetID)
    if err != nil {
        return err
    }

    // A farm can only update details while the asset is still at the farm.
    if asset.Status != "AT_FARM" {
        return fmt.Errorf("asset %s with status '%s' cannot be updated by the farm", assetID, asset.Status)
    }

    var updatedFarmDetails FarmDetails
    if err := json.Unmarshal([]byte(updatedFarmDetailsJSON), &updatedFarmDetails); err != nil {
        return fmt.Errorf("failed to unmarshal updatedFarmDetailsJSON: %v", err)
    }

    // Find the original FARMING event and update its details.
    // This is better than creating a new event to avoid cluttering the history.
    updated := false
    for i, event := range asset.History {
        if event.Type == "FARMING" {
            asset.History[i].Details = updatedFarmDetails
            updated = true
            break
        }
    }

    if !updated {
        return fmt.Errorf("could not find the original FARMING event to update for asset %s", assetID)
    }

    return s.updateAsset(ctx, asset)
}

// SỬA LỖI CÚ PHÁP: Thêm lại các hàm đã mất
func (s *SmartContract) UpdateStorageInfo(ctx contractapi.TransactionContextInterface, assetID string, storageDetailsJSON string) error {
	asset, err := s.readAsset(ctx, assetID)
	if err != nil {
		return err
	}

	var storageDetails StorageDetails
	if err := json.Unmarshal([]byte(storageDetailsJSON), &storageDetails); err != nil {
		return fmt.Errorf("failed to unmarshal storageDetailsJSON: %v", err)
	}

	return s.addEvent(ctx, asset, "STORAGE_UPDATE", asset.Status, storageDetails)
}

// SplitBatchToUnits creates a specified number of individual retail units from a product batch.
func (s *SmartContract) SplitBatchToUnits(ctx contractapi.TransactionContextInterface, parentAssetID string, unitCount int, unitIDPrefix string) error {
    // TODO: Phân quyền (chỉ Retailer)

    parentAsset, err := s.readAsset(ctx, parentAssetID)
    if err != nil {
        return err
    }

    if parentAsset.Status != "AT_RETAILER" {
        return fmt.Errorf("asset %s with status '%s' cannot be split into units", parentAssetID, parentAsset.Status)
    }

    // Kiểm tra số lượng
    if float64(unitCount) > parentAsset.CurrentQuantity.Value {
        return fmt.Errorf("unit count (%d) exceeds parent batch quantity (%f)", unitCount, parentAsset.CurrentQuantity.Value)
    }

    clientMSP, _ := ctx.GetClientIdentity().GetMSPID()
    txID := ctx.GetStub().GetTxID()
    timestamp, _ := ctx.GetStub().GetTxTimestamp()
    formattedTime := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).Format(time.RFC3339)

    // Lặp và tạo ra các unit con
    for i := 1; i <= unitCount; i++ {
        // Tự động tạo ID cho unit mới
        unitAssetID := fmt.Sprintf("%s%d", unitIDPrefix, i)

        exists, err := s.assetExists(ctx, unitAssetID)
        if err != nil {
            return err
        }
        if exists {
            // Nếu ID đã tồn tại, có thể bỏ qua hoặc báo lỗi. Báo lỗi an toàn hơn.
            return fmt.Errorf("unit asset %s already exists", unitAssetID)
        }

        creationEvent := Event{
            Type:      "CREATED_AS_UNIT",
            ActorMSP:  clientMSP,
            Timestamp: formattedTime,
            TxID:      txID,
            Details:   fmt.Sprintf("Split from product batch %s", parentAssetID),
        }
        
        unitHistory := append(parentAsset.History, creationEvent)

        // Mỗi unit có số lượng là 1
        unitQuantity := Quantity{
            Unit:  parentAsset.OriginalQuantity.Unit, // Kế thừa đơn vị của cha
            Value: 1,
        }

        newUnitAsset := MeatAsset{
            ObjectType:       "MeatAsset",
            AssetID:          unitAssetID,
            ParentAssetIDs:   []string{parentAssetID},
            ProductName:      parentAsset.ProductName, // Kế thừa tên sản phẩm
            Status:           "ON_SHELF",
            OriginalQuantity: unitQuantity,
            CurrentQuantity:  unitQuantity,
            History:          unitHistory,
        }
        err = s.updateAsset(ctx, &newUnitAsset)
        if err != nil {
            return err
        }
    }

    // Cập nhật lô cha
    parentAsset.CurrentQuantity.Value -= float64(unitCount)
    if parentAsset.CurrentQuantity.Value == 0 {
        parentAsset.Status = "SPLIT_INTO_UNITS_FULL"
    } else {
        parentAsset.Status = "SPLIT_INTO_UNITS_PARTIAL"
    }
    
    err = s.updateAsset(ctx, parentAsset)
    if err != nil {
        return err
    }

    return nil
}


func (s *SmartContract) MarkAsSold(ctx contractapi.TransactionContextInterface, assetID string, soldDetailsJSON string) error {
	asset, err := s.readAsset(ctx, assetID)
	if err != nil {
		return err
	}

	if asset.Status != "ON_SHELF" {
		return fmt.Errorf("asset %s with status '%s' cannot be sold", assetID, asset.Status)
	}

	var soldDetails SoldDetails
	if err := json.Unmarshal([]byte(soldDetailsJSON), &soldDetails); err != nil {
		return fmt.Errorf("failed to unmarshal soldDetailsJSON: %v", err)
	}

	return s.addEvent(ctx, asset, "SOLD", "SOLD", soldDetails)
}

// ===============================================================
// CÁC HÀM QUERY (READ-ONLY)
// ===============================================================

func (s *SmartContract) GetAssetWithFullHistory(ctx contractapi.TransactionContextInterface, assetID string) (*FullAssetTrace, error) {
	asset, err := s.readAsset(ctx, assetID)
	if err != nil {
		return nil, err
	}

	fullHistory, err := s.getAssetHistoryRecursive(ctx, assetID)
	if err != nil {
		return nil, err
	}

	traceResult := FullAssetTrace{
		AssetID:          asset.AssetID,
		ParentAssetIDs:   asset.ParentAssetIDs,
		ProductName:      asset.ProductName,
		Status:           asset.Status,
		OriginalQuantity: asset.OriginalQuantity,
		CurrentQuantity:  asset.CurrentQuantity,
		FullHistory:      fullHistory,
	}

	return &traceResult, nil
}

// ===============================================================
// CÁC HÀM TIỆN ÍCH (PRIVATE)
// ===============================================================

func (s *SmartContract) addEvent(ctx contractapi.TransactionContextInterface, asset *MeatAsset, eventType string, newStatus string, details interface{}) error {
	event, err := s.createEvent(ctx, eventType, details)
	if err != nil {
		return err
	}
	asset.History = append(asset.History, *event)
	asset.Status = newStatus
	return s.updateAsset(ctx, asset)
}

func (s *SmartContract) createEvent(ctx contractapi.TransactionContextInterface, eventType string, details interface{}) (*Event, error) {
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return nil, err
	}
	txID := ctx.GetStub().GetTxID()
	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, err
	}

	event := Event{
		Type:      eventType,
		ActorMSP:  clientMSP,
		Timestamp: time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).Format(time.RFC3339),
		TxID:      txID,
		Details:   details,
	}
	return &event, nil
}

func (s *SmartContract) updateAsset(ctx contractapi.TransactionContextInterface, asset *MeatAsset) error {
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(asset.AssetID, assetJSON)
}

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

func (s *SmartContract) updateShipment(ctx contractapi.TransactionContextInterface, shipment *ShipmentAsset) error {
	shipmentJSON, err := json.Marshal(shipment)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(shipment.ShipmentID, shipmentJSON)
}

func (s *SmartContract) readShipmentAsset(ctx contractapi.TransactionContextInterface, shipmentID string) (*ShipmentAsset, error) {
	shipmentJSON, err := ctx.GetStub().GetState(shipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if shipmentJSON == nil {
		return nil, fmt.Errorf("the shipment %s does not exist", shipmentID)
	}

	var shipment ShipmentAsset
	err = json.Unmarshal(shipmentJSON, &shipment)
	if err != nil {
		return nil, err
	}
	return &shipment, nil
}

func (s *SmartContract) assetExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return assetJSON != nil, nil
}

func (s *SmartContract) getTxTimestamp(ctx contractapi.TransactionContextInterface) string {
	ts, _ := ctx.GetStub().GetTxTimestamp()
	return time.Unix(ts.Seconds, int64(ts.Nanos)).Format(time.RFC3339)
}

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