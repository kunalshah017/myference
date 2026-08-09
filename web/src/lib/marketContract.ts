import { createPublicClient, createWalletClient, custom, getAddress, http, keccak256, stringToHex, type Address, type EIP1193Provider, type Hash, type Hex } from 'viem'
import { monadTestnet } from './chain'

export const marketABI = [
  { type: 'function', name: 'deposit', stateMutability: 'payable', inputs: [], outputs: [] },
  { type: 'function', name: 'requestWithdrawal', stateMutability: 'nonpayable', inputs: [{ name: 'amount', type: 'uint256' }], outputs: [] },
  { type: 'function', name: 'claim', stateMutability: 'nonpayable', inputs: [], outputs: [] },
  { type: 'function', name: 'depositBond', stateMutability: 'payable', inputs: [], outputs: [] },
  { type: 'function', name: 'setProviderSigner', stateMutability: 'nonpayable', inputs: [{ name: 'signer', type: 'address' }, { name: 'allowed', type: 'bool' }], outputs: [] },
  { type: 'function', name: 'requestBondExit', stateMutability: 'nonpayable', inputs: [], outputs: [] },
  { type: 'function', name: 'finalizeBondExit', stateMutability: 'nonpayable', inputs: [], outputs: [] },
  { type: 'function', name: 'publishOffer', stateMutability: 'nonpayable', inputs: [{ name: 'offerId', type: 'bytes32' }, { name: 'modelHash', type: 'bytes32' }, { name: 'capabilityHash', type: 'bytes32' }, { name: 'inputPerMillion', type: 'uint256' }, { name: 'outputPerMillion', type: 'uint256' }, { name: 'computePerSecond', type: 'uint256' }], outputs: [] },
  { type: 'function', name: 'latestOfferVersion', stateMutability: 'view', inputs: [{ name: 'provider', type: 'address' }, { name: 'offerId', type: 'bytes32' }], outputs: [{ name: '', type: 'uint64' }] },
  { type: 'function', name: 'openSession', stateMutability: 'nonpayable', inputs: [{ name: 'sessionId', type: 'bytes32' }, { name: 'allowance', type: 'uint256' }, { name: 'expiresAt', type: 'uint64' }], outputs: [] },
  { type: 'function', name: 'requestSessionClose', stateMutability: 'nonpayable', inputs: [{ name: 'sessionId', type: 'bytes32' }], outputs: [] },
  { type: 'function', name: 'finalizeSessionClose', stateMutability: 'nonpayable', inputs: [{ name: 'sessionId', type: 'bytes32' }], outputs: [] },
] as const

export type TransactionConfirmation = { offerVersion?: number }
export type SubmittedTransaction = { hash: Hash; confirm: () => Promise<TransactionConfirmation | void> }
export type OfferInput = { offerID: string; model: string; capabilities: string[]; inputPerMillion: bigint; outputPerMillion: bigint; computePerSecond: bigint }
export type MarketConfiguration = { chain_id: number; contract_address: `0x${string}`; confirmations: number }
export function assertSuccessfulReceipt(receipt: { status: string }): void { if (receipt.status !== 'success') throw new Error('Monad transaction reverted.') }
export interface MarketWriter {
  deposit(value: bigint): Promise<SubmittedTransaction>
  requestWithdrawal(amount: bigint): Promise<SubmittedTransaction>
  claim(): Promise<SubmittedTransaction>
  depositBond(value: bigint): Promise<SubmittedTransaction>
  setProviderSigner(signer: Address, allowed: boolean): Promise<SubmittedTransaction>
  requestBondExit(): Promise<SubmittedTransaction>
  finalizeBondExit(): Promise<SubmittedTransaction>
  publishOffer(offer: OfferInput): Promise<SubmittedTransaction>
  openSession(sessionID: Hex, allowance: bigint, expiresAt: bigint): Promise<SubmittedTransaction>
  requestSessionClose(sessionID: Hex): Promise<SubmittedTransaction>
  finalizeSessionClose(sessionID: Hex): Promise<SubmittedTransaction>
}

export class ViemMarketWriter implements MarketWriter {
  private readonly address: Address
	private readonly operations: MarketConfiguration
  private readonly provider?: EIP1193Provider
	constructor(operations: MarketConfiguration, provider?: EIP1193Provider) { this.operations = operations; this.provider = provider; this.address = getAddress(operations.contract_address) }

