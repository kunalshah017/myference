import { defineChain, numberToHex, stringToHex, type Address, type EIP1193Provider, type Hex } from 'viem'

export const monadTestnet = defineChain({
  id: 10143,
  name: 'Monad Testnet',
  nativeCurrency: { name: 'MON', symbol: 'MON', decimals: 18 },
  rpcUrls: { default: { http: [import.meta.env.VITE_MONAD_RPC_URL ?? 'https://testnet-rpc.monad.xyz'] } },
  blockExplorers: { default: { name: 'Monad Explorer', url: import.meta.env.VITE_MONAD_EXPLORER_URL ?? 'https://testnet.monadexplorer.com' } },
  testnet: true,
})

export async function authenticateWallet(provider: EIP1193Provider, challenge: (address: Address) => Promise<{ message: string }>): Promise<{ address: Address; signature: Hex }> {
  const chain = await provider.request({ method: 'eth_chainId' })
  if (chain !== numberToHex(monadTestnet.id)) throw new Error('Switch to Monad Testnet (chain 10143) before connecting.')
  const accounts = await provider.request({ method: 'eth_requestAccounts' }) as Address[]
  const address = accounts[0]
  if (!address) throw new Error('The wallet did not return an account.')
  const { message } = await challenge(address)
  const signature = await provider.request({ method: 'personal_sign', params: [stringToHex(message), address] }) as Hex
  return { address, signature }
}

export function injectedProvider(): EIP1193Provider | undefined {
  return (window as Window & { ethereum?: EIP1193Provider }).ethereum
}
