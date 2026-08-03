import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Check, ChevronRight, Cpu, RefreshCw, Server, WalletCards } from 'lucide-react'
import { formatUnits } from 'viem'
import { ConnectWallet } from '../auth/ConnectWallet'
import { DeviceApproval } from '../auth/DeviceApproval'
import { ProviderConsole } from '../provider/ProviderConsole'
import { Money } from '../../components/Money'
import { parseMON } from '../../lib/amount'
import { injectedProvider } from '../../lib/chain'
import { hashLabel, ViemMarketWriter, type MarketWriter, type SubmittedTransaction } from '../../lib/marketContract'
import { AnalyticsAPI, APIError, AuthAPI, InferenceAPI, MarketplaceAPI, OperationsAPI, ReferencePriceAPI, type APIKey, type MarketModel, type Session } from '../../lib/api'
import { deriveConsumerProgress, deriveProviderProgress, rankLiveModels, recommendedStarterSpend, type OnboardingProgress, type OnboardingRole } from './onboarding'

type Props = {
  session?: Session
  initialRole?: OnboardingRole
  authAPI?: AuthAPI
  operationsAPI?: OperationsAPI
  marketplaceAPI?: MarketplaceAPI
  analyticsAPI?: AnalyticsAPI
  referencePriceAPI?: ReferencePriceAPI
  inferenceAPI?: InferenceAPI
  writer?: MarketWriter
  onConnected?: (session: Session) => void
  onSkip?: () => void
  onComplete?: (role: OnboardingRole) => void
  onRoleChange?: (role: OnboardingRole) => void
  onSessionExpired?: () => void
}

function RoleChoice({ choose, skip }: { choose: (role: OnboardingRole) => void; skip?: () => void }) {
  return <section className="onboarding-choice" aria-labelledby="onboarding-choice-title">
    <p className="eyebrow">First route</p>
    <h1 id="onboarding-choice-title">What do you want to do first?</h1>
    <p>Use and host inference from the same account. Start with one path and add the other whenever you are ready.</p>
    <div className="onboarding-role-grid">
      <button type="button" onClick={() => choose('consumer')}><Cpu aria-hidden="true" /><span><strong>Use AI inference</strong><small>Fund an account and run a live model.</small></span><ChevronRight aria-hidden="true" /></button>
      <button type="button" className="secondary-action" onClick={() => choose('provider')}><Server aria-hidden="true" /><span><strong>Host your AI inference</strong><small>Connect an unused machine and earn MON.</small></span><ChevronRight aria-hidden="true" /></button>
    </div>
    {skip && <button type="button" className="onboarding-skip" onClick={skip}>Skip for now</button>}
  </section>
}

function RouteMap({ role, progress }: { role: OnboardingRole; progress: OnboardingProgress }) {
  return <aside className="onboarding-route" aria-label={`${role} setup progress`}>
    <p>{role === 'consumer' ? 'Use inference' : 'Host inference'}</p>
    <ol>{progress.steps.map((step, index) => { const current = progress.next?.id === step.id; return <li key={step.id} data-state={step.complete ? 'complete' : current ? 'active' : 'pending'} aria-current={current ? 'step' : undefined} aria-label={`${step.label}, ${step.complete ? 'complete' : current ? 'current step' : 'not complete'}`}>
      <span>{step.complete ? <Check size={14} aria-hidden="true" /> : String(index + 1).padStart(2, '0')}</span><strong>{step.label}</strong>
    </li>})}</ol>
  </aside>
}

function TransactionStatus({ status, error }: { status: string; error: string }) {
  return <>{status && <p role="status" className="transaction-proof">{status}</p>}{error && <p role="alert" className="inline-error">{error}</p>}</>
}

