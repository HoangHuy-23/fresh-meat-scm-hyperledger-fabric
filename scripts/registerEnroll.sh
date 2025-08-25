#!/bin/bash
# scripts/registerEnroll.sh

# Dừng lại nếu có lỗi
set -e
source .env

# --- Dọn dẹp ---
if [ -d "network/crypto-config" ]; then
  echo "--- Xóa thư mục crypto-config cũ bằng sudo ---"
  sudo rm -rf network/crypto-config
fi

# --- Kiểm tra công cụ ---
if ! command -v fabric-ca-client > /dev/null; then
  echo "LỖI: fabric-ca-client không được tìm thấy."
  exit 1
fi

# --- Khởi động CAs ---
echo "--- Khởi động Fabric CAs ---"
docker-compose -f network/docker/docker-compose-ca.yaml up -d
# Lấy lại quyền sở hữu ngay sau khi docker chạy
echo "--- Sửa lỗi phân quyền cho thư mục crypto-config ---"
sudo chown -R $(whoami):$(whoami) network/crypto-config
echo "--- Chờ CA khởi động ---"
sleep 4

# =================================================================
# TẠO CHỨNG CHỈ CHO MEATSUPPLYORG
# =================================================================
echo "--- Đang tạo chứng chỉ cho MeatSupplyOrg ---"
export FABRIC_CA_CLIENT_HOME=${PWD}/network/crypto-config/peerOrganizations/meatsupply.example.com/
CA1_TLS_CERT_PATH=${PWD}/network/crypto-config/fabric-ca/meatsupply/ca-cert.pem
CA1_URL="https://admin:adminpw@localhost:7054"
CA1_NAME="ca.meatsupply.example.com"

fabric-ca-client enroll -u $CA1_URL --caname $CA1_NAME --tls.certfiles $CA1_TLS_CERT_PATH

echo 'NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/localhost-7054-ca-meatsupply-example-com.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/localhost-7054-ca-meatsupply-example-com.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/localhost-7054-ca-meatsupply-example-com.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/localhost-7054-ca-meatsupply-example-com.pem
    OrganizationalUnitIdentifier: orderer' > "${FABRIC_CA_CLIENT_HOME}/msp/config.yaml"

fabric-ca-client register --caname $CA1_NAME --id.name peer0 --id.secret peer0pw --id.type peer --tls.certfiles $CA1_TLS_CERT_PATH
fabric-ca-client enroll -u https://peer0:peer0pw@localhost:7054 --caname $CA1_NAME -M "${FABRIC_CA_CLIENT_HOME}/peers/peer0.meatsupply.example.com/msp" --csr.hosts peer0.meatsupply.example.com --tls.certfiles $CA1_TLS_CERT_PATH
cp "${FABRIC_CA_CLIENT_HOME}/msp/config.yaml" "${FABRIC_CA_CLIENT_HOME}/peers/peer0.meatsupply.example.com/msp/config.yaml"
fabric-ca-client enroll -u https://peer0:peer0pw@localhost:7054 --caname $CA1_NAME -M "${FABRIC_CA_CLIENT_HOME}/peers/peer0.meatsupply.example.com/tls" --enrollment.profile tls --csr.hosts peer0.meatsupply.example.com --csr.hosts localhost --tls.certfiles $CA1_TLS_CERT_PATH
cp "${FABRIC_CA_CLIENT_HOME}/peers/peer0.meatsupply.example.com/tls/tlscacerts/"* "${FABRIC_CA_CLIENT_HOME}/peers/peer0.meatsupply.example.com/tls/ca.crt"
cp "${FABRIC_CA_CLIENT_HOME}/peers/peer0.meatsupply.example.com/tls/signcerts/"* "${FABRIC_CA_CLIENT_HOME}/peers/peer0.meatsupply.example.com/tls/server.crt"
cp "${FABRIC_CA_CLIENT_HOME}/peers/peer0.meatsupply.example.com/tls/keystore/"* "${FABRIC_CA_CLIENT_HOME}/peers/peer0.meatsupply.example.com/tls/server.key"

