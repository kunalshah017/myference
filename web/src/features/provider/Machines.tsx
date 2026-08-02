import type { OperationMachine } from '../../lib/api'

export function Machines({ machines }: { machines: OperationMachine[] }) {
  if (!machines.length) return <p>No CLI machines are connected to this account.</p>
  return <div className="machine-ledger">{machines.map((machine)=><article key={machine.id}><div><strong>{machine.name}</strong><code>{machine.id}</code></div>{machine.backends.length===0?<span>No backends registered</span>:machine.backends.map((backend)=><div key={backend.id}><span>{backend.kind} / {backend.model}</span><span>{backend.enabled ? backend.healthy ? `${backend.capacity} slots live` : 'awaiting heartbeat' : 'stopped in CLI'}</span></div>)}</article>)}</div>
}
