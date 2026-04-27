export type StatusResponse = {
  status: string
  uptime: string
  uptime_seconds: number
  total_subscribers: number
  active_topics: number
  in_flight_tasks: number
  total_events_published: number
  total_responses: number
  total_timeouts: number
  events_per_second: number
  timeout_rate: number
  ws_clients: number
}

export type SubscriberInfo = {
  id: string
  topic: string
  is_active: boolean
  connected_at: string
}

export type TopicInfo = {
  name: string
  subscriber_count: number
}

export type EventRecord = {
  timestamp: string
  type: string
  topic: string
  correlation_id: string
  subscriber_id: string
  latency_ms: number
  strategy: string
  status: string
  subscriber_count?: number
  response_count?: number
  inputs?: Record<string, string>
  outputs?: Record<string, string>
  merged_outputs?: Record<string, string>
}

export type RecentEventsResponse = {
  events: EventRecord[]
}

export type FieldSchema = {
  name: string
  type: string
  required: boolean
  description: string
  description_en: string
}

export type TopicSchema = {
  name: string
  pipeline: string
  stage: string
  description: string
  description_en: string
  inputs: FieldSchema[]
  outputs: FieldSchema[]
  recommended_strategy: string
  recommended_weight: number
}

export type MethodSchema = {
  name: string
  input_type: string
  output_type: string
  description: string
  description_en: string
}

export type MessageSchema = {
  name: string
  description: string
  fields: FieldSchema[]
}

export type ServiceSchema = {
  name: string
  package: string
  description: string
  description_en: string
  methods: MethodSchema[]
  messages: MessageSchema[]
}