# fabric-ca-client register --caname $CA1_NAME --id.name user1 --id.secret user1pw --id.type client --tls.certfiles $CA1_TLS_CERT_PATH
# fabric-ca-client enroll -u https://user1:user1pw@localhost:7054 --caname $CA1_NAME -M "${FABRIC_CA_CLIENT_HOME}/users/User1@meatsupply.example.com/msp" --tls.certfiles $CA1_TLS_CERT_PATH
# cp "${FABRIC_CA_CLIENT_HOME}/msp/config.yaml" "${FABRIC_CA_CLIENT_HOME}/users/User1@meatsupply.example.com/msp/config.yaml"

fabric-ca-client register --caname $CA1_NAME --id.name superadmin --id.secret superadminpw --id.type admin --id.attrs '"hf.Registrar.Roles=*","hf.Registrar.DelegateRoles=*","hf.Registrar.Attributes=*","hf.Revoker=true","hf.GenCRL=true","hf.AffiliationMgr=true"' --tls.certfiles $CA1_TLS_CERT_PATH
fabric-ca-client enroll -u https://superadmin:superadminpw@localhost:7054 --caname $CA1_NAME -M "${FABRIC_CA_CLIENT_HOME}/users/SuperAdmin@meatsupply.example.com/msp" --tls.certfiles $CA1_TLS_CERT_PATH   --enrollment.attrs "hf.Registrar.Roles,hf.Registrar.DelegateRoles,hf.Registrar.Attributes,hf.Revoker,hf.GenCRL,hf.AffiliationMgr"
cp "${FABRIC_CA_CLIENT_HOME}/msp/config.yaml" "${FABRIC_CA_CLIENT_HOME}/users/SuperAdmin@meatsupply.example.com/msp/config.yaml"

# # === BỔ SUNG: TẠO DANH TÍNH RIÊNG CHO API SERVER ===
# echo "--- Đang tạo danh tính cho API Server ---"
# fabric-ca-client register --caname $CA1_NAME --id.name apiserver --id.secret apiserverpw --id.type client --id.attrs 'role=superadmin:ecert,entityId=global:ecert' --tls.certfiles $CA1_TLS_CERT_PATH
# fabric-ca-client enroll -u https://apiserver:apiserverpw@localhost:7054 --caname $CA1_NAME -M "${FABRIC_CA_CLIENT_HOME}/users/ApiServer@meatsupply.example.com/msp" --tls.certfiles $CA1_TLS_CERT_PATH
# cp "${FABRIC_CA_CLIENT_HOME}/msp/config.yaml" "${FABRIC_CA_CLIENT_HOME}/users/ApiServer@meatsupply.example.com/msp/config.yaml"

# =================================================================
# TẠO CHỨNG CHỈ CHO REGULATORORG
# =================================================================
echo "--- Đang tạo chứng chỉ cho RegulatorOrg ---"
export FABRIC_CA_CLIENT_HOME=${PWD}/network/crypto-config/peerOrganizations/regulator.example.com/
CA2_TLS_CERT_PATH=${PWD}/network/crypto-config/fabric-ca/regulator/ca-cert.pem
CA2_URL="https://admin:adminpw@localhost:8054"
CA2_NAME="ca.regulator.example.com"

fabric-ca-client enroll -u $CA2_URL --caname $CA2_NAME --tls.certfiles $CA2_TLS_CERT_PATH

echo 'NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/localhost-8054-ca-regulator-example-com.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/localhost-8054-ca-regulator-example-com.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/localhost-8054-ca-regulator-example-com.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/localhost-8054-ca-regulator-example-com.pem
    OrganizationalUnitIdentifier: orderer' > "${FABRIC_CA_CLIENT_HOME}/msp/config.yaml"

