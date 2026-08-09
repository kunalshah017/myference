import { useState, type ReactNode } from 'react'
import { ArrowRight, BookOpen, Check, Clipboard, Cloud, Code2, Cpu, ExternalLink, KeyRound, Laptop, LockKeyhole, Server, ShieldCheck, Terminal, TriangleAlert, WalletCards } from 'lucide-react'

const API_URL = 'https://api.myference.xyz'
const RELEASE_TAG = 'v0.2.0-alpha.1'
const RELEASE_URL = `https://github.com/kunalshah017/myference/releases/download/${RELEASE_TAG}`

const navigation = [
  ['start', 'Start here'], ['use', 'Use inference'], ['openai', 'OpenAI API'], ['anthropic', 'Anthropic API'],
  ['host', 'Host inference'], ['backends', 'Backend options'], ['settlement', 'Pricing & settlement'],
  ['security', 'Security'], ['troubleshooting', 'Troubleshooting'], ['reference', 'Reference'],
]

function CopyBlock({ label, children }: { label: string; children: string }) {
  const [copied, setCopied] = useState(false)
  const copy = async () => {
    await navigator.clipboard?.writeText(children)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1800)
  }
  return <div className="docs-code">
    <div><span>{label}</span><button type="button" onClick={() => void copy()} aria-label={`Copy ${label}`}><Clipboard size={14} />{copied ? 'Copied' : 'Copy'}</button></div>
    <pre><code>{children}</code></pre>
  </div>
}

function Step({ number, title, children }: { number: string; title: string; children: ReactNode }) {
  return <li className="docs-step"><span>{number}</span><div><h3>{title}</h3>{children}</div></li>
}

