import { create } from 'zustand'
import { fetchStatus, fetchSubscribers, fetchTopics, fetchRecentEvents, fetchTopicSchemas, fetchServiceSchemas } from '@/lib/api'
import { EventBusWebSocket } from '@/lib/websocket'
import { WS_URL, POLL_INTERVAL } from '@/lib/constants'
import type { StatusResponse, SubscriberInfo, TopicInfo, EventRecord, TopicSchema, ServiceSchema } from '@/types/api'

interface DashboardState {
  status: StatusResponse | null
  subscribers: SubscriberInfo[]
  topics: TopicInfo[]
  recentEvents: EventRecord[]
  liveEvents: EventRecord[]
  topicSchemas: TopicSchema[]
  serviceSchemas: ServiceSchema[]
  wsConnected: boolean

  // actions
  fetchStatus: () => Promise<void>
  fetchSubscribers: () => Promise<void>
  fetchTopics: () => Promise<void>
  fetchRecentEvents: () => Promise<void>
  fetchTopicSchemas: () => Promise<void>
  fetchServiceSchemas: () => Promise<void>
  addLiveEvent: (event: EventRecord) => void
  setWsConnected: (connected: boolean) => void
  startPolling: () => void
  stopPolling: () => void
  connectWS: () => void
  disconnectWS: () => void
}

let pollingTimer: ReturnType<typeof setInterval> | null = null
let wsClient: EventBusWebSocket | null = null

export const useDashboardStore = create<DashboardState>((set, get) => ({
  status: null,
  subscribers: [],
  topics: [],
  recentEvents: [],
  liveEvents: [],
  topicSchemas: [],
  serviceSchemas: [],
  wsConnected: false,

  fetchStatus: async () => {
    try {
      const status = await fetchStatus()
      set({ status })
    } catch {
      // ignore fetch errors
    }
  },

  fetchSubscribers: async () => {
    try {
      const subscribers = await fetchSubscribers()
      set({ subscribers })
    } catch {
      set({ subscribers: [] })
    }
  },

  fetchTopics: async () => {
    try {
      const topics = await fetchTopics()
      set({ topics })
    } catch {
      set({ topics: [] })
    }
  },

  fetchRecentEvents: async () => {
    try {
      const data = await fetchRecentEvents(200)
      set({ recentEvents: data.events || [] })
    } catch {
      set({ recentEvents: [] })
    }
  },

  fetchTopicSchemas: async () => {
    try {
      const schemas = await fetchTopicSchemas()
      set({ topicSchemas: schemas })
    } catch {
      set({ topicSchemas: [] })
    }
  },

  fetchServiceSchemas: async () => {
    try {
      const schemas = await fetchServiceSchemas()
      set({ serviceSchemas: schemas })
    } catch {
      set({ serviceSchemas: [] })
    }
  },

  addLiveEvent: (event) => {
    set((state) => {
      const liveEvents = [event, ...state.liveEvents].slice(0, 500)
      return { liveEvents }
    })
  },

  setWsConnected: (wsConnected) => set({ wsConnected }),

  startPolling: () => {
    const { fetchStatus, fetchSubscribers, fetchTopics, fetchRecentEvents, fetchTopicSchemas, fetchServiceSchemas } = get()
    // Immediate fetch
    fetchStatus()
    fetchSubscribers()
    fetchTopics()
    fetchRecentEvents()
    fetchTopicSchemas()
    fetchServiceSchemas()
    // Periodic polling
    if (pollingTimer) clearInterval(pollingTimer)
    pollingTimer = setInterval(() => {
      fetchStatus()
      fetchSubscribers()
      fetchTopics()
    }, POLL_INTERVAL)
  },

  stopPolling: () => {
    if (pollingTimer) {
      clearInterval(pollingTimer)
      pollingTimer = null
    }
  },

  connectWS: () => {
    if (wsClient) wsClient.disconnect()
    wsClient = new EventBusWebSocket(WS_URL, (event) => {
      get().addLiveEvent(event)
    })
    wsClient.onConnectionChange = (connected) => {
      get().setWsConnected(connected)
    }
    wsClient.connect()
  },

  disconnectWS: () => {
    if (wsClient) {
      wsClient.disconnect()
      wsClient = null
    }
  },
}))
