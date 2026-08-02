import { publicAPIBaseURL } from '../../lib/api'

export function ApiAccessGuide() {
  const baseURL = publicAPIBaseURL()
  return <section className="api-guide" aria-labelledby="api-guide-title">
    <p className="eyebrow">Compatible endpoints</p><h2 id="api-guide-title">Connect your application</h2>
    <div className="endpoint-grid"><article><span>Base URL</span><code>{baseURL}/v1</code></article><article><span>OpenAI-compatible</span><code>POST /v1/chat/completions</code></article><article><span>Anthropic-compatible</span><code>POST /v1/messages</code></article></div>
    <pre><code>{`curl ${baseURL}/v1/chat/completions \\\n  -H "Authorization: Bearer $MYFERENCE_API_KEY" \\\n  -H "Content-Type: application/json"`}</code></pre>
  </section>
}
