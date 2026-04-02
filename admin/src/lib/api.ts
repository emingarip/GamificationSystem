import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1'
const SETTINGS_KEY = 'gamification-admin-settings'

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('refreshToken')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

const nowIso = () => new Date().toISOString()

const parseJSON = <T>(value: unknown, fallback: T): T => {
  if (value == null) return fallback
  if (typeof value !== 'string') return value as T
  try {
    return JSON.parse(value) as T
  } catch {
    return fallback
  }
}

const normalizeTrigger = (trigger?: string) => {
  const normalized = (trigger || '').trim().toLowerCase()
  if (!normalized) return 'goal'
  return normalized.replace(/\./g, '_')
}

const inferRuleType = (rule: any): Rule['type'] => {
  const actions = rule.actions || []
  if (Array.isArray(actions)) {
    if (actions.some((a: any) => a.action_type === 'grant_badge')) return 'badge'
    if (actions.some((a: any) => a.action_type === 'award_points')) return 'points'
  }
  // Fallback for legacy
  const rewards = typeof rule.rewards === 'string' ? parseJSON<any>(rule.rewards, {}) : (rule.rewards || {})
  if (rewards.badge_id) return 'badge'
  if (rewards.points || rule.points > 0) return 'points'
  return 'streak'
}

const toBadgeSummary = (badge: any): Badge => {
  const id = badge.id || badge.badge_id || `badge-${Math.random().toString(36).slice(2, 8)}`
  const rarity = (badge.rarity || badge.category || 'common') as Badge['rarity']
  const colorMap: Record<Badge['rarity'], string> = {
    common: '#6b7280',
    rare: '#3b82f6',
    epic: '#8b5cf6',
    legendary: '#f59e0b',
  }

  return {
    id,
    name: badge.name || id,
    description: badge.description || '',
    icon: badge.icon || 'award',
    color: badge.color || colorMap[rarity],
    points: badge.points || 0,
    rarity,
    requirements: badge.requirements || badge.criteria || badge.criteria || '{}',
    createdAt: badge.createdAt || badge.created_at || nowIso(),
    updatedAt: badge.updatedAt || badge.updated_at || badge.createdAt || badge.created_at || nowIso(),
  }
}

// RichBadgeInfo from backend
interface RichBadgeInfo {
  id: string
  name: string
  description: string
  icon: string
  points: number
  earned_at: string
  reason: string
}

const toUser = (user: any): User => {
  // Use rich_badge_info directly from backend
  const richBadges: RichBadgeInfo[] = user.rich_badge_info || []
  
  const badges = richBadges.map((badge) =>
    toBadgeSummary({
      id: badge.id,
      name: badge.name,
      description: badge.description,
      icon: badge.icon,
      points: badge.points,
      rarity: 'common',
      createdAt: badge.earned_at,
    })
  )

  return {
    id: user.id,
    email: user.email || '',
    name: user.name || user.username || user.email?.split('@')[0] || 'User',
    avatar: user.avatar,
    points: user.points || 0,
    level: user.level || 1,
    badges,
    createdAt: user.createdAt || user.created_at || nowIso(),
    updatedAt: user.updatedAt || user.updated_at || user.createdAt || user.created_at || nowIso(),
    rich_badge_info: richBadges,
    recent_activity: user.recent_activity || [],
  }
}

