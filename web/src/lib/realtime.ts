import type { AuthAPI } from './api'

const terminalStates = new Set(['settled', 'failed', 'cancelled', 'expired', 'rejected'])

export type RequestEventState = { cursor: number; state: string; needsRefetch: boolean }
export type RequestEvent = { cursor: number; state: string }

export function reconcileRequestEvent(current: RequestEventState, event: RequestEvent): RequestEventState {
  if (event.cursor <= current.cursor) return current
  return {
    cursor: event.cursor,
    state: terminalStates.has(current.state) ? current.state : event.state,
    needsRefetch: current.needsRefetch || event.cursor > current.cursor + 1,
  }
}

const eventNames = ['request.reserved', 'request.offered', 'request.accepted', 'request.streaming', 'request.completed', 'request.signed', 'request.submitted', 'request.settled', 'request.failed', 'request.cancelled', 'request.expired', 'request.rejected']

export function subscribeToActivity(api: AuthAPI, onEvent: (event: RequestEvent) => void, onReconnect: () => void): () => void {
  let source: EventSource | undefined
  let stopped = false
  let reconnect: number | undefined
  const connect = async () => {
    try {
      const ticket = await api.streamTicket()
      if (stopped) return
      source = new EventSource(api.eventsURL(ticket.ticket), { withCredentials: true })
      for (const name of eventNames) source.addEventListener(name, (raw) => {
        const message = raw as MessageEvent
        onEvent({ cursor: Number(message.lastEventId), state: name.slice('request.'.length) })
      })
      source.onerror = () => {
        source?.close()
        if (!stopped) { onReconnect(); reconnect = window.setTimeout(() => void connect(), 1000) }
      }
    } catch {
      if (!stopped) { onReconnect(); reconnect = window.setTimeout(() => void connect(), 1000) }
    }
  }
  void connect()
  return () => { stopped = true; source?.close(); if (reconnect !== undefined) window.clearTimeout(reconnect) }
}
