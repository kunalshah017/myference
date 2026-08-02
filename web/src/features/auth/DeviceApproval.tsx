import { useState, type FormEvent } from 'react'
import type { Address } from 'viem'
import { AuthAPI, OperationsAPI, type PendingDevice } from '../../lib/api'
import { injectedProvider } from '../../lib/chain'
import { ViemMarketWriter } from '../../lib/marketContract'

export function DeviceApproval({ api = new AuthAPI(), authorizeSigner }: { api?: AuthAPI; authorizeSigner?: (signer: Address) => Promise<void> }) {
  const [code, setCode] = useState('')
  const [pending, setPending] = useState<PendingDevice>()
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const inspect = async (event: FormEvent) => {
    event.preventDefault(); setError(''); setMessage('')
    try { setPending(await api.inspectDevice(code.trim().toUpperCase())) } catch (reason) { setError(reason instanceof Error ? reason.message : 'Device code could not be checked.') }
  }
  const approve = async () => {
    if (!pending) return
    setError('')
    try {
      if (authorizeSigner) await authorizeSigner(pending.signer_address)
      else {
        const operations = await new OperationsAPI().operations()
        const transaction = await new ViemMarketWriter(operations, injectedProvider()).setProviderSigner(pending.signer_address, true)
        await transaction.confirm()
      }
      await api.approveDevice(code.trim().toUpperCase())
      setMessage('Machine approved. Its headless receipt signer is authorized on Monad.')
      setPending(undefined)
    } catch (reason) { setError(reason instanceof Error ? reason.message : 'Machine approval failed.') }
  }
  return <section className="auth-section" aria-labelledby="device-approval-title"><p className="eyebrow">Machine authorization</p><h2 id="device-approval-title">Approve this exact machine</h2><form onSubmit={inspect}><label htmlFor="device-code">Device code</label><div className="inline-form"><input id="device-code" value={code} onChange={(event) => setCode(event.target.value)} autoComplete="one-time-code" required /><button type="submit">Review machine</button></div></form>{pending && <div className="machine-proof"><span>Pending machine</span><strong>{pending.machine_name}</strong><code>{pending.signer_address}</code><time dateTime={pending.expires_at}>Code expires {new Date(pending.expires_at).toLocaleString()}</time><button type="button" onClick={approve}>Approve {pending.machine_name}</button></div>}{message && <p role="status">{message}</p>}{error && <p role="alert" className="inline-error">{error}</p>}</section>
}