const toRule = (rule: any): Rule => ({
  id: rule.id || rule.rule_id || `rule-${Math.random().toString(36).slice(2, 8)}`,
  name: rule.name || 'Untitled Rule',
  description: rule.description || '',
  type: inferRuleType(rule),
  trigger: rule.trigger || rule.event_type || 'goal',
  conditions: typeof rule.conditions === 'string' ? rule.conditions : JSON.stringify(rule.conditions || [], null, 2),
  actions: (() => {
    let actions = rule.actions || []
    if (actions.length > 0) return JSON.stringify(actions, null, 2)
    
    // Fallback to legacy rewards
    actions = []
    const rewards = typeof rule.rewards === 'string' ? parseJSON<any>(rule.rewards, {}) : (rule.rewards || {})
    if (rewards.points || rule.points > 0) {
      actions.push({ action_type: 'award_points', params: { points: rewards.points || rule.points, reason: rewards.reason || rule.name } })
    }
    if (rewards.badge_id) {
      actions.push({ action_type: 'grant_badge', params: { badge_id: rewards.badge_id } })
    }
    return JSON.stringify(actions, null, 2)
  })(),
  enabled: rule.enabled ?? rule.is_active ?? true,
  priority: rule.priority || 1,
  createdAt: rule.createdAt || rule.created_at || nowIso(),
  updatedAt: rule.updatedAt || rule.updated_at || rule.createdAt || rule.created_at || nowIso(),
})

const ruleToRequest = (data: Partial<Rule>) => {
  const actions = parseJSON<unknown[]>(data.actions, [])
  const conditions = parseJSON<unknown[]>(data.conditions, [])

  return {
    id: data.id,
    name: data.name,
    description: data.description,
    event_type: normalizeTrigger(data.trigger),
    points: 0,
    enabled: data.enabled ?? true,
    conditions: Array.isArray(conditions) ? conditions : [],
    actions: Array.isArray(actions) ? actions : [],
    cooldown: 0,
  }
}

const loadSavedSettings = () => {
  const defaults = {
    appName: 'Gamification Platform',
    apiUrl: 'http://gamification.boskale.com/api/v1',
    wsUrl: 'ws://gamification.boskale.com/ws',
    language: 'en',
    timezone: 'UTC',
    allowRegistration: true,
    requireEmailVerification: false,
    maxPointsPerDay: 1000,
    pointsExpirationDays: 365,
    enableNotifications: true,
    enableEmailNotifications: true,
    enablePushNotifications: false,
    logLevel: 'info',
    corsOrigins: '*',
  }

  const saved = localStorage.getItem(SETTINGS_KEY)
  if (!saved) return defaults
  return { ...defaults, ...parseJSON(saved, {}) }
}

// Auth
export interface LoginResponse {
  token: string
  refreshToken: string
  user: User
}

export const login = async (credentials: { email: string; password: string }): Promise<LoginResponse> => {
  const response = await api.post('/auth/login', credentials)
  return {
    token: response.data.token,
    refreshToken: response.data.refreshToken,
    user: toUser(response.data.user),
  }
}

export const logout = async () => {
  try {
    await api.post('/auth/logout')
  } catch {
    // Ignore logout failures in local development.
  }
}

export const getCurrentUser = async () => {
  const response = await api.get('/auth/me')
  return { data: toUser(response.data) }
}

// Users
export const getUsers = async (_params?: { page?: number; limit?: number; search?: string }) => {
  const response = await api.get('/users')
  return { data: (response.data.users || []).map(toUser) }
}

export const getUser = async (id: string) => {
  const response = await api.get(`/users/${id}`)
  return { data: toUser(response.data) }
}

export const updateUser = async (id: string, data: Partial<User>) => {
  const response = await api.put(`/users/${id}`, data)
  return response.data
}

export const deleteUser = async (id: string) => {
  const response = await api.delete(`/users/${id}`)
  return response.data
}

// Rules
export const getRules = async (_params?: { page?: number; limit?: number; type?: string }) => {
  const response = await api.get('/rules')
  return { data: (response.data.rules || []).map(toRule) }
}

export const getRule = async (id: string) => {
  const response = await api.get(`/rules/${id}`)
  return { data: toRule(response.data) }
}

export const createRule = async (data: Omit<Rule, 'id' | 'createdAt' | 'updatedAt'>) => {
  const response = await api.post('/rules', ruleToRequest(data))
  return response.data
}

export const updateRule = async (id: string, data: Partial<Rule>) => {
  const response = await api.put(`/rules/${id}`, ruleToRequest(data))
  return response.data
}

export const deleteRule = async (id: string) => {
  const response = await api.delete(`/rules/${id}`)
  return response.data
}

