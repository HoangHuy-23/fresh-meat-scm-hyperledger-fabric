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
	callerOrg, found, err := ctx.GetClientIdentity().GetAttributeValue("orgShortName")
	if err != nil {
		return fmt.Errorf("failed to get 'orgShortName' attribute: %v", err)
	}
	if !found {
		return fmt.Errorf("the client identity does not have an 'orgShortName' attribute")
	}

	if asset.OwnerOrg != callerOrg {
		return fmt.Errorf("caller from org '%s' is not the owner of asset %s (owner is '%s')", callerOrg, asset.AssetID, asset.OwnerOrg)
	}

	return nil
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