// meatcc/contract.go
package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract cung cấp các hàm quản lý sản phẩm thịt và lô vận chuyển.
type SmartContract struct {
	contractapi.Contract
}

// Lấy lịch sử đầy đủ của một asset, bao gồm cả các asset cha.
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

// --- Các hàm hỗ trợ nội bộ ---

// Thêm một sự kiện vào asset, cập nhật trạng thái mới và lưu lại asset.
func (s *SmartContract) addEvent(ctx contractapi.TransactionContextInterface, asset *MeatAsset, eventType string, newStatus string, details interface{}) error {
	event, err := s.createEvent(ctx, eventType, details)
	if err != nil {
		return err
	}
	asset.History = append(asset.History, *event)
	asset.Status = newStatus
	return s.updateAsset(ctx, asset)
}

// Tạo một sự kiện mới với thông tin người thực hiện, thời gian, chi tiết.
func (s *SmartContract) createEvent(ctx contractapi.TransactionContextInterface, eventType string, details interface{}) (*Event, error) {
	clientMSP, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return nil, err
	}
	enrollmentID, err := getEnrollmentID(ctx)
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
		ActorID:   enrollmentID,
		Timestamp: time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).Format(time.RFC3339),
		TxID:      txID,
		Details:   details,
	}
	return &event, nil
}

// Lưu asset vào world state.
func (s *SmartContract) updateAsset(ctx contractapi.TransactionContextInterface, asset *MeatAsset) error {
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(asset.AssetID, assetJSON)
}

// Đọc thông tin asset từ world state.
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

// Lưu shipment vào world state.
func (s *SmartContract) updateShipment(ctx contractapi.TransactionContextInterface, shipment *ShipmentAsset) error {
	shipmentJSON, err := json.Marshal(shipment)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(shipment.ShipmentID, shipmentJSON)
}

// Đọc thông tin shipment từ world state.
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

// Kiểm tra asset có tồn tại trong world state không.
func (s *SmartContract) assetExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return assetJSON != nil, nil
}

// Lấy timestamp của transaction hiện tại.
func (s *SmartContract) getTxTimestamp(ctx contractapi.TransactionContextInterface) string {
	ts, _ := ctx.GetStub().GetTxTimestamp()
	return time.Unix(ts.Seconds, int64(ts.Nanos)).Format(time.RFC3339)
}

// Lấy lịch sử các sự kiện của asset và các asset cha (truy xuất đệ quy).
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