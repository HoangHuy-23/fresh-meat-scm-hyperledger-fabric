package main

import (
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Kiểm tra vai trò của client có nằm trong danh sách cho phép không.
func requireRole(ctx contractapi.TransactionContextInterface, allowedRoles ...string) error {
	role, found, err := ctx.GetClientIdentity().GetAttributeValue("role")
	if err != nil {
		return fmt.Errorf("failed to get 'role' attribute: %v", err)
	}
	if !found {
		return fmt.Errorf("the client identity does not have a 'role' attribute")
	}

	for _, allowedRole := range allowedRoles {
		if role == allowedRole {
			return nil
		}
	}

	return fmt.Errorf("caller with role '%s' is not authorized", role)
}

// Kiểm tra client có phải là chủ sở hữu của asset không.
func requireOwnership(ctx contractapi.TransactionContextInterface, asset *MeatAsset) error {
	callerFacilityID, found, err := ctx.GetClientIdentity().GetAttributeValue("facilityID")
	if err != nil {
		return fmt.Errorf("failed to get 'facilityID' attribute: %v", err)
	}
	if !found {
		return fmt.Errorf("the client identity does not have an 'facilityID' attribute")
	}

	if asset.OwnerOrg != callerFacilityID {
		return fmt.Errorf("caller from facility '%s' is not the owner of asset %s (owner is '%s')", callerFacilityID, asset.AssetID, asset.OwnerOrg)
	}

	return nil
}

// Kiểm tra loại cơ sở của client có nằm trong danh sách cho phép không.
func requireFacilityType(ctx contractapi.TransactionContextInterface, allowedTypes ...string) error {
	facilityType, found, err := ctx.GetClientIdentity().GetAttributeValue("facilityType")
	if err != nil {
		return fmt.Errorf("failed to get 'facilityType' attribute: %v", err)
	}
	if !found {
		return fmt.Errorf("the client identity does not have a 'facilityType' attribute")
	}

	for _, allowedType := range allowedTypes {
		if facilityType == allowedType {
			return nil // Hợp lệ
		}
	}

	return fmt.Errorf("caller from facility type '%s' is not authorized for this action", facilityType)
}

// Trích xuất tên chung (CN) từ chứng chỉ của client, được sử dụng làm ID đăng ký.
func getEnrollmentID(ctx contractapi.TransactionContextInterface) (string, error) {
	cert, err := ctx.GetClientIdentity().GetX509Certificate()
	if err != nil {
		return "", fmt.Errorf("failed to get X509 certificate: %v", err)
	}
	return cert.Subject.CommonName, nil
}

// Kiểm tra client có phải là tài xế được chỉ định cho lô vận chuyển không.
func requireAssignedDriver(ctx contractapi.TransactionContextInterface, shipment *ShipmentAsset) error {
	callerEnrollmentID, err := getEnrollmentID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client enrollment ID: %v", err)
	}

	if shipment.DriverEnrollmentID != callerEnrollmentID {
		return fmt.Errorf("caller '%s' is not the designated driver for shipment %s (driver is '%s')", callerEnrollmentID, shipment.ShipmentID, shipment.DriverEnrollmentID)
	}

	return nil
}