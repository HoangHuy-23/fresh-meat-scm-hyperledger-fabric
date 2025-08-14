#!/bin/bash
set -e

ORG_NUM=$1
source .env
source scripts/envVar.sh

ORG_MSP_ID=""
if [ "$ORG_NUM" -eq 1 ]; then
  ORG_MSP_ID="MeatSupplyOrgMSP"
elif [ "$ORG_NUM" -eq 2 ]; then
  ORG_MSP_ID="RegulatorOrgMSP"
else
  echo "Lỗi: Tổ chức không hợp lệ."
  exit 1
fi

echo "--- Tạo transaction cập nhật anchor peer cho ${ORG_MSP_ID} ---"
configtxgen -profile TwoOrgsChannel -outputAnchorPeersUpdate \
./network/channel-artifacts/${ORG_MSP_ID}anchors.tx -channelID $CHANNEL_NAME -asOrg $ORG_MSP_ID

echo "--- ${ORG_MSP_ID} cập nhật anchor peer ---"
export_org_vars $ORG_NUM

# === SỬA LỖI ĐƯỜNG DẪN --cafile ===
ORDERER_CA=${CRYPTO_PATH}/ordererOrganizations/example.com/orderers/orderer1.meatsupply.example.com/tls/ca.crt

peer channel update -o localhost:${ORDERER1_PORT} --ordererTLSHostnameOverride orderer1.meatsupply.example.com \
-c $CHANNEL_NAME -f ./network/channel-artifacts/${ORG_MSP_ID}anchors.tx \
--tls --cafile "$ORDERER_CA"