export const generateRule = async (prompt: string) => {
  const lowered = prompt.toLowerCase()
  const badgeLike = lowered.includes('badge') || lowered.includes('rozet')

  const generated: Rule = {
    id: `rule-${Date.now()}`,
    name: prompt.slice(0, 48) || 'Generated Rule',
    description: prompt,
    type: badgeLike ? 'badge' : 'points',
    trigger: 'goal',
    conditions: JSON.stringify([], null, 2),
    actions: JSON.stringify(badgeLike ? [{ action_type: 'grant_badge', params: { badge_id: 'generated_badge' } }] : [{ action_type: 'award_points', params: { points: 100 } }], null, 2),
    enabled: true,
    priority: 1,
    createdAt: nowIso(),
    updatedAt: nowIso(),
  }

  return { rule: generated }
}

// Event Types
// Dynamic event type support - event types come from Redis registry
export interface EventTypeInfo {
  key: string
  name: string
  description: string
  category: string
  enabled: boolean
  created_at?: string
}

export const getEventTypes = async (): Promise<EventTypeInfo[]> => {
  const response = await api.get('/event-types')
  return response.data.event_types || []
}

export const createEventType = async (data: { key: string; name: string; description?: string; category?: string; enabled?: boolean }) => {
  const response = await api.post('/event-types', data)
  return response.data
}

export const updateEventType = async (key: string, data: { name?: string; description?: string; category?: string; enabled?: boolean }) => {
  const response = await api.put(`/event-types/${key}`, data)
  return response.data
}

export const deleteEventType = async (key: string) => {
  const response = await api.delete(`/event-types/${key}`)
  return response.data
}

// Test Event
// Generic event types for non-sport events like daily_login, app_shared, purchase_completed
export interface TestEventRequest {
  event: {
    event_id: string
    event_type: string
    match_id?: string
    team_id?: string
    player_id?: string
    minute?: number
    timestamp: string
    metadata?: Record<string, unknown>
    // Generic fields for non-sport events
    subject_id?: string
    actor_id?: string
    source?: string
    context?: Record<string, unknown>
  }
  dry_run?: boolean
}

export interface TestEventResponse {
  matches: Array<{ rule_id: string; name: string; matched: boolean }>
  affected_users: string[]
  actions: Array<{ action_type: string; params: Record<string, unknown> }>
  executed: boolean
}

export const testEvent = async (data: TestEventRequest): Promise<{ data: TestEventResponse }> => {
  const response = await api.post('/events/test', data)
  return response.data
}

// Badges
export const getBadges = async (_params?: { page?: number; limit?: number }) => {
  const response = await api.get('/badges')
  return { data: (response.data.badges || []).map(toBadgeSummary) }
}

export const getBadge = async (id: string) => {
  const response = await api.get(`/badges/${id}`)
  return { data: toBadgeSummary(response.data) }
}

export const createBadge = async (data: Omit<Badge, 'id' | 'createdAt' | 'updatedAt'>) => {
  const response = await api.post('/badges', {
    name: data.name,
    description: data.description,
    icon: data.icon,
    points: data.points,
    criteria: data.requirements,
    rarity: data.rarity,
  })
  return response.data
}

export const updateBadge = async (id: string, data: Partial<Badge>) => {
  const response = await api.put(`/badges/${id}`, data)
  return response.data
}

export const deleteBadge = async (id: string) => {
  const response = await api.delete(`/badges/${id}`)
  return response.data
}

// Analytics - Real Endpoints
export interface AnalyticsSummary {
  total_users: number
  total_badges: number
  badge_catalog_count: number
  active_users: number
  active_rules: number
  points_distributed: number
  events_processed: number
}

export interface ActivityEntry {
  user_id: string
  action_type: string
  points: number
  reason: string
  timestamp: string
}

export interface PointsHistoryEntry {
  date: string
  points: number
}

export const getAnalyticsSummary = async (): Promise<{ data: AnalyticsSummary }> => {
  const response = await api.get('/analytics/summary')
  return { data: response.data }
}

