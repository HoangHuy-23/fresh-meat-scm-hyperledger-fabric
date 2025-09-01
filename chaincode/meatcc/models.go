package main

// Quantity định nghĩa đơn vị và giá trị số lượng.
type Quantity struct {
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
}

// MediaPointer là tham chiếu đến file media lưu ngoài blockchain.
type MediaPointer struct {
	S3Bucket string `json:"s3Bucket"`
	S3Key    string `json:"s3Key"`
	MimeType string `json:"mimeType"`
}

// FarmDetails lưu thông tin giai đoạn nuôi/trồng tại trang trại.
type FarmDetails struct {
	FacilityID   string         `json:"facilityID"`
	FacilityName string         `json:"facilityName"`
	SowingDate   string         `json:"sowingDate"`
	HarvestDate  string         `json:"harvestDate"`
	Fertilizers  []string       `json:"fertilizers"`
	Pesticides   []string       `json:"pesticides"`
	Certificates []MediaPointer `json:"certificates"`
}

// ProcessingStep mô tả một bước trong quá trình chế biến.
type ProcessingStep struct {
	Name      string `json:"name"`
	Technique string `json:"technique"`
	Timestamp string `json:"timestamp"`
}

// ProcessingDetails lưu thông tin về quá trình chế biến.
type ProcessingDetails struct {
	ProcessorOrgName string           `json:"processorOrgName"`
	FacilityName     string           `json:"facilityName"`
	Steps            []ProcessingStep `json:"steps"`
	Certificates     []MediaPointer   `json:"certificates"`
}

// ShipmentTimeline lưu mốc thời gian trong quá trình vận chuyển.
type ShipmentTimeline struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Location  string `json:"location,omitempty"`
}

// StorageDetails lưu thông tin về quá trình lưu kho.
type StorageDetails struct {
	OwnerOrgName    string `json:"ownerOrgName"`
	FacilityName    string `json:"facilityName"`
	LocationInStore string `json:"locationInStore,omitempty"`
	Temperature     string `json:"temperature,omitempty"`
	Note            string `json:"note"`
}

// SoldDetails lưu thông tin về việc bán hàng cuối cùng.
type SoldDetails struct {
	RetailerOrgName string `json:"retailerOrgName"`
	FacilityName    string `json:"facilityName"`
	SaleTimestamp   string `json:"saleTimestamp"`
}

// Event lưu lại sự kiện quan trọng trong vòng đời của asset.
type Event struct {
	Type      string      `json:"type"`
	ActorMSP  string      `json:"actorMSP"`
	ActorID   string      `json:"actorID"`
	Timestamp string      `json:"timestamp"`
	TxID      string      `json:"txID"`
	Details   interface{} `json:"details"`
}

// MeatAsset là đối tượng chính để truy xuất nguồn gốc sản phẩm.
type MeatAsset struct {
	ObjectType       string   `json:"docType"`
	AssetID          string   `json:"assetID"`
	ParentAssetIDs   []string `json:"parentAssetIDs"`
	ProductName      string   `json:"productName"`
	Status           string   `json:"status"`
	OwnerOrg         string   `json:"ownerOrg"`
	OriginalQuantity Quantity `json:"originalQuantity"`
	CurrentQuantity  Quantity `json:"currentQuantity"`
	History          []Event  `json:"history"`
}

// ItemInShipment mô tả một sản phẩm nằm trong lô vận chuyển.
type ItemInShipment struct {
	AssetID  string   `json:"assetID"`
	Quantity Quantity `json:"quantity"`
}

// StopInJourney mô tả một điểm dừng trong hành trình vận chuyển.
type StopInJourney struct {
	FacilityID      string           `json:"facilityID"`
	FacilityName    string           `json:"facilityName"`    // <-- THÊM MỚI
	FacilityAddress string           `json:"facilityAddress"` // <-- THÊM MỚI
	Action          string           `json:"action"`
	Status          string           `json:"status"`
	Items           []ItemInShipment `json:"items"`
}

// ShipmentAsset mô tả một lô vận chuyển.
type ShipmentAsset struct {
	ObjectType         string             `json:"docType"`
	ShipmentID         string             `json:"shipmentID"`
	ShipmentType       string             `json:"shipmentType"`
	DriverEnrollmentID string             `json:"driverEnrollmentID"`
	DriverName         string             `json:"driverName"`
	VehiclePlate       string             `json:"vehiclePlate"`
	Status             string             `json:"status"`
	Stops              []StopInJourney    `json:"stops"`
	Timeline           []ShipmentTimeline `json:"timeline"`
	History            []Event            `json:"history"`
}

// ChildAssetInput dùng cho các hàm tách lô sản phẩm.
type ChildAssetInput struct {
	AssetID     string   `json:"assetID"`
	ProductName string   `json:"productName"`
	Quantity    Quantity `json:"quantity"`
}

// FullAssetTrace là cấu trúc trả về khi truy xuất nguồn gốc asset.
type FullAssetTrace struct {
	AssetID          string   `json:"assetID"`
	ParentAssetIDs   []string `json:"parentAssetIDs"`
	ProductName      string   `json:"productName"`
	Status           string   `json:"status"`
	OriginalQuantity Quantity `json:"originalQuantity"`
	CurrentQuantity  Quantity `json:"currentQuantity"`
	FullHistory      []Event  `json:"fullHistory"`
}
