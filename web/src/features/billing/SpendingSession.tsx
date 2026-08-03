import { useState, type FormEvent } from 'react'
import { hashLabel, type MarketWriter } from '../../lib/marketContract'
import { parseMON } from '../../lib/amount'
import { Money } from '../../components/Money'
import type { OperationSession } from '../../lib/api'

export function SpendingSession({ writer, sessions, submit }: { writer: MarketWriter; sessions: OperationSession[]; submit: (action: () => ReturnType<MarketWriter['openSession']>) => Promise<void> }) {
  const [allowance, setAllowance] = useState('')
  const [hours, setHours] = useState('24')
  const open = async (event: FormEvent) => {
    event.preventDefault()
    const sessionID = hashLabel(crypto.randomUUID())
    const expiresAt = BigInt(Math.floor(Date.now()/1000) + Number(hours)*3600)
    await submit(() => writer.openSession(sessionID, parseMON(allowance), expiresAt))
  }
  return <div className="session-control"><h3>Bounded spending sessions</h3><form onSubmit={open}><label htmlFor="session-allowance">Allowance (MON)</label><input id="session-allowance" value={allowance} onChange={(event)=>setAllowance(event.target.value)} required /><label htmlFor="session-hours">Duration (hours)</label><input id="session-hours" inputMode="numeric" value={hours} onChange={(event)=>setHours(event.target.value)} required /><button type="submit">Open spending session</button></form><ul>{sessions.map((session)=><li key={session.session_id}><code>{session.session_id}</code><span><Money wei={session.spent_wei}/> spent of <Money wei={session.allowance_wei}/></span><span>{session.finalized ? 'closed' : session.close_available_at ? 'close delay active' : 'open'}</span>{!session.finalized && <button type="button" onClick={()=>void submit(()=>session.close_available_at ? writer.finalizeSessionClose(session.session_id) : writer.requestSessionClose(session.session_id))}>{session.close_available_at ? 'Finalize close' : 'Request close'}</button>}</li>)}</ul></div>
}
