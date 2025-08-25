# scripts/envVar.sh

export_org_vars() {
  if [ -z "$1" ]; then
    echo "Usage: export_org_vars ORG_NUM"
    return 1
  fi

  ORG_NUM=$1
  source .env

  export CRYPTO_PATH=${PWD}/network/crypto-config
  export PEER_CFG_PATH=${PWD}/network

  if [ "$ORG_NUM" -eq 1 ]; then
    export CORE_PEER_LOCALMSPID="MeatSupplyOrgMSP"
    export CORE_PEER_TLS_ROOTCERT_FILE=${CRYPTO_PATH}/peerOrganizations/meatsupply.example.com/peers/peer0.meatsupply.example.com/tls/ca.crt
    export CORE_PEER_MSPCONFIGPATH=${CRYPTO_PATH}/peerOrganizations/meatsupply.example.com/users/SuperAdmin@meatsupply.example.com/msp
    export CORE_PEER_ADDRESS=localhost:${PEER0_ORG1_PORT}
    export TARGET_PEER_HOST="peer0.meatsupply.example.com"
    export TARGET_PEER_URL="localhost:${PEER0_ORG1_PORT}"
    export TARGET_TLS_CERT=${CRYPTO_PATH}/peerOrganizations/meatsupply.example.com/peers/peer0.meatsupply.example.com/tls/ca.crt
  elif [ "$ORG_NUM" -eq 2 ]; then
    export CORE_PEER_LOCALMSPID="RegulatorOrgMSP"
    export CORE_PEER_TLS_ROOTCERT_FILE=${CRYPTO_PATH}/peerOrganizations/regulator.example.com/peers/peer0.regulator.example.com/tls/ca.crt
    export CORE_PEER_MSPCONFIGPATH=${CRYPTO_PATH}/peerOrganizations/regulator.example.com/users/Admin@regulator.example.com/msp
    export CORE_PEER_ADDRESS=localhost:${PEER0_ORG2_PORT}
    export TARGET_PEER_HOST="peer0.regulator.example.com"
    export TARGET_PEER_URL="localhost:${PEER0_ORG2_PORT}"
    export TARGET_TLS_CERT=${CRYPTO_PATH}/peerOrganizations/regulator.example.com/peers/peer0.regulator.example.com/tls/ca.crt
  else
    echo "Invalid organization number."
    return 1
  fi

  echo "Environment variables set for Org${ORG_NUM}"
}