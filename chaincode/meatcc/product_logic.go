package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)


// CreateProduct tạo một sản phẩm mới trong danh mục.
// Chỉ Super Admin mới có quyền gọi.
func (s *SmartContract) CreateProduct(ctx contractapi.TransactionContextInterface, sku string, name string, description string, unit string, sourceType string, category string) error {
	// if err := requireRole(ctx, "superadmin"); err != nil {
	// 	return err
	// }
	exists, err := s.assetExists(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("product with name %s already exists", name)
	}

	product := Product{
		ObjectType:  "Product",
		SKU:         sku,
		Name:        name,
		Description: description,
		Unit:        unit,
		SourceType:  sourceType,
		Category:    category,
		Active:      true,
	}
	productJSON, _ := json.Marshal(product)
	return ctx.GetStub().PutState(product.SKU, productJSON)
}

// QueryProducts truy vấn danh sách sản phẩm theo loại nguồn và danh mục.
func (s *SmartContract) QueryProducts(ctx contractapi.TransactionContextInterface, sourceType string, category string) ([]*Product, error) {
	queryString := fmt.Sprintf(`{
		"selector":{
			"docType":"Product",
			"sourceType":"%s",
			"category":"%s",
			"active":true
		}
	}`, sourceType, category)

	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)

	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var products []*Product
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var product Product
		err = json.Unmarshal(queryResponse.Value, &product)
		if err != nil {
			return nil, err
		}
		products = append(products, &product)
	}
	return products, nil
}

// GetProduct lấy thông tin chi tiết của một sản phẩm bằng SKU.
func (s *SmartContract) GetProduct(ctx contractapi.TransactionContextInterface, sku string) (*Product, error) {
	productJSON, err := ctx.GetStub().GetState(sku)
	if err != nil {
		return nil, err
	}
	if productJSON == nil {
		return nil, fmt.Errorf("product with SKU %s does not exist", sku)
	}

	var product Product
	err = json.Unmarshal(productJSON, &product)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// DeactivateProduct hủy kích hoạt một sản phẩm.
// Chỉ Super Admin mới có quyền gọi.
func (s *SmartContract) DeactivateProduct(ctx contractapi.TransactionContextInterface, sku string) error {
	if err := requireRole(ctx, "superadmin"); err != nil {
		return err
	}
	product, err := s.GetProduct(ctx, sku)
	if err != nil {
		return err
	}
	product.Active = false
	productJSON, _ := json.Marshal(product)
	return ctx.GetStub().PutState(sku, productJSON)
}

// ActivateProduct kích hoạt lại một sản phẩm.
// Chỉ Super Admin mới có quyền gọi.
func (s *SmartContract) ActivateProduct(ctx contractapi.TransactionContextInterface, sku string) error {
	if err := requireRole(ctx, "superadmin"); err != nil {
		return err
	}
	product, err := s.GetProduct(ctx, sku)
	if err != nil {
		return err
	}
	product.Active = true
	productJSON, _ := json.Marshal(product)
	return ctx.GetStub().PutState(sku, productJSON)
}

// UpdateProduct cập nhật thông tin mô tả của sản phẩm.
// Chỉ Super Admin mới có quyền gọi.
func (s *SmartContract) UpdateProduct(ctx contractapi.TransactionContextInterface, sku string, name string, description string, unit string) error {
	if err := requireRole(ctx, "superadmin"); err != nil {
		return err
	}
	product, err := s.GetProduct(ctx, sku)
	if err != nil {
		return err
	}
	product.Name = name
	product.Description = description
	product.Unit = unit
	productJSON, _ := json.Marshal(product)
	return ctx.GetStub().PutState(sku, productJSON)
}


