import axios from 'axios'
import { API_BASE_URL } from './constants'
import type { StatusResponse, SubscriberInfo, TopicInfo, RecentEventsResponse, TopicSchema, ServiceSchema } from '@/types/api'

const api = axios.create({ baseURL: API_BASE_URL })

export async function fetchStatus(): Promise<StatusResponse> {
  const { data } = await api.get('/api/status')
  return data
}

export async function fetchSubscribers(): Promise<SubscriberInfo[]> {
  const { data } = await api.get('/api/subscribers')
  return data || []
}

export async function fetchTopics(): Promise<TopicInfo[]> {
  const { data } = await api.get('/api/topics')
  return data || []
}

export async function fetchRecentEvents(n = 100): Promise<RecentEventsResponse> {
  const { data } = await api.get(`/api/events/recent?n=${n}`)
  return data
}

export async function fetchTopicSchemas(): Promise<TopicSchema[]> {
  const { data } = await api.get('/api/topics/schemas')
  return data || []
}

export async function fetchEventDetail(correlationId: string): Promise<RecentEventsResponse> {
  const { data } = await api.get(`/api/events/detail/${correlationId}`)
  return data
}

export async function fetchServiceSchemas(): Promise<ServiceSchema[]> {
  const { data } = await api.get('/api/services/schemas')
  return data || []
}