export const getAnalyticsActivity = async (params?: { limit?: number }): Promise<{ data: ActivityEntry[] }> => {
  const response = await api.get('/analytics/activity', { params: { limit: params?.limit || 50 } })
  return { data: response.data.activities || [] }
}

export const getPointsHistory = async (params?: { period?: string }): Promise<{ data: PointsHistoryEntry[] }> => {
  const response = await api.get('/analytics/points-history', { params: { period: params?.period || 'month' } })
  return { data: response.data.history || [] }
}

export interface BadgeDistributionEntry {
  badge_id: string
  count: number
}

export const getBadgeDistribution = async (): Promise<{ data: BadgeDistributionEntry[] }> => {
  const response = await api.get('/analytics/badge-distribution')
  return { data: response.data.distribution || [] }
}

export interface EventDebugLog {
  timestamp?: string
  event: TestEventRequest['event']
  triggered_rules: any[]
  total_time_ms: number
  success: boolean
  error?: string
  skipped: boolean
  skip_reason?: string
}

export const getEventLogs = async (): Promise<{ data: EventDebugLog[] }> => {
  const response = await api.get('/analytics/event-logs')
  return { data: response.data.logs || [] }
}

// Legacy Stats (for backward compatibility)
export const getStats = async (): Promise<Stats> => {
  const summaryResponse = await getAnalyticsSummary()
  const summary = summaryResponse.data
  
  return {
    totalUsers: summary.total_users,
    activeUsers: summary.active_users,
    totalPoints: summary.points_distributed,
    totalBadges: summary.total_badges,
    badgeCatalogCount: summary.badge_catalog_count,
    totalRules: summary.active_rules,
    recentActivity: [],
  }
}

export const getLeaderboard = async (params?: { limit?: number; period?: string }) => {
  const response = await api.get('/leaderboard', { params: { limit: params?.limit } })
  return { data: response.data.entries || [] }
}

export const getActivityHistory = async (_params?: { limit?: number; userId?: string }) => {
  const response = await api.get('/analytics/activity', { params: { limit: _params?.limit || 20 } })
  const activities: ActivityEntry[] = response.data.activities || []
  
  return { 
    data: activities.map((activity, index) => ({
      id: `${activity.user_id}-activity-${index}`,
      userId: activity.user_id,
      userName: activity.user_id, // Will be resolved by the component
      type: activity.action_type,
      description: activity.reason,
      points: activity.points,
      timestamp: activity.timestamp,
    }))
  }
}

// Settings
export const getConfig = async () => {
  return { data: loadSavedSettings() }
}

export const updateConfig = async (data: Record<string, unknown>) => {
  const next = { ...loadSavedSettings(), ...data }
  localStorage.setItem(SETTINGS_KEY, JSON.stringify(next))
  return { data: next }
}

// Types
export interface User {
  id: string
  email: string
  name: string
  avatar?: string
  points: number
  level: number
  badges: Badge[]
  rich_badge_info?: RichBadgeInfo[]
  recent_activity?: any[]
  createdAt: string
  updatedAt: string
}

export interface Rule {
  id: string
  name: string
  description: string
  type: 'points' | 'badge' | 'level' | 'streak'
  trigger: string
  conditions: string
  actions: string
  enabled: boolean
  priority: number
  createdAt: string
  updatedAt: string
}

export interface Badge {
  id: string
  name: string
  description: string
  icon: string
  color: string
  points: number
  rarity: 'common' | 'rare' | 'epic' | 'legendary'
  requirements: string
  createdAt: string
  updatedAt: string
}

export interface Stats {
  totalUsers: number
  activeUsers: number
  totalPoints: number
  totalBadges: number
  badgeCatalogCount: number
  totalRules: number
  recentActivity: Activity[]
}

export interface Activity {
  id: string
  userId: string
  userName: string
  type: string
  description: string
  points: number
  timestamp: string
}

export interface LeaderboardEntry {
  rank: number
  user: User
  points: number
  badges: number
}
