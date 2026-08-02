const stages = [
  { label: 'Escrow', states: ['reserved'] },
  { label: 'Router', states: ['offered', 'accepted'] },
  { label: 'Provider', states: ['streaming', 'completed'] },
  { label: 'Settlement', states: ['signed', 'submitted', 'settled'] },
]

const order = ['reserved', 'offered', 'accepted', 'streaming', 'completed', 'signed', 'submitted', 'settled']

export function RoutingRail({ state = '' }: { state?: string }) {
  const current = order.indexOf(state)
  return <section className="routing-rail" aria-labelledby="routing-title"><div className="rail-heading"><p className="eyebrow" id="routing-title">Live routing rail</p><p>{state ? `Latest request: ${state}` : 'Awaiting a live request'}</p></div><ol>{stages.map((stage, index) => { const stageOrder = Math.min(...stage.states.map((item) => order.indexOf(item)).filter((item) => item >= 0)); const status = current < 0 ? 'idle' : stage.states.includes(state) ? 'active' : current > stageOrder ? 'complete' : 'idle'; return <li key={stage.label} data-state={status}><span className="stage-index" aria-hidden="true">{String(index + 1).padStart(2, '0')}</span><strong>{stage.label}</strong><span>{status === 'active' ? state : status}</span></li> })}</ol></section>
}