fabric-ca-client register --caname $CA2_NAME --id.name peer0 --id.secret peer0pw --id.type peer --tls.certfiles $CA2_TLS_CERT_PATH
fabric-ca-client enroll -u https://peer0:peer0pw@localhost:8054 --caname $CA2_NAME -M "${FABRIC_CA_CLIENT_HOME}/peers/peer0.regulator.example.com/msp" --csr.hosts peer0.regulator.example.com --tls.certfiles $CA2_TLS_CERT_PATH
cp "${FABRIC_CA_CLIENT_HOME}/msp/config.yaml" "${FABRIC_CA_CLIENT_HOME}/peers/peer0.regulator.example.com/msp/config.yaml"
fabric-ca-client enroll -u https://peer0:peer0pw@localhost:8054 --caname $CA2_NAME -M "${FABRIC_CA_CLIENT_HOME}/peers/peer0.regulator.example.com/tls" --enrollment.profile tls --csr.hosts peer0.regulator.example.com --csr.hosts localhost --tls.certfiles $CA2_TLS_CERT_PATH
cp "${FABRIC_CA_CLIENT_HOME}/peers/peer0.regulator.example.com/tls/tlscacerts/"* "${FABRIC_CA_CLIENT_HOME}/peers/peer0.regulator.example.com/tls/ca.crt"
cp "${FABRIC_CA_CLIENT_HOME}/peers/peer0.regulator.example.com/tls/signcerts/"* "${FABRIC_CA_CLIENT_HOME}/peers/peer0.regulator.example.com/tls/server.crt"
cp "${FABRIC_CA_CLIENT_HOME}/peers/peer0.regulator.example.com/tls/keystore/"* "${FABRIC_CA_CLIENT_HOME}/peers/peer0.regulator.example.com/tls/server.key"

fabric-ca-client register --caname $CA2_NAME --id.name user1 --id.secret user1pw --id.type client --tls.certfiles $CA2_TLS_CERT_PATH
fabric-ca-client enroll -u https://user1:user1pw@localhost:8054 --caname $CA2_NAME -M "${FABRIC_CA_CLIENT_HOME}/users/User1@regulator.example.com/msp" --tls.certfiles $CA2_TLS_CERT_PATH
cp "${FABRIC_CA_CLIENT_HOME}/msp/config.yaml" "${FABRIC_CA_CLIENT_HOME}/users/User1@regulator.example.com/msp/config.yaml"

fabric-ca-client register --caname $CA2_NAME --id.name org2admin --id.secret adminpw --id.type admin --tls.certfiles $CA2_TLS_CERT_PATH
fabric-ca-client enroll -u https://org2admin:adminpw@localhost:8054 --caname $CA2_NAME -M "${FABRIC_CA_CLIENT_HOME}/users/Admin@regulator.example.com/msp" --tls.certfiles $CA2_TLS_CERT_PATH
cp "${FABRIC_CA_CLIENT_HOME}/msp/config.yaml" "${FABRIC_CA_CLIENT_HOME}/users/Admin@regulator.example.com/msp/config.yaml"

# =================================================================
# TẠO CHỨNG CHỈ CHO ORDERER ORG
# =================================================================
echo "--- Đang tạo chứng chỉ cho Orderer Org ---"
ORDERER_ORG_DIR=${PWD}/network/crypto-config/ordererOrganizations/example.com
mkdir -p $ORDERER_ORG_DIR

export FABRIC_CA_CLIENT_HOME=${PWD}/network/crypto-config/peerOrganizations/meatsupply.example.com/
fabric-ca-client register --caname $CA1_NAME --id.name orderer1 --id.secret ordererpw --id.type orderer --tls.certfiles $CA1_TLS_CERT_PATH
fabric-ca-client register --caname $CA1_NAME --id.name orderer2 --id.secret ordererpw --id.type orderer --tls.certfiles $CA1_TLS_CERT_PATH
fabric-ca-client register --caname $CA1_NAME --id.name ordererAdmin --id.secret adminpw --id.type admin --tls.certfiles $CA1_TLS_CERT_PATH

export FABRIC_CA_CLIENT_HOME=${PWD}/network/crypto-config/peerOrganizations/regulator.example.com/
fabric-ca-client register --caname $CA2_NAME --id.name orderer3 --id.secret ordererpw --id.type orderer --tls.certfiles $CA2_TLS_CERT_PATH

