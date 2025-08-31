package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Tạo một lô vận chuyển mới, lưu thông tin tài xế, phương tiện, các điểm dừng và ghi lại sự kiện khởi tạo shipment.
func (s *SmartContract) CreateShipment(ctx contractapi.TransactionContextInterface, shipmentID string, shipmentType, driverEnrollmentID, driverName, vehiclePlate, fromFacilityID string, stopsJSON string) error {
	if err := requireRole(ctx, "admin", "worker"); err != nil {
		return err
	}
	exists, err := s.assetExists(ctx, shipmentID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("shipment %s already exists", shipmentID)
	}

	var stops []StopInJourney
	if err := json.Unmarshal([]byte(stopsJSON), &stops); err != nil {
		return fmt.Errorf("failed to unmarshal stopsJSON: %v", err)
	}

	event, err := s.createEvent(ctx, "SHIPMENT_CREATED", "Shipment created and pending.")
	if err != nil {
		return err
	}

	shipment := ShipmentAsset{
		ObjectType:         "ShipmentAsset",
		ShipmentID:         shipmentID,
		ShipmentType:       shipmentType,
		DriverEnrollmentID: driverEnrollmentID,
		DriverName:         driverName,
		VehiclePlate:       vehiclePlate,
		Status:             "PENDING",
		Stops:              stops,
		History:            []Event{*event},
	}

	return s.updateShipment(ctx, &shipment)
}

// Xác nhận việc lấy hàng tại một điểm dừng, cập nhật số lượng asset và trạng thái điểm dừng thành COMPLETED.
func (s *SmartContract) ConfirmPickup(ctx contractapi.TransactionContextInterface, shipmentID string, facilityID string, actualItemsJSON string) error {
	shipment, err := s.readShipmentAsset(ctx, shipmentID)
	if err != nil {
		return err
	}
	if shipment.Status != "PENDING" {
		return fmt.Errorf("shipment %s is not in PENDING state", shipmentID)
	}

	var actualItems []ItemInShipment
	if err := json.Unmarshal([]byte(actualItemsJSON), &actualItems); err != nil {
		return fmt.Errorf("failed to unmarshal actualItemsJSON: %v", err)
	}

	stopFound := false
	for i, stop := range shipment.Stops {
		if stop.FacilityID == facilityID && stop.Action == "PICKUP" {
			if err := requireRole(ctx, "admin", "worker"); err != nil {
				return err
			}

			for _, actualItem := range actualItems {
				asset, err := s.readAsset(ctx, actualItem.AssetID)
				if err != nil {
					return err
				}
				if err := requireOwnership(ctx, asset); err != nil {
					return err
				}
				if asset.CurrentQuantity.Value < actualItem.Quantity.Value {
					return fmt.Errorf("insufficient quantity for asset %s", actualItem.AssetID)
				}
				asset.CurrentQuantity.Value -= actualItem.Quantity.Value
				err = s.addEvent(ctx, asset, "PICKED_UP_FOR_SHIPMENT", asset.Status, map[string]interface{}{"shipmentID": shipmentID, "quantity": actualItem.Quantity})
				if err != nil {
					return err
				}
			}
			shipment.Stops[i].Items = actualItems
			shipment.Stops[i].Status = "COMPLETED"
			stopFound = true
			break
		}
	}
	if !stopFound {
		return fmt.Errorf("no pending pickup stop found for facility %s", facilityID)
	}

	return s.updateShipment(ctx, shipment)
}

// Bắt đầu quá trình vận chuyển, cập nhật trạng thái shipment thành IN_TRANSIT và ghi lại sự kiện khởi hành.
func (s *SmartContract) StartShipment(ctx contractapi.TransactionContextInterface, shipmentID string) error {
	shipment, err := s.readShipmentAsset(ctx, shipmentID)
	if err != nil {
		return err
	}
	if err := requireAssignedDriver(ctx, shipment); err != nil {
		return err
	}
	if shipment.Status != "PENDING" {
		return fmt.Errorf("shipment %s has already started or is completed", shipmentID)
	}

	for _, stop := range shipment.Stops {
		if stop.Action == "PICKUP" && stop.Status == "COMPLETED" {
			for _, item := range stop.Items {
				asset, err := s.readAsset(ctx, item.AssetID)
				if err != nil {
					continue
				}
				var newStatus string
				if asset.CurrentQuantity.Value > 0 {
					newStatus = "PARTIALLY_SHIPPED"
				} else {
					newStatus = "SHIPPED_FULL"
				}
				s.addEvent(ctx, asset, "SHIPPING_STARTED", newStatus, map[string]string{"shipmentID": shipmentID})
			}
		}
	}

	shipment.Status = "IN_TRANSIT"
	timelineEvent := ShipmentTimeline{
		Type:      "departure",
		Timestamp: s.getTxTimestamp(ctx),
	}
	shipment.Timeline = append(shipment.Timeline, timelineEvent)

	return s.updateShipment(ctx, shipment)
}

// Xác nhận việc giao hàng tại một điểm dừng, tạo asset mới cho bên nhận và cập nhật trạng thái shipment nếu đã giao hết.
func (s *SmartContract) ConfirmShipmentDelivery(ctx contractapi.TransactionContextInterface, shipmentID string, facilityID string, newAssetIDPrefix string) error {
	if err := requireRole(ctx, "admin", "worker"); err != nil {
		return err
	}
	receiverOrg, _, _ := ctx.GetClientIdentity().GetAttributeValue("orgShortName")

	shipment, err := s.readShipmentAsset(ctx, shipmentID)
	if err != nil {
		return err
	}
	if shipment.Status != "IN_TRANSIT" {
		return fmt.Errorf("shipment %s is not in transit", shipmentID)
	}

	stopFound := false
	for i, stop := range shipment.Stops {
		if stop.FacilityID == facilityID && stop.Action == "DELIVERY" && stop.Status == "PENDING" {
			stopFound = true
			shipment.Stops[i].Status = "COMPLETED"

			for j, item := range stop.Items {
				parentAsset, err := s.readAsset(ctx, item.AssetID)
				if err != nil {
					return err
				}

				var newStatus string
				if receiverOrg == "retailer" { // Example logic
					newStatus = "AT_RETAILER"
				} else {
					newStatus = "RECEIVED"
				}

				newAssetID := fmt.Sprintf("%s-%d", newAssetIDPrefix, j)
				event, err := s.createEvent(ctx, "RECEIVING", map[string]interface{}{"shipmentID": shipmentID, "quantityReceived": item.Quantity})
				if err != nil {
					return err
				}

				newAsset := MeatAsset{
					ObjectType:       "MeatAsset",
					AssetID:          newAssetID,
					ParentAssetIDs:   []string{item.AssetID},
					ProductName:      parentAsset.ProductName,
					Status:           newStatus,
					OwnerOrg:         receiverOrg,
					OriginalQuantity: item.Quantity,
					CurrentQuantity:  item.Quantity,
					History:          []Event{*event},
				}
				err = s.updateAsset(ctx, &newAsset)
				if err != nil {
					return err
				}
			}
			break
		}
	}
	if !stopFound {
		return fmt.Errorf("no pending delivery stop found for facility %s", facilityID)
	}

	allDelivered := true
	for _, stop := range shipment.Stops {
		if stop.Status != "COMPLETED" {
			allDelivered = false
			break
		}
	}
	if allDelivered {
		shipment.Status = "COMPLETED"
	}

	return s.updateShipment(ctx, shipment)
}