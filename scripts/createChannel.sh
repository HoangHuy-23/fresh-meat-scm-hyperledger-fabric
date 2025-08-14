#!/bin/bash
set -e

source .env
source scripts/envVar.sh

echo "--- Tạo transaction để tạo channel ---"
configtxgen -profile TwoOrgsChannel -outputCreateChannelTx ./network/channel-artifacts/${CHANNEL_NAME}.tx -channelID $CHANNEL_NAME

echo "--- Org1 tạo channel ---"
export_org_vars 1

# === SỬA LỖI ĐƯỜNG DẪN --cafile ===
# Đường dẫn đúng đến TLS CA cert của orderer là trong thư mục /tls/ca.crt của nó
ORDERER_CA=${CRYPTO_PATH}/ordererOrganizations/example.com/orderers/orderer1.meatsupply.example.com/tls/ca.crt

peer channel create -o localhost:${ORDERER1_PORT} --ordererTLSHostnameOverride orderer1.meatsupply.example.com \
-c $CHANNEL_NAME -f ./network/channel-artifacts/${CHANNEL_NAME}.tx --outputBlock ./network/channel-artifacts/${CHANNEL_NAME}.block \
--tls --cafile "$ORDERER_CA"

echo "--- Org1 join channel ---"
peer channel join -b ./network/channel-artifacts/${CHANNEL_NAME}.block

echo "--- Org2 join channel ---"
export_org_vars 2
peer channel join -b ./network/channel-artifacts/${CHANNEL_NAME}.block

echo "--- Cập nhật anchor peer cho Org1 ---"
./scripts/setAnchorPeer.sh 1

echo "--- Cập nhật anchor peer cho Org2 ---"
./scripts/setAnchorPeer.sh 2

echo "--- Channel '${CHANNEL_NAME}' đã được tạo và cấu hình thành công! ---"