export function OnboardingFlow({ session, initialRole, authAPI: suppliedAuth, operationsAPI: suppliedOperations, marketplaceAPI: suppliedMarketplace, analyticsAPI: suppliedAnalytics, referencePriceAPI: suppliedPrice, inferenceAPI: suppliedInference, writer: suppliedWriter, onConnected, onSkip, onComplete, onRoleChange, onSessionExpired }: Props) {
  const authAPI = useMemo(() => suppliedAuth ?? new AuthAPI(), [suppliedAuth])
  const operationsAPI = useMemo(() => suppliedOperations ?? new OperationsAPI(), [suppliedOperations])
  const marketplaceAPI = useMemo(() => suppliedMarketplace ?? new MarketplaceAPI(), [suppliedMarketplace])
  const analyticsAPI = useMemo(() => suppliedAnalytics ?? new AnalyticsAPI(), [suppliedAnalytics])
  const referencePriceAPI = useMemo(() => suppliedPrice ?? new ReferencePriceAPI(), [suppliedPrice])
  const inferenceAPI = useMemo(() => suppliedInference ?? new InferenceAPI(), [suppliedInference])
  const [role, setRole] = useState<OnboardingRole | undefined>(initialRole)
  const [modelName, setModelName] = useState('')
  const [depositAmount, setDepositAmount] = useState('')
  const [sessionAllowance, setSessionAllowance] = useState('')
  const [keyMaximum, setKeyMaximum] = useState('')
  const [keySecret, setKeySecret] = useState('')
  const [prompt, setPrompt] = useState('Explain Myference in one short sentence.')
  const [answer, setAnswer] = useState('')
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const [inferenceSucceeded, setInferenceSucceeded] = useState(false)
  const completed = useRef<OnboardingRole | undefined>(undefined)
  const previousRecommendation = useRef(0n)
  const enabled = Boolean(session && role)

  const operations = useQuery({ queryKey: ['onboarding-operations', session?.account_id], queryFn: () => operationsAPI.operations(), enabled, retry: false, refetchInterval: 10_000 })
  const inventory = useQuery({ queryKey: ['onboarding-models'], queryFn: () => marketplaceAPI.models(), enabled: role === 'consumer', retry: false, refetchInterval: 15_000 })
  const keys = useQuery({ queryKey: ['onboarding-keys', session?.account_id], queryFn: () => authAPI.listAPIKeys(), enabled: Boolean(session && role === 'consumer'), retry: false })
  const analytics = useQuery({ queryKey: ['onboarding-analytics', session?.account_id], queryFn: () => analyticsAPI.analytics(), enabled, retry: false, refetchInterval: 15_000 })
  const price = useQuery({ queryKey: ['reference-price'], queryFn: () => referencePriceAPI.price(), enabled: role === 'consumer', retry: false, staleTime: 60_000 })
  const models = useMemo(() => rankLiveModels(inventory.data ?? []), [inventory.data])
  const selectedModel = models.find((model) => model.model === modelName) ?? models[0]
  const recommended = selectedModel ? recommendedStarterSpend(selectedModel) : 0n
  const expired = [operations.error, keys.error, analytics.error].some((reason) => reason instanceof APIError && reason.status === 401)
  const accountSession = expired ? undefined : session

  useEffect(() => {
    if (!modelName && models[0]) setModelName(models[0].model)
  }, [modelName, models])
  useEffect(() => {
    if (!recommended) return
    const previous = previousRecommendation.current ? formatUnits(previousRecommendation.current, 18) : ''
    const next = formatUnits(recommended, 18)
    if (!depositAmount || depositAmount === previous) setDepositAmount(next)
    if (!keyMaximum || keyMaximum === previous) setKeyMaximum(next)
    previousRecommendation.current = recommended
  }, [depositAmount, keyMaximum, recommended])
  useEffect(() => { if (expired) onSessionExpired?.() }, [expired, onSessionExpired])
  useEffect(() => { setKeySecret(''); setInferenceSucceeded(false) }, [modelName])
  useEffect(() => {
    if (!operations.data || sessionAllowance) return
    const available = BigInt(operations.data.customer_balance_wei)
    if (available > 0n) setSessionAllowance(formatUnits(available, 18))
  }, [operations.data, sessionAllowance])

  const consumerProgress = deriveConsumerProgress({ connected: Boolean(accountSession), selectedModel, operations: operations.data, apiKeys: keys.data ?? [], analytics: analytics.data, inferenceSucceeded })
  const providerProgress = deriveProviderProgress({ connected: Boolean(accountSession), operations: operations.data })
  const progress = role === 'provider' ? providerProgress : consumerProgress

  useEffect(() => {
    if (role && progress.complete && completed.current !== role) {
      completed.current = role
      onComplete?.(role)
    }
  }, [onComplete, progress.complete, role])

  const choose = (next: OnboardingRole) => { setRole(next); onRoleChange?.(next); setError(''); setStatus('') }
  if (!role) return <div className="onboarding-screen"><a className="wordmark" href="/" aria-label="Myference home"><span>M/</span> Myference</a><RoleChoice choose={choose} skip={onSkip} /></div>

  const marketWriter = suppliedWriter ?? (operations.data ? new ViemMarketWriter(operations.data, injectedProvider()) : undefined)
  const transact = async (action: (writer: MarketWriter) => Promise<SubmittedTransaction>) => {
    if (!marketWriter) { setError('Account operations are still loading.'); return }
    setError('')
    try {
      const transaction = await action(marketWriter)
      setStatus(`${transaction.hash} pending on Monad Testnet.`)
      await transaction.confirm()
      setStatus('Transaction finalized. Waiting for the indexer to confirm this step…')
      await operations.refetch()
    } catch (reason) { setStatus(''); setError(reason instanceof Error ? reason.message : 'Transaction failed.') }
  }
  const deposit = async (event: FormEvent) => { event.preventDefault(); await transact((writer) => writer.deposit(parseMON(depositAmount))) }
  const openSession = async (event: FormEvent) => {
    event.preventDefault()
    await transact((writer) => writer.openSession(hashLabel(crypto.randomUUID()), parseMON(sessionAllowance), BigInt(Math.floor(Date.now() / 1000) + 86_400)))
  }
  const createKey = async (event: FormEvent) => {
    event.preventDefault(); setError('')
    if (!selectedModel) return
    try {
      const key = await authAPI.createAPIKey({ models: [selectedModel.model], endpoints: ['/v1/chat/completions', '/v1/messages'], max_spend_wei: parseMON(keyMaximum).toString() })
      if (!key.token) throw new Error('The API key was created without a one-time secret.')
      setKeySecret(key.token)
      await keys.refetch()
      setStatus('API key created. It is loaded into the test request below; copy it now if you also want to use it elsewhere.')
    } catch (reason) { setError(reason instanceof Error ? reason.message : 'API key creation failed.') }
  }
  const infer = async (event: FormEvent) => {
    event.preventDefault(); setError(''); setAnswer('')
    if (!selectedModel) return
    try {
      const response = await inferenceAPI.chat(selectedModel.model, keySecret.trim(), parseMON(keyMaximum).toString(), [{ role: 'user', content: prompt.trim() }], 1_000)
      setAnswer(response); setInferenceSucceeded(true); setStatus('Your first live inference completed.')
    } catch (reason) { setError(reason instanceof Error ? reason.message : 'Inference request failed.') }
  }

  const loading = Boolean(accountSession && (operations.isPending || analytics.isPending || (role === 'consumer' && (inventory.isPending || keys.isPending))))
  return <div className="onboarding-screen onboarding-active">
    <header className="onboarding-header"><a className="wordmark" href="/" aria-label="Myference home"><span>M/</span> Myference</a><div><button type="button" className="onboarding-switch" onClick={() => choose(role === 'consumer' ? 'provider' : 'consumer')}>{role === 'consumer' ? 'Host instead' : 'Use inference instead'}</button>{onSkip && <button type="button" className="onboarding-skip" onClick={onSkip}>Skip for now</button>}</div></header>
    <div className="onboarding-layout">
      <RouteMap role={role} progress={progress} />
      <main className="onboarding-content" aria-live="polite">
        {!accountSession ? <section className="onboarding-step"><p className="eyebrow">Step 1</p><h1>Connect your account</h1><p>{expired ? 'Your browser session expired. Sign again to continue from your on-chain state.' : 'Your wallet signs a login message. No MON moves until you approve a separate transaction.'}</p><ConnectWallet api={authAPI} onConnected={onConnected} /></section>
        : loading ? <div className="dashboard-empty" role="status"><strong>Reading your account state…</strong><p>Checking Monad Testnet and the Myference indexer.</p></div>
        : operations.isError ? <section className="onboarding-step"><p className="eyebrow">Account state</p><h1>Try the indexer again</h1><p role="alert">Your account state could not be loaded. No transaction was sent.</p><button type="button" onClick={() => void operations.refetch()}><RefreshCw size={16} aria-hidden="true" /> Retry account state</button></section>
        : role === 'consumer' ? <ConsumerSteps models={models} selectedModel={selectedModel} modelName={modelName} setModelName={setModelName} inventoryError={inventory.isError} refreshInventory={() => void inventory.refetch()} operationsError={operations.isError} operations={operations.data} keys={keys.data ?? []} keySecret={keySecret} recommended={recommended} usdPerMON={price.data?.usd_per_mon} depositAmount={depositAmount} setDepositAmount={setDepositAmount} deposit={deposit} sessionAllowance={sessionAllowance} setSessionAllowance={setSessionAllowance} openSession={openSession} keyMaximum={keyMaximum} setKeyMaximum={setKeyMaximum} createKey={createKey} prompt={prompt} setPrompt={setPrompt} infer={infer} answer={answer} progress={consumerProgress} finish={onSkip} />
        : <ProviderSteps authAPI={authAPI} operationsAPI={operationsAPI} operations={operations.data} writer={marketWriter} progress={providerProgress} finish={onSkip} />}
        <TransactionStatus status={status} error={error} />
      </main>
    </div>
  </div>
}

