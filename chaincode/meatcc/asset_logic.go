package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Tạo một lô thịt mới tại trang trại, lưu thông tin số lượng và chi tiết trang trại, đồng thời ghi lại sự kiện FARMING.
func (s *SmartContract) CreateFarmingBatch(ctx contractapi.TransactionContextInterface, assetID string, productName string, quantityJSON string, farmDetailsJSON string) error {
	if err := requireRole(ctx, "admin", "worker"); err != nil {
		return err
	}
	callerOrg, _, _ := ctx.GetClientIdentity().GetAttributeValue("facilityID")

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
		OwnerOrg:         callerOrg,
		OriginalQuantity: quantity,
		CurrentQuantity:  quantity,
		History:          []Event{*event},
	}

	return s.updateAsset(ctx, &asset)
}

// Xử lý và tách một lô thịt thành nhiều lô con, cập nhật sự kiện PROCESSING cho lô cha và tạo các lô con mới.
func (s *SmartContract) ProcessAndSplitBatch(ctx contractapi.TransactionContextInterface, parentAssetID string, childAssetsJSON string, processingDetailsJSON string) error {
	if err := requireRole(ctx, "admin", "worker"); err != nil {
		return err
	}
	
	parentAsset, err := s.readAsset(ctx, parentAssetID)
	if err != nil {
		return err
	}

	if parentAsset.Status != "AT_PROCESSOR" {
		return fmt.Errorf("asset %s with status '%s' cannot be split into units", parentAssetID, parentAsset.Status)
	}

	if err := requireOwnership(ctx, parentAsset); err != nil {
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

	for _, child := range childAssets {
		exists, err := s.assetExists(ctx, child.AssetID)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("child asset %s already exists", child.AssetID)
		}

		details := fmt.Sprintf("Created from parent batch %s", parentAssetID)
		creationEvent, err := s.createEvent(ctx, "CREATED_FROM_PROCESSING", details)
		if err != nil {
			return fmt.Errorf("failed to create event for child asset %s: %v", child.AssetID, err)
		}

		newChildAsset := MeatAsset{
			ObjectType:       "MeatAsset",
			AssetID:          child.AssetID,
			ParentAssetIDs:   []string{parentAssetID},
			ProductName:      child.ProductName,
			Status:           "PACKAGED",
			OwnerOrg:         parentAsset.OwnerOrg,
			OriginalQuantity: child.Quantity,
			CurrentQuantity:  child.Quantity,
			History:          []Event{*creationEvent},
		}
		err = s.updateAsset(ctx, &newChildAsset)
		if err != nil {
			return err
		}
	}

	return nil
}

// Cập nhật thông tin trang trại cho một lô thịt đang ở trạng thái AT_FARM, chỉnh sửa sự kiện FARMING trong lịch sử.
func (s *SmartContract) UpdateFarmingDetails(ctx contractapi.TransactionContextInterface, assetID string, updatedFarmDetailsJSON string) error {
	if err := requireRole(ctx, "admin", "worker"); err != nil {
		return err
	}
	asset, err := s.readAsset(ctx, assetID)
	if err != nil {
		return err
	}
	if err := requireOwnership(ctx, asset); err != nil {
		return err
	}
	if asset.Status != "AT_FARM" {
		return fmt.Errorf("asset %s with status '%s' cannot be updated by the farm", assetID, asset.Status)
	}

	// === LOGIC MERGE MỚI ===

	// 1. Unmarshal các chi tiết MỚI từ request vào một map
	var newDetails map[string]interface{}
	if err := json.Unmarshal([]byte(updatedFarmDetailsJSON), &newDetails); err != nil {
		return fmt.Errorf("failed to unmarshal updatedFarmDetailsJSON: %v", err)
	}

	updated := false
	for i, event := range asset.History {
		if event.Type == "FARMING" {
			// 2. Chuyển đổi chi tiết CŨ (là interface{}) thành một map
			existingDetails, ok := event.Details.(map[string]interface{})
			if !ok {
				return fmt.Errorf("could not parse existing farming details for asset %s", assetID)
			}

			// 3. Lặp qua các chi tiết MỚI và cập nhật/thêm vào chi tiết CŨ
			for key, value := range newDetails {
				existingDetails[key] = value
			}

			// 4. Gán lại map đã được hợp nhất vào lịch sử
			asset.History[i].Details = existingDetails
			updated = true
			break
		}
	}
	if !updated {
		return fmt.Errorf("could not find the original FARMING event to update for asset %s", assetID)
	}
	// ========================

	return s.updateAsset(ctx, asset)
}

