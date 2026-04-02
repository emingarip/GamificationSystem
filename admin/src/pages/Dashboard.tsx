import { useQuery } from '@tanstack/react-query'
import { Users, Award, Activity, TrendingUp, Loader2, AlertCircle, RefreshCw } from 'lucide-react'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import { getStats, getPointsHistory, getActivityHistory } from '@/lib/api'
import { formatNumber } from '@/lib/utils'

export default function Dashboard() {
  const { data: stats, isLoading: statsLoading, error: statsError, refetch: refetchStats } = useQuery({
    queryKey: ['stats'],
    queryFn: getStats,
  })

  const { data: pointsData, isLoading: pointsLoading, error: pointsError, refetch: refetchPoints } = useQuery({
    queryKey: ['pointsHistory'],
    queryFn: () => getPointsHistory({ period: 'month' }),
  })

  const { data: activityData, isLoading: activityLoading, error: activityError, refetch: refetchActivity } = useQuery({
    queryKey: ['activityHistory'],
    queryFn: () => getActivityHistory({ limit: 10 }),
  })

  // Show error state if stats failed
  if (statsError) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px] space-y-4">
        <AlertCircle className="h-12 w-12 text-destructive" />
        <div className="text-center">
          <h3 className="text-lg font-semibold text-foreground">Panel verileri yüklenemedi</h3>
          <p className="text-sm text-muted-foreground mt-1">{statsError.message || 'Sunucuya bağlanılamıyor'}</p>
        </div>
        <button
          onClick={() => refetchStats()}
          className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
        >
          <RefreshCw className="h-4 w-4" />
          Yeniden Dene
        </button>
      </div>
    )
  }

  // Show loading state
  if (statsLoading) {
    return (
      <div className="space-y-6">
        {/* Page Header */}
        <div>
          <h2 className="text-2xl font-bold text-foreground">Genel Bakış</h2>
          <p className="text-muted-foreground">Oyunlaştırma platformunuzdaki son durum</p>
        </div>

        {/* Stats Cards Skeleton */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="p-6 rounded-xl bg-card border border-border animate-pulse">
              <div className="flex items-center justify-between">
                <div className="h-10 w-10 rounded-lg bg-muted" />
                <div className="h-4 w-12 rounded bg-muted" />
              </div>
              <div className="mt-4 space-y-2">
                <div className="h-8 w-24 rounded bg-muted" />
                <div className="h-4 w-32 rounded bg-muted" />
              </div>
            </div>
          ))}
        </div>

        {/* Charts Skeleton */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="p-6 rounded-xl bg-card border border-border animate-pulse">
            <div className="h-6 w-40 rounded bg-muted mb-4" />
            <div className="h-[300px] bg-muted rounded" />
          </div>
          <div className="p-6 rounded-xl bg-card border border-border animate-pulse">
            <div className="h-6 w-40 rounded bg-muted mb-4" />
            <div className="h-[300px] bg-muted rounded" />
          </div>
        </div>

        {/* Recent Activity Skeleton */}
        <div className="p-6 rounded-xl bg-card border border-border animate-pulse">
          <div className="h-6 w-40 rounded bg-muted mb-4" />
          <div className="space-y-4">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="flex items-center justify-between p-3 rounded-lg">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-muted" />
                  <div className="space-y-1">
                    <div className="h-4 w-48 rounded bg-muted" />
                    <div className="h-3 w-20 rounded bg-muted" />
                  </div>
                </div>
                <div className="h-4 w-16 rounded bg-muted" />
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  const statCards = [
    {
      title: 'Toplam Kullanıcı',
      value: formatNumber(stats?.totalUsers ?? 0),
      icon: Users,
      color: 'text-blue-500',
      bgColor: 'bg-blue-500/10',
    },
    {
      title: 'Aktif Kullanıcı',
      value: formatNumber(stats?.activeUsers ?? 0),
      icon: Activity,
      color: 'text-green-500',
      bgColor: 'bg-green-500/10',
    },
    {
      title: 'Toplam Puan',
      value: formatNumber(stats?.totalPoints ?? 0),
      icon: TrendingUp,
      color: 'text-purple-500',
      bgColor: 'bg-purple-500/10',
    },
    {
      title: 'Kazanılan Rozet',
      value: formatNumber(stats?.totalBadges ?? 0),
      icon: Award,
      color: 'text-yellow-500',
      bgColor: 'bg-yellow-500/10',
    },
    {
      title: 'Rozet Türleri',
      value: formatNumber(stats?.badgeCatalogCount ?? 0),
      icon: Award,
      color: 'text-orange-500',
      bgColor: 'bg-orange-500/10',
    },
  ]

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h2 className="text-2xl font-bold text-foreground">Genel Bakış</h2>
        <p className="text-muted-foreground">Oyunlaştırma platformunuzdaki son durum</p>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {statCards.map((stat, index) => (
          <div
            key={index}
            className="p-6 rounded-xl bg-card border border-border hover:border-primary/50 transition-colors"
          >
            <div className="flex items-center justify-between">
              <div className={`p-2 rounded-lg ${stat.bgColor}`}>
                <stat.icon className={`h-5 w-5 ${stat.color}`} />
              </div>
            </div>
            <div className="mt-4">
              <p className="text-2xl font-bold text-foreground">{stat.value}</p>
              <p className="text-sm text-muted-foreground">{stat.title}</p>
            </div>
          </div>
        ))}
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Points Over Time */}
        <div className="p-6 rounded-xl bg-card border border-border">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-foreground">Points Over Time</h3>
            {pointsError && (
              <button
                onClick={() => refetchPoints()}
                className="text-sm text-destructive hover:underline flex items-center gap-1"
              >
                <RefreshCw className="h-3 w-3" /> Retry
              </button>
            )}
          </div>
          {pointsLoading ? (
            <div className="h-[300px] flex items-center justify-center">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : pointsError ? (
            <div className="h-[300px] flex items-center justify-center text-destructive">
              <AlertCircle className="h-5 w-5 mr-2" />
              Failed to load points data
            </div>
          ) : pointsData?.data ? (
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={pointsData.data}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                <XAxis dataKey="date" className="text-xs fill-muted-foreground" />
                <YAxis className="text-xs fill-muted-foreground" />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'hsl(var(--card))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '8px',
                  }}
                />
                <Area
                  type="monotone"
                  dataKey="points"
                  stroke="hsl(var(--primary))"
                  fill="hsl(var(--primary) / 0.2)"
                  strokeWidth={2}
                />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-[300px] flex items-center justify-center text-muted-foreground">
              No data available
            </div>
          )}
        </div>

        {/* Users by Level - Placeholder for future API */}
        <div className="p-6 rounded-xl bg-card border border-border">
          <h3 className="text-lg font-semibold text-foreground mb-4">Users by Level</h3>
          <div className="h-[300px] flex items-center justify-center text-muted-foreground">
            <div className="text-center">
              <Activity className="h-8 w-8 mx-auto mb-2 opacity-50" />
              <p className="text-sm">No data available</p>
            </div>
          </div>
        </div>
      </div>

      {/* Recent Activity */}
      <div className="p-6 rounded-xl bg-card border border-border">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-foreground">Recent Activity</h3>
          {activityError && (
            <button
              onClick={() => refetchActivity()}
              className="text-sm text-destructive hover:underline flex items-center gap-1"
            >
              <RefreshCw className="h-3 w-3" /> Retry
            </button>
          )}
        </div>
        {activityLoading ? (
          <div className="space-y-4">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="flex items-center justify-between p-3 rounded-lg animate-pulse">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-muted" />
                  <div className="space-y-1">
                    <div className="h-4 w-48 rounded bg-muted" />
                    <div className="h-3 w-20 rounded bg-muted" />
                  </div>
                </div>
                <div className="h-4 w-16 rounded bg-muted" />
              </div>
            ))}
          </div>
        ) : activityError ? (
          <div className="flex items-center justify-center py-8 text-destructive">
            <AlertCircle className="h-5 w-5 mr-2" />
            Failed to load recent activity
          </div>
        ) : activityData?.data && activityData.data.length > 0 ? (
          <div className="space-y-4">
            {activityData.data.map((activity: any) => (
              <div
                key={activity.id}
                className="flex items-center justify-between p-3 rounded-lg bg-muted/50"
              >
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
                    <Activity className="h-5 w-5 text-primary" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-foreground">
                      {activity.userName || activity.userId} - {activity.description}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {new Date(activity.timestamp).toLocaleString()}
                    </p>
                  </div>
                </div>
                <span className="text-sm font-semibold text-green-500">+{activity.points} pts</span>
              </div>
            ))}
          </div>
        ) : (
          <div className="flex items-center justify-center py-8 text-muted-foreground">
            No recent activity
          </div>
        )}
      </div>
    </div>
  )
}