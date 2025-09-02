package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Tạo một lô vận chuyển mới, lưu thông tin tài xế, phương tiện, các điểm dừng và ghi lại sự kiện khởi tạo shipment.
func (s *SmartContract) CreateShipment(ctx contractapi.TransactionContextInterface, shipmentID string, shipmentType, driverEnrollmentID, driverName, vehiclePlate string, stopsJSON string) error {
	if err := requireRole(ctx, "admin", "driver"); err != nil {
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

	for i := range stops {
		stops[i].Status = "PENDING"
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
		Timeline:           []ShipmentTimeline{},
		History:            []Event{*event},
	}

	return s.updateShipment(ctx, &shipment)
}

// Xác nhận việc lấy hàng tại một điểm dừng, cập nhật số lượng asset và trạng thái điểm dừng thành COMPLETED.
func (s *SmartContract) ConfirmPickup(ctx contractapi.TransactionContextInterface, shipmentID string, facilityID string, actualItemsJSON string, proofJSON string) error {
	shipment, err := s.readShipmentAsset(ctx, shipmentID)
	if err != nil {
		return err
	}
	if shipment.Status != "PENDING" {
		return fmt.Errorf("shipment %s is not in PENDING state", shipmentID)
	}

	// Unmarshal proofJSON để kiểm tra
	var proofDetails map[string]interface{}
	if err := json.Unmarshal([]byte(proofJSON), &proofDetails); err != nil {
		return fmt.Errorf("failed to unmarshal proofJSON: %v", err)
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

			// === PHẦN NÂNG CẤP: GHI LẠI SỰ KIỆN PICKUP ===
			pickupEvent := ShipmentTimeline{
				Type:      "pickup_confirmed",
				Timestamp: s.getTxTimestamp(ctx),
				Location:  stop.FacilityAddress, // Lấy địa chỉ từ chính điểm dừng này
				Proof:     proofDetails,          // Ghi lại URL và Hash
			}
			shipment.Timeline = append(shipment.Timeline, pickupEvent)
			// =============================================

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

				// === NÂNG CẤP SỰ KIỆN ===
				// Ghi lại cả bằng chứng ảnh vào sự kiện của asset
				eventDetails := map[string]interface{}{
					"shipmentID": shipmentID,
					"quantity":   actualItem.Quantity,
					"proof":      proofDetails, // Ghi lại URL và Hash
				}
				err = s.addEvent(ctx, asset, "PICKED_UP_FOR_SHIPMENT", asset.Status, eventDetails)
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
	if err := requireRole(ctx, "admin", "driver"); err != nil {
		return err
	}
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

	var departureLocation string // Biến để lưu địa điểm khởi hành

	for _, stop := range shipment.Stops {
		if stop.Action == "PICKUP" && stop.Status == "COMPLETED" {
			// Lấy địa điểm của điểm pickup đầu tiên làm địa điểm khởi hành
			if departureLocation == "" {
				departureLocation = stop.FacilityAddress // Lấy địa chỉ đầy đủ
			}
			// =====================
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
				err = s.addEvent(ctx, asset, "SHIPPING_STARTED", newStatus, map[string]string{"shipmentID": shipmentID})
				if err != nil {
					return fmt.Errorf("failed to update event for asset %s: %v", item.AssetID, err)
				}
			}
		}
	}

	shipment.Status = "IN_TRANSIT"
	timelineEvent := ShipmentTimeline{
		Type:      "departure",
		Timestamp: s.getTxTimestamp(ctx),
		Location:  departureLocation,
		Proof:     make(map[string]interface{}), // Không có bằng chứng cụ thể lúc này
	}
	shipment.Timeline = append(shipment.Timeline, timelineEvent)

	return s.updateShipment(ctx, shipment)
}

// Xác nhận việc giao hàng tại một điểm dừng, tạo asset mới cho bên nhận và cập nhật trạng thái shipment nếu đã giao hết.
func (s *SmartContract) ConfirmShipmentDelivery(ctx contractapi.TransactionContextInterface, shipmentID string, facilityID string, newAssetIDPrefix string, proofJSON string) error {
	if err := requireRole(ctx, "admin", "worker"); err != nil {
		return err
	}

	receiverFacilityID, _, _ := ctx.GetClientIdentity().GetAttributeValue("facilityID")   
	receiverFacilityType, _, _ := ctx.GetClientIdentity().GetAttributeValue("facilityType") 

	// Unmarshal proofJSON để sử dụng
	var proofDetails map[string]interface{}
	if err := json.Unmarshal([]byte(proofJSON), &proofDetails); err != nil {
		return fmt.Errorf("failed to unmarshal proofJSON: %v", err)
	}

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

			// === NÂNG CẤP SỰ KIỆN ARRIVAL ===
			arrivalEvent := ShipmentTimeline{
				Type:      "arrival",
				Timestamp: s.getTxTimestamp(ctx),
				Location:  stop.FacilityAddress,
				Proof:     proofDetails, 
			}
			shipment.Timeline = append(shipment.Timeline, arrivalEvent)
			// =================================

			for j, item := range stop.Items {
				parentAsset, err := s.readAsset(ctx, item.AssetID)
				if err != nil {
					return err
				}

				var newStatus string
				switch receiverFacilityType {
				case "RETAILER":
					newStatus = "AT_RETAILER"
				case "PROCESSOR":
					newStatus = "AT_PROCESSOR"
				case "WAREHOUSE":
					newStatus = "AT_WAREHOUSE"
				default:
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
					OwnerOrg:         receiverFacilityID,
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

// GetShipment lấy các chi tiết của một lô hàng cụ thể.
// Đây là một chức năng truy vấn có thể được gọi thông qua EvaluateTransaction.
func (s *SmartContract) GetShipment(ctx contractapi.TransactionContextInterface, shipmentID string) (*ShipmentAsset, error) {
	return s.readShipmentAsset(ctx, shipmentID)
}