  private async clients() {
    if (!this.provider) throw new Error('Connect an EVM wallet before sending a transaction.')
    const chainID = await this.provider.request({ method: 'eth_chainId' })
    if (chainID !== '0x279f') throw new Error('Switch to Monad Testnet (chain 10143).')
    const wallet = createWalletClient({ chain: monadTestnet, transport: custom(this.provider) })
    const [account] = await wallet.requestAddresses()
    if (!account) throw new Error('The wallet did not return an account.')
    const publicClient = createPublicClient({ chain: monadTestnet, transport: http() })
    const submitted = (hash: Hash): SubmittedTransaction => ({ hash, confirm: async () => { const receipt=await publicClient.waitForTransactionReceipt({ hash, confirmations: this.operations.confirmations });assertSuccessfulReceipt(receipt) } })
    return { wallet, publicClient, account, submitted }
  }

  async deposit(value: bigint) { const c=await this.clients(); const s=await c.publicClient.simulateContract({address:this.address,abi:marketABI,functionName:'deposit',account:c.account,value}); return c.submitted(await c.wallet.writeContract(s.request)) }
  async requestWithdrawal(amount: bigint) { const c=await this.clients(); const s=await c.publicClient.simulateContract({address:this.address,abi:marketABI,functionName:'requestWithdrawal',args:[amount],account:c.account}); return c.submitted(await c.wallet.writeContract(s.request)) }
  async claim() { const c=await this.clients(); const s=await c.publicClient.simulateContract({address:this.address,abi:marketABI,functionName:'claim',account:c.account}); return c.submitted(await c.wallet.writeContract(s.request)) }
  async depositBond(value: bigint) { const c=await this.clients(); const s=await c.publicClient.simulateContract({address:this.address,abi:marketABI,functionName:'depositBond',account:c.account,value}); return c.submitted(await c.wallet.writeContract(s.request)) }
  async setProviderSigner(signer: Address, allowed: boolean) { const c=await this.clients(); const s=await c.publicClient.simulateContract({address:this.address,abi:marketABI,functionName:'setProviderSigner',args:[getAddress(signer),allowed],account:c.account}); return c.submitted(await c.wallet.writeContract(s.request)) }
  async requestBondExit() { const c=await this.clients(); const s=await c.publicClient.simulateContract({address:this.address,abi:marketABI,functionName:'requestBondExit',account:c.account}); return c.submitted(await c.wallet.writeContract(s.request)) }
  async finalizeBondExit() { const c=await this.clients(); const s=await c.publicClient.simulateContract({address:this.address,abi:marketABI,functionName:'finalizeBondExit',account:c.account}); return c.submitted(await c.wallet.writeContract(s.request)) }
  async publishOffer(offer: OfferInput) { const c=await this.clients(); const offerHash=hashLabel(offer.offerID);const args=[offerHash,hashLabel(offer.model),hashLabel([...offer.capabilities].sort().join(',')),offer.inputPerMillion,offer.outputPerMillion,offer.computePerSecond] as const; const s=await c.publicClient.simulateContract({address:this.address,abi:marketABI,functionName:'publishOffer',args,account:c.account});const hash=await c.wallet.writeContract(s.request);return {hash,confirm:async()=>{const receipt=await c.publicClient.waitForTransactionReceipt({hash,confirmations:this.operations.confirmations});assertSuccessfulReceipt(receipt);const version=await c.publicClient.readContract({address:this.address,abi:marketABI,functionName:'latestOfferVersion',args:[c.account,offerHash]});return {offerVersion:Number(version)}}} }
  async openSession(sessionID: Hex, allowance: bigint, expiresAt: bigint) { const c=await this.clients(); const s=await c.publicClient.simulateContract({address:this.address,abi:marketABI,functionName:'openSession',args:[sessionID,allowance,expiresAt],account:c.account}); return c.submitted(await c.wallet.writeContract(s.request)) }
  async requestSessionClose(sessionID: Hex) { const c=await this.clients(); const s=await c.publicClient.simulateContract({address:this.address,abi:marketABI,functionName:'requestSessionClose',args:[sessionID],account:c.account}); return c.submitted(await c.wallet.writeContract(s.request)) }
  async finalizeSessionClose(sessionID: Hex) { const c=await this.clients(); const s=await c.publicClient.simulateContract({address:this.address,abi:marketABI,functionName:'finalizeSessionClose',args:[sessionID],account:c.account}); return c.submitted(await c.wallet.writeContract(s.request)) }
}

export function hashLabel(value: string): Hex { return keccak256(stringToHex(value)) }