// Cập nhật thông tin lưu kho cho một lô thịt, thêm sự kiện STORAGE_UPDATE vào lịch sử asset.
func (s *SmartContract) UpdateStorageInfo(ctx contractapi.TransactionContextInterface, assetID string, storageDetailsJSON string) error {
	if err := requireRole(ctx, "admin", "worker"); err != nil {
		return err
	}
	asset, err := s.readAsset(ctx, assetID)
	if err != nil {
		return err
	}
	if err := requireOwnership(ctx, asset); err != nil {
		return err
	}

	var storageDetails StorageDetails
	if err := json.Unmarshal([]byte(storageDetailsJSON), &storageDetails); err != nil {
		return fmt.Errorf("failed to unmarshal storageDetailsJSON: %v", err)
	}

	return s.addEvent(ctx, asset, "STORAGE_UPDATE", asset.Status, storageDetails)
}

// Tách một lô thịt tại nhà bán lẻ thành các đơn vị nhỏ hơn, tạo các asset mới cho từng đơn vị và cập nhật sự kiện SPLIT_INTO_UNITS.
func (s *SmartContract) SplitBatchToUnits(ctx contractapi.TransactionContextInterface, parentAssetID string, unitCount int, unitIDPrefix string) error {
	if err := requireRole(ctx, "admin", "worker"); err != nil {
		return err
	}
	parentAsset, err := s.readAsset(ctx, parentAssetID)
	if err != nil {
		return err
	}
	if err := requireOwnership(ctx, parentAsset); err != nil {
		return err
	}
	if parentAsset.Status != "AT_RETAILER" {
		return fmt.Errorf("asset %s with status '%s' cannot be split into units", parentAssetID, parentAsset.Status)
	}
	if float64(unitCount) > parentAsset.CurrentQuantity.Value {
		return fmt.Errorf("unit count (%d) exceeds parent batch quantity (%f)", unitCount, parentAsset.CurrentQuantity.Value)
	}

	for i := 1; i <= unitCount; i++ {
		unitAssetID := fmt.Sprintf("%s%d", unitIDPrefix, i)
		exists, err := s.assetExists(ctx, unitAssetID)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("unit asset %s already exists", unitAssetID)
		}

		details := fmt.Sprintf("Split from product batch %s", parentAssetID)
		creationEvent, err := s.createEvent(ctx, "CREATED_AS_UNIT", details)
		if err != nil {
			return fmt.Errorf("failed to create event for unit asset %s: %v", unitAssetID, err)
		}

		unitQuantity := Quantity{
			Unit:  parentAsset.OriginalQuantity.Unit,
			Value: 1,
		}

		newUnitAsset := MeatAsset{
			ObjectType:       "MeatAsset",
			AssetID:          unitAssetID,
			ParentAssetIDs:   []string{parentAssetID},
			ProductName:      parentAsset.ProductName,
			Status:           "ON_SHELF",
			OwnerOrg:         parentAsset.OwnerOrg,
			OriginalQuantity: unitQuantity,
			CurrentQuantity:  unitQuantity,
			History:          []Event{*creationEvent},
		}
		err = s.updateAsset(ctx, &newUnitAsset)
		if err != nil {
			return err
		}
	}

	parentAsset.CurrentQuantity.Value -= float64(unitCount)
	splitEventDetails := map[string]interface{}{
		"unitCount":    unitCount,
		"unitIDPrefix": unitIDPrefix,
	}
	return s.addEvent(ctx, parentAsset, "SPLIT_INTO_UNITS", "SPLIT_INTO_UNITS_COMPLETED", splitEventDetails)
}

// Đánh dấu một đơn vị thịt đã được bán, thêm sự kiện SOLD vào lịch sử asset.
func (s *SmartContract) MarkAsSold(ctx contractapi.TransactionContextInterface, assetID string, soldDetailsJSON string) error {
	if err := requireRole(ctx, "admin", "worker"); err != nil {
		return err
	}
	asset, err := s.readAsset(ctx, assetID)
	if err != nil {
		return err
	}
	if err := requireOwnership(ctx, asset); err != nil {
		return err
	}
	if asset.Status != "ON_SHELF" {
		return fmt.Errorf("asset %s with status '%s' cannot be sold", assetID, asset.Status)
	}

	var soldDetails map[string]interface{}
	if err := json.Unmarshal([]byte(soldDetailsJSON), &soldDetails); err != nil {
		return fmt.Errorf("failed to unmarshal soldDetailsJSON: %v", err)
	}

	txTimestamp := s.getTxTimestamp(ctx)

	soldDetails["saleTimestamp"] = txTimestamp

	return s.addEvent(ctx, asset, "SOLD", "SOLD", soldDetails)
}