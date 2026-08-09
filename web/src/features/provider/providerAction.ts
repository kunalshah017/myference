import { getAddress, type Hash } from 'viem'
import type { ProviderAPI, ProviderAction } from '../../lib/api'
import type { MarketWriter, SubmittedTransaction } from '../../lib/marketContract'
import { injectedProvider } from '../../lib/chain'

type Wait = (milliseconds: number) => Promise<void>

export async function assertProviderWallet(action: ProviderAction) {
  const provider = injectedProvider()
  if (!provider) throw new Error('Connect the provider account wallet to approve this action.')
  const accounts = await provider.request({ method: 'eth_requestAccounts' }) as string[]
  if (!accounts[0] || getAddress(accounts[0]) !== getAddress(action.wallet_address)) throw new Error(`Switch your wallet to ${action.wallet_address}.`)
}

export async function executeProviderAction(action: ProviderAction, writer: MarketWriter, api: ProviderAPI, wait: Wait = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds))) {
  const transactions: SubmittedTransaction[] = []
  if (action.kind === 'publish_offer') {
    for (const offer of action.offers ?? []) transactions.push(await writer.publishOffer({ offerID: offer.offer_id, model: offer.model, capabilities: offer.capabilities, inputPerMillion: BigInt(offer.input_per_million_wei), outputPerMillion: BigInt(offer.output_per_million_wei), computePerSecond: BigInt(offer.compute_per_second_wei) }))
  } else if (action.kind === 'deposit_collateral') transactions.push(await writer.depositBond(BigInt(action.amount_wei ?? '0')))
  else if (action.kind === 'request_collateral_exit') transactions.push(await writer.requestBondExit())
  else if (action.kind === 'finalize_collateral_exit') transactions.push(await writer.finalizeBondExit())
  if (transactions.length === 0) throw new Error('This provider action has no wallet transaction.')
  for (const transaction of transactions) await transaction.confirm()
  let current = await api.submitted(action.id, transactions.map((transaction) => transaction.hash as Hash))
  while (current.status !== 'confirmed') {
    if (Date.now() >= new Date(current.expires_at).getTime()) throw new Error('The provider action expired before it was indexed.')
    await wait(1_000)
    current = await api.get(action.id)
  }
  return current
}
