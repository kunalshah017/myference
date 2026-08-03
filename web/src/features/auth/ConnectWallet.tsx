import { useState } from 'react'
import type { EIP1193Provider } from 'viem'
import { AuthAPI, type Session } from '../../lib/api'
import { authenticateWallet, injectedProvider } from '../../lib/chain'

export function ConnectWallet({ api = new AuthAPI(), provider = injectedProvider(), session: restoredSession, onConnected, onDisconnected }: { api?: AuthAPI; provider?: EIP1193Provider; session?: Session; onConnected?: (session: Session) => void; onDisconnected?: () => void }) {
  const [session, setSession] = useState<Session>()
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const connect = async () => {
    if (!provider) { setError('Install or open an EVM wallet to connect.'); return }
    setBusy(true); setError('')
    try {
      let challengeId = ''
      const signed = await authenticateWallet(provider, async (address) => {
        const challenge = await api.challenge(address)
        challengeId = challenge.id
        return challenge
      })
      const established = await api.verify(challengeId, signed.signature)
      setSession(established)
      onConnected?.(established)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Wallet connection failed.')
    } finally { setBusy(false) }
  }

  const disconnect = async () => {
    setBusy(true); setError('')
    try {
      await api.logout()
      setSession(undefined)
      onDisconnected?.()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Wallet disconnect failed.')
    } finally { setBusy(false) }
  }

  const connected = restoredSession ?? session
  if (connected) return <div className="wallet-control wallet-connected"><span className="wallet-address" title={connected.wallet_address}>{connected.wallet_address.slice(0, 6)}…{connected.wallet_address.slice(-4)}</span><button type="button" className="wallet-disconnect" onClick={disconnect} disabled={busy}>{busy ? 'Disconnecting…' : 'Disconnect wallet'}</button>{error && <p role="alert" className="inline-error">{error}</p>}</div>
  return <div className="wallet-control"><button type="button" onClick={connect} disabled={busy}>{busy ? 'Connecting…' : 'Connect wallet'}</button>{error && <p role="alert" className="inline-error">{error}</p>}</div>
}
