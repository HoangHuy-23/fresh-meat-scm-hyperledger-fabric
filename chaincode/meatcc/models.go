package main

// Quantity định nghĩa đơn vị và giá trị số lượng.
type Quantity struct {
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
}

// MediaPointer là tham chiếu đến file media lưu ngoài blockchain.
type MediaPointer struct {
	URL      string `json:"url"`
	MimeType string `json:"mimeType"`
}

// Certificate lưu trữ thông tin về chứng nhận.
type Certificate struct {
	Name  string       `json:"name"`
	Media MediaPointer `json:"media"`
}

// Address lưu trữ thông tin địa chỉ.
type Address struct {
	FullText  string  `json:"fullText"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Feed lưu trữ thông tin về thức ăn.
type Feed struct {
    Name         string  `json:"name"`          // Tên loại thức ăn (vd: "Green Feed tập ăn")
    DosageKg     float64 `json:"dosageKg"`      // Liều lượng mỗi ngày (kg/con hoặc kg/tổng đàn)
    StartDate    string  `json:"startDate"`     // Ngày bắt đầu sử dụng (YYYY-MM-DD)
    EndDate      string  `json:"endDate"`       // (Tùy chọn) Ngày kết thúc sử dụng
    Notes        string  `json:"notes"`         // (Tùy chọn) Ghi chú thêm như "giai đoạn tập ăn", "tăng trọng"
}

// Medication lưu trữ thông tin về thuốc và chất bổ sung.
type Medication struct {
    Name         string  `json:"name"` 		// Tên thuốc/chất bổ sung (vd: "Vitamin C", "Thuốc giảm đau")
    Dose         string  `json:"dose"` 	   // Liều dùng (vd: "500mg", "2ml/con")
    DateApplied  string  `json:"dateApplied"` // Ngày áp dụng (YYYY-MM-DD)
    NextDueDate  string  `json:"nextDueDate"` // (Tùy chọn) Ngày cần áp dụng tiếp theo
}

// FarmDetails lưu thông tin giai đoạn nuôi/trồng tại trang trại.
type FarmDetails struct {
	FacilityID   string         `json:"facilityID"`
	FacilityName string         `json:"facilityName"`
	Address      Address        `json:"address"`
	SowingDate   string         `json:"sowingDate"`
	StartDate    string         `json:"startDate"`
	ExpectedHarvestDate string         `json:"expectedHarvestDate"`
	HarvestDate  string         `json:"harvestDate"`
	Feeds        []Feed         `json:"feeds"` 
	Medications   []Medication  `json:"medications"` 
	Certificates []Certificate `json:"certificates"`
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
	Address          Address          `json:"address"`
	Steps            []ProcessingStep `json:"steps"`
	Certificates     []Certificate    `json:"certificates"`
}

// ShipmentTimeline lưu mốc thời gian trong quá trình vận chuyển.
type ShipmentTimeline struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Location  string `json:"location,omitempty"`
	FacilityID string `json:"facilityID"`
	Proof     map[string]interface{} `json:"proof"`
}

// StorageDetails lưu thông tin về quá trình lưu kho.
type StorageDetails struct {
	OwnerOrgName    string `json:"ownerOrgName"`
	FacilityName    string `json:"facilityName"`
	Address         Address `json:"address"`
	LocationInStore string `json:"locationInStore,omitempty"`
	Temperature     string `json:"temperature,omitempty"`
	Note            string `json:"note"`
}

// SoldDetails lưu thông tin về việc bán hàng cuối cùng.
type SoldDetails struct {
	RetailerOrgName string `json:"retailerOrgName"`
	FacilityName    string `json:"facilityName"`
	Address         Address `json:"address"`
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
	SKU              string   `json:"sku"`
	AverageWeight    Weight   `json:"averageWeight"`
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
	FacilityAddress Address          `json:"facilityAddress"` // <-- THÊM MỚI
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
	SKU         string   `json:"sku"`
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

// Weight lưu thông tin cân nặng.
type Weight struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"` // e.g., "kg", "g", "lb"
}

// Product defines a product in the catalog.
type Product struct {
	ObjectType    string  `json:"docType"`
	SKU           string  `json:"sku"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Unit          string  `json:"unit"` // e.g., "box", "tray", "piece"
	AverageWeight Weight  `json:"averageWeight"` 
	SourceType    string  `json:"sourceType"` //BEEF, PORK, CHICKEN
	Category      string  `json:"category"`   //RAW_MATERIAL, FINISHED_GOOD
	Active        bool    `json:"active"`
}