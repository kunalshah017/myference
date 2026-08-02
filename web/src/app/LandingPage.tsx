const steps = [
  ['01', 'Demand', 'Applications use one OpenAI-compatible endpoint.'],
  ['02', 'Broker', 'Myference selects an eligible provider by model, price, and health.'],
  ['03', 'Supply', 'Unused laptops, desktops, and cloud agents serve verified requests.'],
  ['04', 'Settlement', 'Signed usage receipts settle provider earnings in native MON.'],
]

const faqs = [
  ['What can providers run?', 'Ollama models, hosted API providers, and supported CLI agents can be enabled or stopped independently from the provider console.'],
  ['How do applications connect?', 'Use the OpenAI-compatible API surface, create a bounded spending session, and choose a model from the live marketplace.'],
  ['How is pricing decided?', 'Each provider publishes a token price. The broker routes only to offers that satisfy the request and current spending policy.'],
]

function NetworkVisual() {
  return <div className="landing-network" aria-label="Provider network illustration" role="img">
    <div className="landing-network-core"><span>M/</span><small>MONAD ROUTER</small></div>
    {['OLLAMA', 'API', 'CLI', 'GPU', 'LAPTOP', 'AGENT'].map((label, index) => <span className={`network-node node-${index + 1}`} key={label}>{label}</span>)}
    <i className="network-ring ring-one" /><i className="network-ring ring-two" />
  </div>
}

function LandingPage() {
  return <div className="landing-page">
    <header className="landing-header">
      <a className="landing-brand" href="/" aria-label="Myference home"><span>M/</span> MYFERENCE</a>
      <nav aria-label="Landing navigation"><a href="#platform">Platform</a><a href="#how-it-works">How it works</a><a href="#pricing">Pricing</a></nav>
      <a className="landing-button landing-button-compact" href="/app">Launch app</a>
    </header>

    <main>
      <section className="landing-hero" id="platform">
        <div className="landing-hero-copy"><p className="landing-kicker"><span className="pulse-dot" /> MONAD TESTNET / INFERENCE ROUTER</p>
          <h1>Unused compute,<br /><em>useful inference.</em></h1>
          <p className="landing-lede">Turn idle machines and AI access into a programmable supply network. Route OpenAI-compatible requests to independent providers and settle verified usage in native MON.</p>
          <div className="landing-actions"><a className="landing-button" href="/app">Launch the marketplace</a><a className="landing-text-link" href="#how-it-works">See how it works <span>↘</span></a></div>
        </div>
        <NetworkVisual />
      </section>

      <section className="landing-proof"><span>LOCAL MODELS</span><b>×</b><span>CLOUD APIS</span><b>×</b><span>CLI AGENTS</span><b>×</b><span>MON SETTLEMENT</span></section>

      <section className="landing-section" id="how-it-works"><div className="landing-section-heading"><p className="landing-kicker">THE CORE DIFFERENCE</p><h2>One router.<br /><em>Every kind of provider.</em></h2><p>Myference makes spare compute useful without hiding the ledger. Availability, pricing, and settlement remain grounded in provider offers and confirmed network records.</p></div>
        <div className="landing-steps">{steps.map(([number, title, body]) => <article key={number}><span className="step-number">{number}</span><h3>{title}</h3><p>{body}</p></article>)}</div>
      </section>

      <section className="landing-section landing-pricing" id="pricing"><div><p className="landing-kicker">PRICING</p><h2>Pay for the tokens.<br /><em>Keep the compute moving.</em></h2></div><div className="pricing-panel"><p>Providers publish model offers and application users fund bounded sessions. The protocol records the request, usage receipt, platform fee, and provider settlement separately.</p><a className="landing-text-link" href="/app">Inspect live offers <span>↗</span></a></div></section>

      <section className="landing-faq"><p className="landing-kicker">FAQ</p><h2>Built for real machines.</h2>{faqs.map(([question, answer]) => <details key={question}><summary>{question}<span>+</span></summary><p>{answer}</p></details>)}</section>
    </main>
    <footer className="landing-footer"><span>MYFERENCE / REAL INFERENCE, PROVABLE SETTLEMENT</span><a href="/app">Open operational client ↗</a></footer>
  </div>
}

export default LandingPage