fabric-ca-client enroll -u https://orderer1:ordererpw@localhost:7054 --caname $CA1_NAME -M "${ORDERER_ORG_DIR}/orderers/orderer1.meatsupply.example.com/msp" --csr.hosts orderer1.meatsupply.example.com --tls.certfiles $CA1_TLS_CERT_PATH
cp "${PWD}/network/crypto-config/peerOrganizations/meatsupply.example.com/msp/config.yaml" "${ORDERER_ORG_DIR}/orderers/orderer1.meatsupply.example.com/msp/config.yaml"
fabric-ca-client enroll -u https://orderer1:ordererpw@localhost:7054 --caname $CA1_NAME -M "${ORDERER_ORG_DIR}/orderers/orderer1.meatsupply.example.com/tls" --enrollment.profile tls --csr.hosts orderer1.meatsupply.example.com --csr.hosts localhost --tls.certfiles $CA1_TLS_CERT_PATH
cp "${ORDERER_ORG_DIR}/orderers/orderer1.meatsupply.example.com/tls/tlscacerts/"* "${ORDERER_ORG_DIR}/orderers/orderer1.meatsupply.example.com/tls/ca.crt"
cp "${ORDERER_ORG_DIR}/orderers/orderer1.meatsupply.example.com/tls/signcerts/"* "${ORDERER_ORG_DIR}/orderers/orderer1.meatsupply.example.com/tls/server.crt"
cp "${ORDERER_ORG_DIR}/orderers/orderer1.meatsupply.example.com/tls/keystore/"* "${ORDERER_ORG_DIR}/orderers/orderer1.meatsupply.example.com/tls/server.key"

fabric-ca-client enroll -u https://orderer2:ordererpw@localhost:7054 --caname $CA1_NAME -M "${ORDERER_ORG_DIR}/orderers/orderer2.meatsupply.example.com/msp" --csr.hosts orderer2.meatsupply.example.com --tls.certfiles $CA1_TLS_CERT_PATH
cp "${PWD}/network/crypto-config/peerOrganizations/meatsupply.example.com/msp/config.yaml" "${ORDERER_ORG_DIR}/orderers/orderer2.meatsupply.example.com/msp/config.yaml"
fabric-ca-client enroll -u https://orderer2:ordererpw@localhost:7054 --caname $CA1_NAME -M "${ORDERER_ORG_DIR}/orderers/orderer2.meatsupply.example.com/tls" --enrollment.profile tls --csr.hosts orderer2.meatsupply.example.com --csr.hosts localhost --tls.certfiles $CA1_TLS_CERT_PATH
cp "${ORDERER_ORG_DIR}/orderers/orderer2.meatsupply.example.com/tls/tlscacerts/"* "${ORDERER_ORG_DIR}/orderers/orderer2.meatsupply.example.com/tls/ca.crt"
cp "${ORDERER_ORG_DIR}/orderers/orderer2.meatsupply.example.com/tls/signcerts/"* "${ORDERER_ORG_DIR}/orderers/orderer2.meatsupply.example.com/tls/server.crt"
cp "${ORDERER_ORG_DIR}/orderers/orderer2.meatsupply.example.com/tls/keystore/"* "${ORDERER_ORG_DIR}/orderers/orderer2.meatsupply.example.com/tls/server.key"

