import { useState, type FormEvent } from 'react'
import { hashLabel, type MarketWriter } from '../../lib/marketContract'
import { parseMON } from '../../lib/amount'
import { Money } from '../../components/Money'
import type { OperationSession } from '../../lib/api'

type PendingTransaction = { id: string; label: string; phase: 'wallet' | 'chain' | 'refresh' }

export function SpendingSession({ writer, sessions, submit, pending }: { writer: MarketWriter; sessions: OperationSession[]; submit: (id: string, label: string, action: () => ReturnType<MarketWriter['openSession']>) => Promise<void>; pending?: PendingTransaction }) {
  const [allowance, setAllowance] = useState('')
  const [hours, setHours] = useState('24')
  const open = async (event: FormEvent) => {
    event.preventDefault()
    const sessionID = hashLabel(crypto.randomUUID())
    const expiresAt = BigInt(Math.floor(Date.now()/1000) + Number(hours)*3600)
    await submit('open-session', 'Spending session', () => writer.openSession(sessionID, parseMON(allowance), expiresAt))
  }
  const pendingLabel = (id: string, fallback: string) => pending?.id === id ? pending.phase === 'wallet' ? `Confirm ${pending.label.toLowerCase()} in wallet…` : pending.phase === 'chain' ? 'Confirming on Monad…' : 'Refreshing account…' : fallback
  return <div className="session-control"><h3>Bounded spending sessions</h3><form onSubmit={open}><label htmlFor="session-allowance">Allowance (MON)</label><input id="session-allowance" value={allowance} onChange={(event)=>setAllowance(event.target.value)} disabled={Boolean(pending)} required /><label htmlFor="session-hours">Duration (hours)</label><input id="session-hours" inputMode="numeric" value={hours} onChange={(event)=>setHours(event.target.value)} disabled={Boolean(pending)} required /><button type="submit" disabled={Boolean(pending)}>{pendingLabel('open-session', 'Open spending session')}</button></form><ul>{sessions.map((session)=>{ const id = `close-${session.session_id}`; const label = session.close_available_at ? 'Session close' : 'Close request'; return <li key={session.session_id}><code>{session.session_id}</code><span><Money wei={session.spent_wei}/> spent of <Money wei={session.allowance_wei}/></span><span>{session.finalized ? 'closed' : session.close_available_at ? 'close delay active' : 'open'}</span>{!session.finalized && <button type="button" disabled={Boolean(pending)} onClick={()=>void submit(id, label, ()=>session.close_available_at ? writer.finalizeSessionClose(session.session_id) : writer.requestSessionClose(session.session_id))}>{pendingLabel(id, session.close_available_at ? 'Finalize close' : 'Request close')}</button>}</li>})}</ul></div>
}
