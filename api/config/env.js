import * as dotenv from "dotenv";
dotenv.config();

export const config = {
  MSP_ID: process.env.ACCESSBANK_MSP_ID ?? "AccessBankMSP",
  PEER_ENDPOINT: process.env.ACCESSBANK_PEER_ENDPOINT ?? "localhost:7051",
  TLS_CERT_PATH: process.env.ACCESSBANK_TLS_CERT_PATH,
  ID_CERT_PATH: process.env.ACCESSBANK_ID_CERT_PATH,
  KEY_PATH: process.env.ACCESSBANK_KEY_PATH,
  CHANNEL: process.env.CHANNEL,
  CHAINCODE: process.env.CHAINCODE_NAME,
  CHECKPOINT_FILE: process.env.CHECKPOINT_FILE ?? "./payment-events.chk",
};
