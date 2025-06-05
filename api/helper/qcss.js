import pkg from "@hyperledger/fabric-protos";
const { common , protos } = pkg;

// bank-api/qscc.js
// Lightweight helper that wraps the QSCC system chain‑code for ad‑hoc
// block and transaction queries. Exported as an async factory that takes
// an already‑connected Gateway instance and returns utility functions.

// pbPeer
export async function buildQsccHelpers(gateway, channelName = 'retailchannel') {
  const network = gateway.getNetwork(channelName);
  const qscc    = network.getContract('qscc');

  /**
   * Fetch a single block by number and return the decoded protobuf object.
   */
  async function getBlockByNumber(num) {
    const bytes = await qscc.evaluateTransaction('GetBlockByNumber', channelName, num.toString());
    return protos.Block.decode(bytes);
  }

  /**
   * Return basic ledger info (height, current hash, etc.)
   */
  async function getChainInfo() {
    const bytes = await qscc.evaluateTransaction('GetChainInfo', channelName);
    return common.BlockchainInfo.decode(bytes);
  }

  /**
   * Convenience iterator that yields every block 0‑>(height‑1).
   */
  async function * iterateBlocks() {
    const info = await getChainInfo();
    const height = info.getHeight();
    for (let i = 0n; i < height; i++) {
      yield await getBlockByNumber(i);
    }
  }

  return { getBlockByNumber, getChainInfo, iterateBlocks };
}