function DocsPage() {
  return <div className="docs-page">
    <header className="docs-header">
      <a className="wordmark" href="/" aria-label="Myference home"><span>M/</span> Myference</a>
      <nav aria-label="Documentation header"><a aria-current="page" href="/docs">Docs</a><a href="https://github.com/kunalshah017/myference">GitHub <ExternalLink size={13} /></a></nav>
      <a className="landing-primary compact" href="/app">Open app <ArrowRight size={15} /></a>
    </header>

    <section className="docs-hero">
      <div><p className="eyebrow">Product documentation · Monad Testnet</p><h1>Build with Myference.</h1><p>Use hosted AI through familiar APIs, or turn a Windows or macOS machine into a provider.</p></div>
      <div className="docs-endpoint"><span>LIVE API</span><code>{API_URL}</code><small>Native MON settlement on Monad Testnet</small></div>
    </section>

    <div className="docs-layout">
      <aside className="docs-sidebar">
        <p>Documentation</p>
        <nav aria-label="Documentation sections">{navigation.map(([id, label]) => <a key={id} href={`#${id}`}>{label}</a>)}</nav>
        <a className="docs-sidebar-cta" href="/app"><Cpu size={16} /> Browse live models</a>
      </aside>

      <main className="docs-content">
        <section id="start" className="docs-section">
          <p className="eyebrow">Start here</p><h2>Choose your path.</h2>
          <p className="docs-lead">One wallet can consume and provide inference. Start with either path and switch at any time.</p>
          <div className="docs-paths">
            <a href="#use"><WalletCards /><span><strong>Use inference</strong><small>Fund an account, create a key, and call a model.</small></span><ArrowRight /></a>
            <a href="#host"><Server /><span><strong>Host inference</strong><small>Connect unused compute and earn MON.</small></span><ArrowRight /></a>
          </div>
          <div className="docs-callout info"><BookOpen size={19} /><div><strong>Testnet first</strong><p>Use a Monad Testnet wallet and test MON. Prices and settlement use native MON; USD values in the app are informational references only.</p></div></div>
        </section>

        <section id="use" className="docs-section">
          <p className="eyebrow">Consumer guide</p><h2>Use hosted inference.</h2>
          <ol className="docs-steps">
            <Step number="01" title="Connect your wallet"><p>Open the <a href="/app">dashboard</a>, choose <strong>Connect wallet</strong>, and sign the browser login message. The signature proves wallet ownership; it is not a payment transaction.</p></Step>
            <Step number="02" title="Deposit native MON"><p>Open <strong>Funds</strong>, enter a MON amount, and confirm <strong>Deposit to escrow</strong> in your wallet. Keep a little MON outside escrow for gas.</p></Step>
            <Step number="03" title="Open a bounded session"><p>Set an allowance and duration under <strong>Bounded spending sessions</strong>. Requests can spend only from finalized session allowance, never from your wallet directly.</p></Step>
            <Step number="04" title="Choose a live model"><p>Open <strong>Models</strong> to compare providers, token rates, compute rates, health, and evidence. Stale inventory is marked unavailable and is never selected for a request.</p></Step>
            <Step number="05" title="Create an API key"><p>Open <strong>API access</strong> and set a maximum spend in MON. New keys work with every model by default. For a least-privilege project key, enable the optional model restriction and choose from the live catalog. Copy the secret when shown—it is displayed once.</p></Step>
            <Step number="06" title="Send a real request"><p>Use the in-browser <strong>Playground</strong>, or call either compatibility API below. Myference currently requires streaming requests.</p></Step>
            <Step number="07" title="Review and exit"><p>Use <strong>Usage</strong> to follow reservation, provider streaming, signatures, submission, and final settlement. Revoke unused keys, request session close, finalize it after the displayed delay, then withdraw or claim available MON.</p></Step>
          </ol>
          <div className="docs-callout warning"><TriangleAlert size={19} /><div><strong>The request limit is a reservation ceiling</strong><p><code>X-Myference-Max-Spend</code> is an integer amount in wei MON. It must fit both the API-key scope and an open session. Actual measured usage is charged; unused allowance remains yours. A request is stopped rather than charging beyond its signed maximum.</p></div></div>
        </section>

        <section id="openai" className="docs-section">
          <p className="eyebrow">Compatibility API</p><h2>OpenAI-compatible chat.</h2>
          <p>Point a streaming chat-completions client at <code>{API_URL}/v1</code>. Use your Myference key as the bearer token and add a per-request maximum in wei MON.</p>
          <CopyBlock label="curl">{`curl ${API_URL}/v1/chat/completions \\
  -H "Authorization: Bearer $MYFERENCE_API_KEY" \\
  -H "Content-Type: application/json" \\
  -H "X-Myference-Max-Spend: 1000000000000000" \\
  -d '{"model":"YOUR_MODEL_ID","stream":true,"max_completion_tokens":256,"messages":[{"role":"user","content":"Say hello in one sentence."}]}'`}</CopyBlock>
          <CopyBlock label="JavaScript · OpenAI SDK">{`import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.MYFERENCE_API_KEY,
  baseURL: "${API_URL}/v1",
  defaultHeaders: { "X-Myference-Max-Spend": "1000000000000000" },
});

const stream = await client.chat.completions.create({
  model: "YOUR_MODEL_ID",
  stream: true,
  max_completion_tokens: 256,
  messages: [{ role: "user", content: "Say hello in one sentence." }],
});
for await (const event of stream) process.stdout.write(event.choices[0]?.delta?.content ?? "");`}</CopyBlock>
          <CopyBlock label="Python · OpenAI SDK">{`import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ["MYFERENCE_API_KEY"],
    base_url="${API_URL}/v1",
    default_headers={"X-Myference-Max-Spend": "1000000000000000"},
)
stream = client.chat.completions.create(
    model="YOUR_MODEL_ID",
    stream=True,
    max_completion_tokens=256,
    messages=[{"role": "user", "content": "Say hello in one sentence."}],
)
for event in stream:
    print(event.choices[0].delta.content or "", end="")`}</CopyBlock>
          <div className="docs-callout info"><Code2 size={19} /><div><strong>Choose the exact live model ID</strong><p>Copy it from Models or Playground. Default keys can call any model; restricted keys can call only the models selected from the live catalog when the key was created.</p></div></div>
        </section>

        <section id="anthropic" className="docs-section">
          <p className="eyebrow">Compatibility API</p><h2>Anthropic-compatible messages.</h2>
          <p>Use the same Myference API key with the Messages API. The currently supported protocol version is <code>2023-06-01</code>.</p>
          <CopyBlock label="curl">{`curl ${API_URL}/v1/messages \\
  -H "x-api-key: $MYFERENCE_API_KEY" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "Content-Type: application/json" \\
  -H "X-Myference-Max-Spend: 1000000000000000" \\
  -d '{"model":"YOUR_MODEL_ID","max_tokens":256,"stream":true,"messages":[{"role":"user","content":"Say hello in one sentence."}]}'`}</CopyBlock>
          <p className="docs-note">The response is native Anthropic server-sent events: message start, content deltas, message delta, and message stop. Non-streaming requests are rejected.</p>
        </section>

        <section id="host" className="docs-section">
          <p className="eyebrow">Provider guide</p><h2>Host from your machine.</h2>
          <p className="docs-lead">The shortest path uses a local Ollama model. The CLI connects outbound to Myference, so you do not expose an inbound port or put your laptop directly on the public internet.</p>
          <CopyBlock label="Windows PowerShell">{`irm https://myference.xyz/install.ps1 | iex`}</CopyBlock>
          <CopyBlock label="macOS Terminal">{`curl -fsSL https://myference.xyz/install.sh | sh`}</CopyBlock>
          <p className="docs-note">The installer selects Windows AMD64, macOS Intel, or macOS Apple Silicon automatically, verifies the published SHA-256 checksum, and installs the CLI with its Linux container proxy. Windows updates your user PATH; macOS installs into <code>/usr/local/bin</code> and asks for <code>sudo</code> only when needed.</p>
          <p className="eyebrow">Manual downloads</p>
          <div className="docs-downloads">
            <a href={`${RELEASE_URL}/myference-windows-amd64-${RELEASE_TAG}.zip`}><Laptop /><span><strong>Windows 64-bit</strong><small>myference-windows-amd64</small></span><ArrowRight /></a>
            <a href={`${RELEASE_URL}/myference-macos-arm64-${RELEASE_TAG}.zip`}><Laptop /><span><strong>macOS Apple Silicon</strong><small>myference-macos-arm64</small></span><ArrowRight /></a>
            <a href={`${RELEASE_URL}/myference-macos-amd64-${RELEASE_TAG}.zip`}><Laptop /><span><strong>macOS Intel</strong><small>myference-macos-amd64</small></span><ArrowRight /></a>
          </div>
          <div className="docs-callout info"><ShieldCheck size={19} /><div><strong>Verify the download</strong><p>Download <code>SHA256SUMS</code> from the same release and compare the artifact checksum before running it. Release packages are not currently code-signed or notarized.</p></div></div>
          <ol className="docs-steps">
            <Step number="01" title="Install the CLI">
              <p>Run the one-line installer above, then open a new terminal if Windows has just added Myference to your PATH. Use the manual artifacts only when you need an offline or pinned-version installation. Release packages are checksum-verified but are not yet code-signed or notarized.</p>
            </Step>
            <Step number="02" title="Install Ollama and a model"><p>Install Ollama from its official distribution, keep it on its default loopback address, and pull the model you want to serve.</p><CopyBlock label="Terminal or PowerShell">{`ollama pull qwen2.5:0.5b
ollama list`}</CopyBlock></Step>
            <Step number="03" title="Connect and discover"><p>Run Myference with no arguments to open the terminal UI. It discovers every installed Ollama model plus Codex and Claude CLIs, and it can add OpenAI or any OpenAI-compatible endpoint from its model catalog. Select several providers and start them together.</p><CopyBlock label="Terminal or PowerShell">{`myference`}</CopyBlock><p className="docs-note">Use <code>.\myference.exe</code> on Windows if its folder is not on PATH. Existing commands such as <code>myference host --model qwen2.5:0.5b</code> remain available for automation.</p></Step>
            <Step number="04" title="Approve the machine"><p>In the opened browser, connect the provider wallet, confirm the displayed device code, and approve its generated signer. The signer private key and machine token stay in Windows Credential Manager or macOS Keychain.</p></Step>
				<Step number="05" title="Stake and publish"><p>Stay in the terminal UI to choose Collateral or Offers &amp; Pricing. Compatible offers already owned by this wallet can be attached from another machine without a new transaction. For a new offer, enter MON values there; Myference opens only a minimal short-lived wallet page showing the exact action. The CLI resumes after finalized indexed chain state proves the action. Your wallet key never enters the CLI.</p></Step>
			<Step number="06" title="Keep versions synchronized"><p>Every price change creates a new immutable offer version. A running provider checks account-owned compatible versions every 15 seconds, saves the confirmed version atomically, and reloads capacity automatically. The provider web page may reprice an existing offer, but only the CLI can create an offer or change its model identity.</p></Step>
			<Step number="07" title="Run in the background"><p>First stop the foreground process with Ctrl+C and configure without serving, then install and start the shared provider service:</p><CopyBlock label="Terminal or PowerShell">{`myference host --model qwen2.5:0.5b --setup-only
myference service install
myference service start
myference service status`}</CopyBlock><p className="docs-note">Windows uses a per-user Scheduled Task and macOS uses a per-user LaunchAgent. Both run the same outbound-only provider service.</p></Step>
			<Step number="08" title="Monitor and manage"><p>The terminal UI shows hosting health and controls provider setup. The focused provider account page shows live requests, settlement, earnings, collateral, and existing-offer repricing; it does not discover models or start hosting.</p></Step>
          </ol>
        </section>

        <section id="backends" className="docs-section">
          <p className="eyebrow">Provider configuration</p><h2>Backend options.</h2>
          <div className="docs-table-wrap"><table><thead><tr><th>Backend</th><th>Evidence</th><th>Metering</th><th>Requirements</th></tr></thead><tbody>
            <tr><td><Cpu /> Ollama</td><td>Runtime model digest</td><td>Input, output, compute</td><td>Loopback Ollama</td></tr>
            <tr><td><Cloud /> OpenAI-compatible</td><td>Returned model identifier</td><td>Input, output, compute</td><td>HTTPS endpoint + key</td></tr>
            <tr><td><Terminal /> Codex / Claude / Kimi</td><td>Digest-pinned container image</td><td>Compute only by default</td><td>Docker Desktop + image + key</td></tr>
          </tbody></table></div>
          <CopyBlock label="OpenAI-compatible provider">{`myference backend add --kind openai --name cloud-model \\
  --model YOUR_UPSTREAM_MODEL --url https://YOUR_PROVIDER \\
  --secret "$PROVIDER_API_KEY"`}</CopyBlock>
          <CopyBlock label="Isolated CLI agent">{`myference backend add --kind codex --name codex-agent \\
  --model YOUR_CODEX_MODEL \\
  --image ghcr.io/kunalshah017/myference-codex@sha256:YOUR_VERIFIED_DIGEST \\
  --secret "$OPENAI_API_KEY"`}</CopyBlock>
          <p>Replace <code>codex</code> with <code>claude</code> or <code>kimi</code> for those runners. Command agents receive only bounded disposable workspaces, run in isolated containers, and are advertised as compute-only unless a trustworthy upstream token count is available. The public API exposes model responses only; it does not expose Codex shell, MCP, filesystem, Docker, or other agent tools.</p>
          <CopyBlock label="Manage independently">{`myference backend list
myference backend stop --name cloud-model
myference backend start --name cloud-model
myference capacity`}</CopyBlock>
        </section>

        <section id="settlement" className="docs-section">
          <p className="eyebrow">Native MON</p><h2>Pricing and settlement.</h2>
          <div className="docs-facts">
            <article><strong>Immutable prices</strong><p>Each provider price change creates a new on-chain offer version. An accepted request keeps the version it started with.</p></article>
            <article><strong>Measured charge</strong><p>Token-aware backends report input/output usage and compute time. CLI agents default to measured compute only.</p></article>
            <article><strong>Bounded exposure</strong><p>The API header, key scope, and session allowance cap the request. The signed receipt cannot exceed its maximum charge.</p></article>
            <article><strong>Versioned fee</strong><p>The deployed contract starts at a 5% protocol fee: 95% becomes provider-claimable and 5% fee-recipient-claimable. Delayed, capped fee updates apply by version.</p></article>
          </div>
          <p>After streaming completes, the provider machine and settlement service sign the same EIP-712 receipt. The server submits it to Monad, waits for finality, then updates balances. Provider revenue and protocol fees are claimable rather than pushed automatically.</p>
          <div className="docs-callout warning"><LockKeyhole size={19} /><div><strong>Deposits and bonds are controlled exits</strong><p>Customer funds move from wallet to escrow, then into bounded sessions. Closing a session and exiting a provider bond require request/finalize steps with the contract's displayed delay. This prevents in-flight obligations from being withdrawn underneath a request.</p></div></div>
        </section>

        <section id="security" className="docs-section">
          <p className="eyebrow">Trust boundaries</p><h2>Security and model evidence.</h2>
          <ul className="docs-checks">
            <li><Check /> Provider connections are outbound; no public inbound machine port is required.</li>
            <li><Check /> Long-lived machine, signer, and backend secrets use the OS credential vault, not the JSON config.</li>
            <li><Check /> Provider and broker co-sign the exact usage receipt; contract nonces prevent replay.</li>
            <li><Check /> Ollama is identified by its local runtime digest; compatible clouds by upstream model identity; agents by a pinned container digest.</li>
            <li><Check /> A digest proves the runtime artifact being advertised, not response quality or that two differently packaged models have identical weights.</li>
            <li><Check /> Agent workspaces reject absolute/traversal paths, invalid base64, more than 64 files, or more than 512 KiB decoded content.</li>
            <li><Check /> The public API exposes model responses only; command execution remains private to the disposable, read-only container and has no Docker socket or host-home mount.</li>
            <li><Check /> API keys are shown once and revocable. Model restrictions are optional; endpoint and maximum-spend limits still constrain every key.</li>
          </ul>
        </section>

        <section id="troubleshooting" className="docs-section">
          <p className="eyebrow">Common failures</p><h2>Troubleshooting.</h2>
          <div className="docs-troubleshooting">
            <details><summary>401 Unauthorized</summary><p>Use the newly created <code>mf_…</code> key, not a wallet address or provider secret. OpenAI calls need <code>Authorization: Bearer</code>; Anthropic calls need <code>x-api-key</code>. Recreate the key if it was revoked or not copied when first shown.</p></details>
            <details><summary>No routable provider or model</summary><p>Copy the exact live model ID. The provider must be connected, healthy, bonded, signer-approved, on the published offer version, and have available capacity.</p></details>
            <details><summary><code>myference host</code> cannot find Ollama</summary><p>Start Ollama, confirm <code>ollama list</code> shows the model, and keep the default <code>http://127.0.0.1:11434</code>. For another loopback port, pass <code>--ollama-url</code>.</p></details>
            <details><summary>Machine approval remains pending</summary><p>Use the same browser account you intend to host from, enter the displayed device code, and confirm the signer transaction in the wallet. The CLI keeps polling until approval or cancellation.</p></details>
            <details><summary>Offer is stale after a price change</summary><p>Wait for the publish transaction and indexer confirmation, then run <code>myference backend version --name NAME --price-version VERSION</code>. The daemon reloads the config automatically.</p></details>
            <details><summary>Codex, Claude, or Kimi backend will not start</summary><p>The image reference must include an immutable <code>@sha256:</code> digest, and the runner credential must be valid. On Windows, Myference starts Docker Desktop and pulls missing digest-pinned images during provider startup. Confirm Docker Desktop is configured for Linux containers, then run <code>myference serve</code> in the foreground to see the exact startup error.</p></details>
            <details><summary>USD reference is unavailable</summary><p>Settlement is unaffected. USD is display-only; exact immutable MON rates remain authoritative.</p></details>
            <details><summary>A request exceeds its estimate</summary><p>The router reserves the declared maximum before work starts. Generation is bounded by the request budget, and settlement rejects any receipt above the signed maximum rather than overdrawing the session.</p></details>
          </div>
        </section>

        <section id="reference" className="docs-section">
          <p className="eyebrow">Quick reference</p><h2>Commands and endpoints.</h2>
		  <div className="docs-reference-grid"><article><h3>CLI</h3><code>myference</code><code>myference offer publish|attach|list|sync</code><code>myference collateral status|deposit|request-exit|finalize-exit</code><code>myference backend add|list|start|stop|version</code><code>myference host</code><code>myference status --json</code><code>myference serve</code><code>myference service install|start|stop|status|uninstall</code></article><article><h3>Public API</h3><code>POST /v1/chat/completions</code><code>POST /v1/messages</code><p>Both require <code>stream: true</code>, an API key, and <code>X-Myference-Max-Spend</code> in wei MON.</p></article></div>
          <div className="docs-final-actions"><a className="landing-primary" href="/app"><KeyRound size={17} /> Create an API key</a><a className="landing-secondary" href="/host"><Server size={17} /> Open provider workspace</a><a href="https://github.com/kunalshah017/myference">Read the source <ExternalLink size={15} /></a></div>
        </section>
      </main>

      <aside className="docs-on-page"><p>On this page</p><nav aria-label="On this page"><a href="#use">Consumer flow</a><a href="#openai">OpenAI API</a><a href="#host">Provider flow</a><a href="#settlement">Settlement</a><a href="#troubleshooting">Troubleshooting</a></nav></aside>
    </div>

    <footer className="docs-footer"><a className="wordmark" href="/"><span>M/</span> Myference</a><span>REAL INFERENCE / PROVABLE SETTLEMENT</span><a href="#start">Back to top ↑</a></footer>
  </div>
}

export default DocsPage
