import { ArrowRight, Bot, Braces, Check, CircleDollarSign, Cloud, Code2, Cpu, KeyRound, Laptop, Network, Server, ShieldCheck, WalletCards, Zap } from 'lucide-react'

const LOGO_TOKEN = 'pk_BShsdiwDTuyRVVBW5GadOg'
const providers = [
  ['OpenAI', 'openai.com'], ['Anthropic', 'anthropic.com'], ['Google', 'google.com'], ['Meta', 'meta.com'], ['Mistral', 'mistral.ai'], ['Hugging Face', 'huggingface.co'],
]

const steps = [
  { icon: Braces, label: 'Request', body: 'Your application calls one OpenAI or Anthropic-compatible endpoint.' },
  { icon: Network, label: 'Route', body: 'The broker evaluates live model, price, capacity, and health constraints.' },
  { icon: Cpu, label: 'Infer', body: 'An eligible laptop, desktop, cloud API, or CLI agent runs the request.' },
  { icon: ShieldCheck, label: 'Settle', body: 'Signed usage receipts settle provider earnings in native MON.' },
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
    <header className="landing-header"><a className="wordmark" href="/" aria-label="Myference home"><span>M/</span> Myference</a><nav aria-label="Landing navigation"><a href="#network">Network</a><a href="#how-it-works">How it works</a><a href="#pricing">Pricing</a></nav><a className="landing-primary compact" href="/app">Launch app <ArrowRight size={16} /></a></header>
    <main>
      <section className="landing-hero"><div className="landing-hero-copy"><p className="eyebrow">Distributed inference market / Monad</p><h1>Unused compute,<br /><span>useful inference.</span></h1><p>One programmable market for public AI models and idle machines. Use familiar APIs, compare live provider prices, and settle verified inference in native MON.</p><div className="landing-actions"><a className="landing-primary" href="/app"><Cpu size={18} /> Browse live models</a><a className="landing-secondary" href="#how-it-works">How routing works <ArrowRight size={17} /></a></div></div><RoutingDiagram /></section>
      <section className="provider-strip" aria-labelledby="provider-strip-title"><p id="provider-strip-title">ROUTE ACROSS THE AI ECOSYSTEM</p><div>{providers.map(([name, domain]) => <figure key={name}><img alt={`${name} logo`} src={`https://img.logo.dev/${domain}?token=${LOGO_TOKEN}&format=webp&retina=true`} /><figcaption>{name}</figcaption></figure>)}</div></section>
      <section id="network" className="landing-split"><div><p className="eyebrow">One account, both sides</p><h2>Use inference.<br />Provide inference.</h2></div><div className="landing-role-grid"><article><WalletCards /><h3>For model users</h3><p>Compare live offers, deposit MON, test models in the browser, create scoped API keys, and inspect request activity.</p><a href="/app">Open user dashboard <ArrowRight size={16} /></a></article><article><Server /><h3>For providers</h3><p>Connect machines, enable Ollama, API, or CLI backends, publish prices, monitor requests, bond collateral, and claim earnings.</p><a href="/app">Open provider dashboard <ArrowRight size={16} /></a></article></div></section>
      <section id="how-it-works" className="landing-process"><div className="section-heading"><div><p className="eyebrow">Request lifecycle</p><h2>Transparent from prompt to settlement.</h2></div><span className="data-label">Live data only</span></div><div>{steps.map(({ icon: Icon, label, body }, index) => <article key={label}><span>{String(index + 1).padStart(2, '0')}</span><Icon size={23} /><h3>{label}</h3><p>{body}</p></article>)}</div></section>
      <section id="pricing" className="landing-pricing"><div><p className="eyebrow">Market pricing</p><h2>Providers set prices.<br />You set limits.</h2><p>There is no invented flat rate. Each live offer publishes input, output, and compute pricing. Bounded sessions and scoped keys keep spending under your control.</p></div><ul><li><Check /> Per-provider model pricing</li><li><Check /> Native MON escrow</li><li><Check /> Maximum-spend API keys</li><li><Check /> Signed settlement receipts</li></ul><a className="landing-primary" href="/app"><CircleDollarSign size={18} /> Inspect live pricing</a></section>
      <section className="landing-cta"><div><KeyRound size={28} /><p className="eyebrow">One endpoint</p><h2>Build against the network.</h2><code>POST /v1/chat/completions</code></div><a className="landing-primary" href="/app">Create API access <ArrowRight size={17} /></a></section>
    </main>
    <footer className="landing-footer"><a className="wordmark" href="/"><span>M/</span> Myference</a><span>REAL INFERENCE / PROVABLE SETTLEMENT</span><a href="https://github.com/kunalshah017/myference">Source on GitHub ↗</a></footer>
  </div>
}

export default LandingPage
