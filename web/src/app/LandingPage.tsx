import { ArrowRight, Bot, Braces, Check, CircleDollarSign, Cloud, Code2, Cpu, KeyRound, Laptop, Network, Server, ShieldCheck, WalletCards, Zap } from 'lucide-react'

const LOGO_TOKEN = 'pk_BShsdiwDTuyRVVBW5GadOg'
const providers = [
  ['OpenAI', 'openai.com'], ['Anthropic', 'anthropic.com'], ['Google', 'google.com'], ['Meta', 'meta.com'], ['Mistral', 'mistral.ai'], ['Hugging Face', 'huggingface.co'],
]

const steps = [
  { icon: Braces, label: 'Request', body: 'Call one compatible API.' },
  { icon: Network, label: 'Route', body: 'Match price and capacity.' },
  { icon: Cpu, label: 'Infer', body: 'Run on available compute.' },
  { icon: ShieldCheck, label: 'Settle', body: 'Settle usage in MON.' },
]

function RoutingDiagram() {
  return <div className="landing-diagram" aria-label="Myference request routing diagram">
    <div className="diagram-column"><span>DEMAND</span><article><Code2 size={22} /><strong>Your app</strong><small>OpenAI-compatible</small></article><article><Bot size={22} /><strong>Agent</strong><small>Claude / Codex</small></article></div>
    <div className="diagram-route"><span>REQUEST</span><ArrowRight /><div className="diagram-router"><Zap size={26} /><strong>M/ ROUTER</strong><small>price · health · stake</small></div><ArrowRight /><span>RESPONSE</span></div>
    <div className="diagram-column"><span>SUPPLY</span><article><Laptop size={22} /><strong>Local machine</strong><small>Ollama / CLI</small></article><article><Cloud size={22} /><strong>Cloud access</strong><small>Hosted API</small></article></div>
  </div>
}

function LandingPage() {
  return <div className="landing-page">
    <header className="landing-header"><a className="wordmark" href="/" aria-label="Myference home"><span>M/</span> Myference</a><nav aria-label="Landing navigation"><a href="#network">Network</a><a href="#how-it-works">How it works</a><a href="#pricing">Pricing</a><a href="/docs">Docs</a></nav><a className="landing-primary compact" href="/app">Launch app <ArrowRight size={16} /></a></header>
    <main>
      <section className="landing-hero"><div className="landing-hero-copy"><p className="eyebrow">AI inference on Monad</p><h1>Unused compute,<br /><span>useful inference.</span></h1><p>Use or sell AI inference through one network.</p><div className="landing-actions"><a className="landing-primary" href="/app"><Cpu size={18} /> Browse models</a><a className="landing-secondary" href="#how-it-works">How it works <ArrowRight size={17} /></a></div></div><RoutingDiagram /></section>
      <section className="provider-strip" aria-labelledby="provider-strip-title"><p id="provider-strip-title">SUPPORTED ECOSYSTEM</p><div>{providers.map(([name, domain]) => <figure key={name}><img alt={`${name} logo`} src={`https://img.logo.dev/${domain}?token=${LOGO_TOKEN}&format=webp&retina=true`} /><figcaption>{name}</figcaption></figure>)}</div></section>
      <section id="network" className="landing-split"><div><p className="eyebrow">One account</p><h2>Use inference.<br />Provide inference.</h2></div><div className="landing-role-grid"><article><WalletCards /><h3>Use models</h3><p>Compare prices, deposit MON, chat, and create API keys.</p><a href="/app">User dashboard <ArrowRight size={16} /></a></article><article><Server /><h3>Host models</h3><p>Connect compute, set prices, and earn MON.</p><a href="/app">Provider dashboard <ArrowRight size={16} /></a></article></div></section>
      <section id="how-it-works" className="landing-process"><div className="section-heading"><div><p className="eyebrow">How it works</p><h2>Prompt to settlement.</h2></div><span className="data-label">Live only</span></div><div>{steps.map(({ icon: Icon, label, body }, index) => <article key={label}><span>{String(index + 1).padStart(2, '0')}</span><Icon size={23} /><h3>{label}</h3><p>{body}</p></article>)}</div></section>
      <section id="pricing" className="landing-pricing"><div><p className="eyebrow">Pricing</p><h2>Providers set prices.<br />You set limits.</h2><p>Pay only for verified usage.</p></div><ul><li><Check /> Live model pricing</li><li><Check /> Native MON escrow</li><li><Check /> Spend limits</li><li><Check /> Signed receipts</li></ul><a className="landing-primary" href="/app"><CircleDollarSign size={18} /> View pricing</a></section>
      <section className="landing-cta"><div><KeyRound size={28} /><p className="eyebrow">One endpoint</p><h2>Start building.</h2><code>POST /v1/chat/completions</code></div><a className="landing-primary" href="/app">Create API key <ArrowRight size={17} /></a></section>
    </main>
    <footer className="landing-footer"><a className="wordmark" href="/"><span>M/</span> Myference</a><span>REAL INFERENCE / PROVABLE SETTLEMENT</span><div><a href="/docs">Documentation</a><a href="https://github.com/kunalshah017/myference">Source on GitHub ↗</a></div></footer>
  </div>
}

export default LandingPage
