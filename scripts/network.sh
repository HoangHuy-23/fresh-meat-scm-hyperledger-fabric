#!/bin/bash

set -e

# =================================================================
# HÀM KIỂM TRA QUYỀN SỞ HỮU
# =================================================================
function check_permissions() {
  # Kiểm tra xem thư mục hiện tại có cho phép user hiện tại ghi file không.
  if [ ! -w "." ]; then
    echo "===================== LỖI PHÂN QUYỀN ====================="
    echo "Lỗi: Script không có quyền ghi file trong thư mục hiện tại."
    echo "Nguyên nhân có thể là do Docker hoặc một lệnh sudo trước đó đã"
    echo "tạo ra các file với quyền sở hữu của 'root'."
    echo ""
    echo ">>> ĐỂ SỬA LỖI, HÃY CHẠY LỆNH SAU:"
    echo "sudo chown -R $(whoami):$(whoami) ."
    echo "==========================================================="
    exit 1 # Dừng script ngay lập tức
  fi
}

# GỌI HÀM KIỂM TRA NGAY TỪ ĐẦU
check_permissions

# Nạp biến môi trường ngay từ đầu
source .env

# Import các script tiện ích
. scripts/envVar.sh

# Màu sắc cho output
GREEN='\033[0;32m'
NC='\033[0m' # No Color

function printHelp() {
  echo "Sử dụng:
  network.sh <lệnh>
  Các lệnh:
    - up:      Khởi động mạng lưới (tạo certs, chạy docker, tạo channel)
    - down:    Dừng và xóa mạng lưới (dọn dẹp container, volume, crypto)
    - deployCC: Triển khai chaincode lên channel
  Ví dụ:
    ./scripts/network.sh up
    ./scripts/network.sh deployCC
    ./scripts/network.sh down"
}

function networkUp() {
  # Kiểm tra các công cụ cần thiết
  command -v configtxgen >/dev/null || { echo "configtxgen không được tìm thấy"; exit 1; }
  command -v peer >/dev/null || { echo "peer không được tìm thấy"; exit 1; }

  # 1. Tạo chứng chỉ
  echo -e "${GREEN}--- 1. Đang tạo chứng chỉ... ---${NC}"
  ./scripts/registerEnroll.sh

  # 2. Tạo Genesis Block
  echo -e "${GREEN}--- 2. Đang tạo Genesis Block... ---${NC}"
  configtxgen -profile TwoOrgsOrdererGenesis -channelID system-channel -outputBlock ./network/channel-artifacts/genesis.block

  # 3. Khởi động mạng lưới
  echo -e "${GREEN}--- 3. Đang khởi động các node... ---${NC}"
  docker-compose -f network/docker/docker-compose.yaml up -d
  echo "Chờ 5 giây để các node ổn định..."
  sleep 5

  # 4. Tạo Channel
  echo -e "${GREEN}--- 4. Đang tạo channel '${CHANNEL_NAME}'... ---${NC}"
  ./scripts/createChannel.sh
  
  echo -e "${GREEN}--- Mạng lưới đã sẵn sàng! ---${NC}"
}

function networkDown() {
  echo -e "${GREEN}--- Đang dừng và dọn dẹp mạng lưới... ---${NC}"
  docker-compose -f network/docker/docker-compose.yaml down --volumes --remove-orphans
  docker-compose -f network/docker/docker-compose-ca.yaml down --volumes --remove-orphans
  
  # Xóa chaincode images
  docker rmi $(docker images dev-* -q) 2>/dev/null || echo "Không có chaincode image nào để xóa."

  # Xóa artifacts
  rm -rf network/channel-artifacts/*
  rm -rf network/crypto-config
  
  echo -e "${GREEN}--- Dọn dẹp hoàn tất! ---${NC}"
}

function deployCC() {
  echo -e "${GREEN}--- Đang triển khai chaincode '${CC_NAME}'... ---${NC}"
  ./scripts/deployCC.sh
  echo -e "${GREEN}--- Triển khai chaincode hoàn tất! ---${NC}"
}

# --- Xử lý tham số đầu vào ---
MODE=$1
if [ "$MODE" == "up" ]; then
  networkUp
elif [ "$MODE" == "down" ]; then
  networkDown
elif [ "$MODE" == "deployCC" ]; then
  deployCC
else
  printHelp
  exit 1
fi