export function OnboardingReminder({ role, session, onContinue, onSwitch, onComplete, onSessionExpired }: { role: OnboardingRole; session?: Session; onContinue: () => void; onSwitch: () => void; onComplete?: (role: OnboardingRole) => void; onSessionExpired?: () => void }) {
  const operations = useQuery({ queryKey: ['onboarding-operations', session?.account_id], queryFn: () => new OperationsAPI().operations(), enabled: Boolean(session), retry: false, refetchInterval: 15_000 })
  const analytics = useQuery({ queryKey: ['onboarding-analytics', session?.account_id], queryFn: () => new AnalyticsAPI().analytics(), enabled: Boolean(session), retry: false, refetchInterval: 15_000 })
  const keys = useQuery({ queryKey: ['onboarding-keys', session?.account_id], queryFn: () => new AuthAPI().listAPIKeys(), enabled: Boolean(session && role === 'consumer'), retry: false })
  const inventory = useQuery({ queryKey: ['onboarding-models'], queryFn: () => new MarketplaceAPI().models(), enabled: role === 'consumer', retry: false })
  const selectedModel = rankLiveModels(inventory.data ?? [])[0]
  const expired = [operations.error, keys.error, analytics.error].some((reason) => reason instanceof APIError && reason.status === 401)
  const connected = Boolean(session && !expired)
  const progress = role === 'consumer'
    ? deriveConsumerProgress({ connected, selectedModel, operations: operations.data, apiKeys: keys.data ?? [], analytics: analytics.data })
    : deriveProviderProgress({ connected, operations: operations.data })
  const reported = useRef(false)
  useEffect(() => {
    if (progress.complete && !reported.current) { reported.current = true; onComplete?.(role) }
  }, [onComplete, progress.complete, role])
  useEffect(() => { if (expired) onSessionExpired?.() }, [expired, onSessionExpired])
  return <section className="onboarding-reminder" aria-label="Onboarding reminder">
    <div><p className="eyebrow">{progress.completed} of {progress.steps.length} steps</p><strong>Finish your first live route</strong><span>{role === 'consumer' ? 'Connect, fund, and run a real inference.' : 'Connect a machine, bond collateral, and publish a live offer.'}</span></div>
    <div><button type="button" className="secondary-action" onClick={onSwitch}>{role === 'consumer' ? 'Switch to hosting' : 'Switch to using'}</button><button type="button" onClick={onContinue}>Continue setup</button></div>
  </section>
}

