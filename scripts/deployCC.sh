#!/bin/bash
set -e

source .env
source scripts/envVar.sh

# =================================================================
# ĐỊNH NGHĨA CÁC BIẾN ĐƯỜNG DẪN NGAY TỪ ĐẦU
# =================================================================
export CRYPTO_PATH=${PWD}/network/crypto-config
ORDERER_CA=${CRYPTO_PATH}/ordererOrganizations/example.com/orderers/orderer1.meatsupply.example.com/tls/ca.crt
PEER0_ORG1_CA=${CRYPTO_PATH}/peerOrganizations/meatsupply.example.com/peers/peer0.meatsupply.example.com/tls/ca.crt
PEER0_ORG2_CA=${CRYPTO_PATH}/peerOrganizations/regulator.example.com/peers/peer0.regulator.example.com/tls/ca.crt
# =================================================================

CC_PACKAGE_FILE="${CC_NAME}.tar.gz"

# 1. Đóng gói chaincode
echo "--- Đóng gói chaincode ${CC_NAME} ---"
peer lifecycle chaincode package ${CC_PACKAGE_FILE} --path ${CC_SRC_PATH} --lang golang --label "${CC_NAME}_${CC_VERSION}"

# 2. Cài đặt chaincode cho các peer
echo "--- Cài đặt chaincode lên peer Org1 ---"
export_org_vars 1
peer lifecycle chaincode install ${CC_PACKAGE_FILE} || true # <-- THÊM || true

echo "--- Cài đặt chaincode lên peer Org2 ---"
export_org_vars 2
peer lifecycle chaincode install ${CC_PACKAGE_FILE} || true # <-- THÊM || true

# 3. Phê duyệt chaincode
export_org_vars 1
# Chờ một chút để peer xử lý việc cài đặt
sleep 3
PACKAGE_ID=$(peer lifecycle chaincode queryinstalled | grep "${CC_NAME}_${CC_VERSION}" | sed -n 's/Package ID: \(.*\), Label:.*/\1/p')
echo "Package ID là: ${PACKAGE_ID}"

echo "--- Phê duyệt chaincode cho Org1 ---"
peer lifecycle chaincode approveformyorg -o localhost:${ORDERER1_PORT} --ordererTLSHostnameOverride orderer1.meatsupply.example.com \
--channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${CC_VERSION} --package-id "${PACKAGE_ID}" --sequence ${CC_SEQUENCE} --tls \
--cafile "$ORDERER_CA"

echo "--- Phê duyệt chaincode cho Org2 ---"
export_org_vars 2
peer lifecycle chaincode approveformyorg -o localhost:${ORDERER1_PORT} --ordererTLSHostnameOverride orderer1.meatsupply.example.com \
--channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${CC_VERSION} --package-id "${PACKAGE_ID}" --sequence ${CC_SEQUENCE} --tls \
--cafile "$ORDERER_CA"

# 4. Commit chaincode
echo "--- Commit chaincode definition ---"
export_org_vars 1
peer lifecycle chaincode commit -o localhost:${ORDERER1_PORT} --ordererTLSHostnameOverride orderer1.meatsupply.example.com \
--channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${CC_VERSION} --sequence ${CC_SEQUENCE} --tls \
--cafile "$ORDERER_CA" \
--peerAddresses localhost:${PEER0_ORG1_PORT} --tlsRootCertFiles "$PEER0_ORG1_CA" \
--peerAddresses localhost:${PEER0_ORG2_PORT} --tlsRootCertFiles "$PEER0_ORG2_CA"

echo "--- Chaincode đã được commit thành công! ---"