fabric-ca-client enroll -u https://orderer3:ordererpw@localhost:8054 --caname $CA2_NAME -M "${ORDERER_ORG_DIR}/orderers/orderer3.regulator.example.com/msp" --csr.hosts orderer3.regulator.example.com --tls.certfiles $CA2_TLS_CERT_PATH
cp "${PWD}/network/crypto-config/peerOrganizations/regulator.example.com/msp/config.yaml" "${ORDERER_ORG_DIR}/orderers/orderer3.regulator.example.com/msp/config.yaml"
fabric-ca-client enroll -u https://orderer3:ordererpw@localhost:8054 --caname $CA2_NAME -M "${ORDERER_ORG_DIR}/orderers/orderer3.regulator.example.com/tls" --enrollment.profile tls --csr.hosts orderer3.regulator.example.com --csr.hosts localhost --tls.certfiles $CA2_TLS_CERT_PATH
cp "${ORDERER_ORG_DIR}/orderers/orderer3.regulator.example.com/tls/tlscacerts/"* "${ORDERER_ORG_DIR}/orderers/orderer3.regulator.example.com/tls/ca.crt"
cp "${ORDERER_ORG_DIR}/orderers/orderer3.regulator.example.com/tls/signcerts/"* "${ORDERER_ORG_DIR}/orderers/orderer3.regulator.example.com/tls/server.crt"
cp "${ORDERER_ORG_DIR}/orderers/orderer3.regulator.example.com/tls/keystore/"* "${ORDERER_ORG_DIR}/orderers/orderer3.regulator.example.com/tls/server.key"

fabric-ca-client enroll -u https://ordererAdmin:adminpw@localhost:7054 --caname $CA1_NAME -M "${ORDERER_ORG_DIR}/users/Admin@example.com/msp" --tls.certfiles $CA1_TLS_CERT_PATH

# =================================================================
# TẠO MSP CHUNG CHO ORDERER ORG
# =================================================================
echo "--- Tạo MSP chung cho OrdererOrg ---"
mkdir -p ${ORDERER_ORG_DIR}/msp/cacerts
mkdir -p ${ORDERER_ORG_DIR}/msp/tlscacerts
mkdir -p ${ORDERER_ORG_DIR}/msp/admincerts

cp ${PWD}/network/crypto-config/peerOrganizations/meatsupply.example.com/msp/cacerts/* ${ORDERER_ORG_DIR}/msp/cacerts/
cp ${PWD}/network/crypto-config/peerOrganizations/regulator.example.com/msp/cacerts/* ${ORDERER_ORG_DIR}/msp/cacerts/
cp ${PWD}/network/crypto-config/peerOrganizations/meatsupply.example.com/peers/peer0.meatsupply.example.com/tls/tlscacerts/* ${ORDERER_ORG_DIR}/msp/tlscacerts/
cp ${PWD}/network/crypto-config/peerOrganizations/regulator.example.com/peers/peer0.regulator.example.com/tls/tlscacerts/* ${ORDERER_ORG_DIR}/msp/tlscacerts/
cp "${ORDERER_ORG_DIR}/users/Admin@example.com/msp/signcerts/"* "${ORDERER_ORG_DIR}/msp/admincerts/"

# =================================================================
# TRAO ĐỔI CHỨNG CHỈ TLS CA GIỮA CÁC TỔ CHỨC
# =================================================================
echo "--- Trao đổi chứng chỉ TLS CA ---"
# === SỬA LỖI: TẠO THƯ MỤC tlscacerts TRƯỚC KHI COPY ===
mkdir -p "${PWD}/network/crypto-config/peerOrganizations/meatsupply.example.com/msp/tlscacerts"
mkdir -p "${PWD}/network/crypto-config/peerOrganizations/regulator.example.com/msp/tlscacerts"

# Copy TLS CA cert của RegulatorOrg vào MSP của MeatSupplyOrg
cp "${PWD}/network/crypto-config/peerOrganizations/regulator.example.com/peers/peer0.regulator.example.com/tls/ca.crt" "${PWD}/network/crypto-config/peerOrganizations/meatsupply.example.com/msp/tlscacerts/ca.regulator.example.com-cert.pem"
# Copy TLS CA cert của MeatSupplyOrg vào MSP của RegulatorOrg
cp "${PWD}/network/crypto-config/peerOrganizations/meatsupply.example.com/peers/peer0.meatsupply.example.com/tls/ca.crt" "${PWD}/network/crypto-config/peerOrganizations/regulator.example.com/msp/tlscacerts/ca.meatsupply.example.com-cert.pem"

# --- Dừng các CA container ---
# echo "--- Dừng Fabric CAs ---"
# docker-compose -f network/docker/docker-compose-ca.yaml down

echo "--- Đã tạo xong toàn bộ chứng chỉ ---"