type ConsumerProps = {
  models: MarketModel[]; selectedModel?: MarketModel; modelName: string; setModelName: (value: string) => void; inventoryError: boolean; refreshInventory: () => void; operationsError: boolean
  operations?: Awaited<ReturnType<OperationsAPI['operations']>>; keys: APIKey[]; keySecret: string; recommended: bigint; usdPerMON?: string
  depositAmount: string; setDepositAmount: (value: string) => void; deposit: (event: FormEvent) => void
  sessionAllowance: string; setSessionAllowance: (value: string) => void; openSession: (event: FormEvent) => void
  keyMaximum: string; setKeyMaximum: (value: string) => void; createKey: (event: FormEvent) => void
  prompt: string; setPrompt: (value: string) => void; infer: (event: FormEvent) => void; answer: string; progress: OnboardingProgress; finish?: () => void
}

function ConsumerSteps(props: ConsumerProps) {
  if (props.inventoryError || props.models.length === 0) return <section className="onboarding-step"><p className="eyebrow">Live inventory</p><h1>No model is ready yet</h1><p role="alert">No live inference models are available. Myference will never show a fake provider.</p><button type="button" onClick={props.refreshInventory}><RefreshCw size={16} aria-hidden="true" /> Refresh inventory</button></section>
  if (props.operationsError || !props.operations) return <section className="onboarding-step"><h1>Account state is unavailable</h1><p role="alert">Reconnect your wallet or wait for the indexer, then retry.</p></section>
  const next = props.progress.next?.id
  const activeSession = props.operations.sessions.find((item) => !item.finalized && item.expires_at > Date.now() / 1000 && BigInt(item.allowance_wei) > BigInt(item.spent_wei))
  return <>
    <section className="onboarding-model-picker"><label htmlFor="onboarding-model">Model</label><select id="onboarding-model" value={props.selectedModel?.model ?? ''} onChange={(event) => props.setModelName(event.target.value)}>{props.models.map((model) => <option key={model.model} value={model.model}>{model.model}</option>)}</select><span>{props.selectedModel?.model === props.models[0]?.model ? 'Lowest published live rate' : 'Your selected route'}</span></section>
    {props.keys.length > 0 && !props.keySecret && <p className="onboarding-notice">An existing key's secret cannot be recovered. Create a replacement key below for this browser test.</p>}
    {next === 'deposit' && <section className="onboarding-step"><p className="eyebrow">Fund requests</p><h1>Deposit a small MON budget</h1><p>Funds stay in your contract balance until a settled request spends them or you withdraw them.</p><div className="onboarding-proof"><span>Recommended starter amount</span><strong><Money wei={props.recommended} /></strong>{props.usdPerMON && <small>Live USD reference: 1 MON = ${props.usdPerMON}</small>}</div><form onSubmit={props.deposit}><label htmlFor="onboarding-deposit">Deposit amount (MON)</label><input id="onboarding-deposit" inputMode="decimal" value={props.depositAmount} onChange={(event) => props.setDepositAmount(event.target.value)} required/><button type="submit"><WalletCards size={17} aria-hidden="true" /> Deposit MON</button></form></section>}
    {next === 'session' && <section className="onboarding-step"><p className="eyebrow">Set a limit</p><h1>Open a 24-hour spending session</h1><p>This is the maximum the router may reserve and settle during the session. It cannot exceed your escrow balance.</p><form onSubmit={props.openSession}><label htmlFor="onboarding-allowance">Session allowance (MON)</label><input id="onboarding-allowance" inputMode="decimal" value={props.sessionAllowance} onChange={(event) => props.setSessionAllowance(event.target.value)} required/><button type="submit">Open spending session</button></form></section>}
    {(next === 'key' || (props.keys.length > 0 && !props.keySecret)) && <section className="onboarding-step"><p className="eyebrow">Create access</p><h1>{props.keys.length ? 'Create a replacement key' : 'Create your API key'}</h1><p>The key is limited to {props.selectedModel?.model} and the amount below. Its secret appears once.</p><form onSubmit={props.createKey}><label htmlFor="onboarding-key-max">Key maximum spend (MON)</label><input id="onboarding-key-max" inputMode="decimal" value={props.keyMaximum} onChange={(event) => props.setKeyMaximum(event.target.value)} required/><button type="submit">{props.keys.length ? 'Create replacement key' : 'Create API key'}</button></form></section>}
    {next === 'inference' && props.keySecret && <section className="onboarding-step"><p className="eyebrow">Live test</p><h1>Run your first inference</h1><div className="secret-proof"><span>Copy now — shown once</span><code>{props.keySecret}</code></div><form onSubmit={props.infer}><label htmlFor="onboarding-prompt">Message</label><textarea id="onboarding-prompt" rows={5} value={props.prompt} onChange={(event) => props.setPrompt(event.target.value)} required/><button type="submit">Send live request</button></form>{props.answer && <div className="onboarding-answer" role="status"><span>Provider response</span><p>{props.answer}</p></div>}</section>}
    {props.progress.complete && <section className="onboarding-step onboarding-complete"><Check aria-hidden="true" /><p className="eyebrow">Route complete</p><h1>Your account is ready</h1><p>Your inference ran through a real provider and the usage record will appear after settlement is indexed.</p>{props.answer && <div className="onboarding-answer" role="status"><span>Provider response</span><p>{props.answer}</p></div>}{props.finish && <button type="button" onClick={props.finish}>Open dashboard</button>}</section>}
    {activeSession && next !== 'session' && <p className="onboarding-footnote">Active session: <Money wei={BigInt(activeSession.allowance_wei) - BigInt(activeSession.spent_wei)} /> remaining.</p>}
  </>
}

function ProviderSteps({ authAPI, operationsAPI, operations, writer, progress, finish }: { authAPI: AuthAPI; operationsAPI: OperationsAPI; operations?: Awaited<ReturnType<OperationsAPI['operations']>>; writer?: MarketWriter; progress: OnboardingProgress; finish?: () => void }) {
  const windows = typeof navigator !== 'undefined' && /Windows/i.test(navigator.userAgent)
  const install = windows ? 'irm https://myference.xyz/install.ps1 | iex' : 'curl -fsSL https://myference.xyz/install.sh | sh'
  const next = progress.next?.id
  if (!operations) return <div className="dashboard-empty" role="status"><strong>Loading provider account…</strong></div>
  return <>
    {next === 'machine' && <section className="onboarding-step"><p className="eyebrow">Connect a machine</p><h1>Turn this computer into a provider</h1><p>Install the CLI, then start the guided host command. It discovers the backend and opens a browser device code for this account.</p><div className="onboarding-command"><span>{windows ? 'Windows PowerShell' : 'macOS / Linux'}</span><code>{install}</code></div><div className="onboarding-command"><span>Configure and run</span><code>myference host</code></div><div className="onboarding-command"><span>Keep it running after restart</span><code>myference service install</code></div><DeviceApproval api={authAPI} /></section>}
    {(next === 'bond' || next === 'offer') && <section className="onboarding-step onboarding-provider-console"><p className="eyebrow">{next === 'bond' ? 'Collateral' : 'Publish and activate'}</p><h1>{next === 'bond' ? 'Back your provider' : 'Put the discovered model live'}</h1><p>{next === 'bond' ? 'Collateral remains yours unless a proven serving violation is slashed.' : 'Set readable rates, publish the immutable version on Monad, then run the displayed CLI sync command.'}</p><ProviderConsole api={operationsAPI} writer={writer} /></section>}
    {progress.complete && <section className="onboarding-step onboarding-complete"><Check aria-hidden="true" /><p className="eyebrow">Provider live</p><h1>Your machine can receive requests</h1><p>Keep the daemon running. Capacity, requests, settlement, earnings, and slashing remain visible in the hosting dashboard.</p>{finish && <button type="button" onClick={finish}>Open dashboard</button>}</section>}
  </>
}
