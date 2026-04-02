import { useQuery } from '@tanstack/react-query'
import { Trophy, Medal, Award, TrendingUp, Users, Activity, Loader2 } from 'lucide-react'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts'
import { getLeaderboard, getAnalyticsSummary, getPointsHistory, PointsHistoryEntry, getBadgeDistribution } from '@/lib/api'
import { formatNumber } from '@/lib/utils'

interface LeaderboardUser {
  rank: number
  name: string
  points: number
  badges: number
  level: number
  avatar: string
}

export default function Analytics() {
  const { data: summaryData, isLoading: summaryLoading } = useQuery({
    queryKey: ['analytics-summary'],
    queryFn: getAnalyticsSummary,
  })

  const { data: leaderboardData, isLoading: leaderboardLoading } = useQuery({
    queryKey: ['leaderboard'],
    queryFn: () => getLeaderboard({ limit: 10 }),
  })

  const { data: pointsHistoryData, isLoading: pointsHistoryLoading } = useQuery({
    queryKey: ['points-history'],
    queryFn: () => getPointsHistory({ period: 'month' }),
  })

  const { data: badgeDistData, isLoading: badgeDistLoading } = useQuery({
    queryKey: ['badge-distribution'],
    queryFn: getBadgeDistribution,
  })

  const summary = summaryData?.data || {
    total_users: 0,
    total_badges: 0,
    badge_catalog_count: 0,
    active_users: 0,
    active_rules: 0,
    points_distributed: 0,
    events_processed: 0,
  }

  const stats = {
    totalUsers: summary.total_users,
    activeUsers: summary.active_users,
    totalPoints: summary.points_distributed,
    totalBadges: summary.total_badges,
    badgeCatalogCount: summary.badge_catalog_count,
  }

  const pointsHistory: PointsHistoryEntry[] = pointsHistoryData?.data || []

  const leaderboard = leaderboardData?.data || []

  const badgeDist = badgeDistData?.data || []
  const COLORS = ['#3b82f6', '#22c55e', '#a855f7', '#f59e0b', '#ef4444', '#06b6d4', '#8b5cf6', '#f97316']
  
  const badgeChartData = badgeDist.map((item, index) => ({
    name: item.badge_id.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase()),
    value: item.count,
    color: COLORS[index % COLORS.length]
  }))

  const displayUsers: LeaderboardUser[] = leaderboard.length > 0 ? leaderboard.map((entry: any, index: number) => ({
    rank: entry.rank || index + 1,
    name: entry.user?.name || entry.user_id || `User ${index + 1}`,
    points: entry.score || entry.points || 0,
    badges: entry.badges || 0,
    level: entry.user?.level || 1,
    avatar: (entry.user?.name || 'U').charAt(0).toUpperCase(),
  })) : []

  const getRankIcon = (rank: number) => {
    switch (rank) {
      case 1:
        return <Trophy className="h-5 w-5 text-yellow-500" />
      case 2:
        return <Medal className="h-5 w-5 text-gray-400" />
      case 3:
        return <Award className="h-5 w-5 text-amber-600" />
      default:
        return <span className="text-sm font-medium text-muted-foreground">{rank}</span>
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold text-foreground">Analytics & Leaderboard</h2>
        <p className="text-muted-foreground">View performance metrics and top performers</p>
      </div>

      {/* Stats Overview */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="p-6 rounded-xl bg-card border border-border">
          <div className="flex items-center justify-between">
            <div className="p-2 rounded-lg bg-blue-500/10">
              {summaryLoading ? (
                <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />
              ) : (
                <Users className="h-5 w-5 text-blue-500" />
              )}
            </div>
          </div>
          <div className="mt-4">
            <p className="text-2xl font-bold text-foreground">{formatNumber(stats.totalUsers)}</p>
            <p className="text-sm text-muted-foreground">Total Users</p>
          </div>
        </div>

        <div className="p-6 rounded-xl bg-card border border-border">
          <div className="flex items-center justify-between">
            <div className="p-2 rounded-lg bg-green-500/10">
              <Activity className="h-5 w-5 text-green-500" />
            </div>
          </div>
          <div className="mt-4">
            <p className="text-2xl font-bold text-foreground">{formatNumber(stats.activeUsers)}</p>
            <p className="text-sm text-muted-foreground">Active Users</p>
          </div>
        </div>

        <div className="p-6 rounded-xl bg-card border border-border">
          <div className="flex items-center justify-between">
            <div className="p-2 rounded-lg bg-purple-500/10">
              <TrendingUp className="h-5 w-5 text-purple-500" />
            </div>
          </div>
          <div className="mt-4">
            <p className="text-2xl font-bold text-foreground">{formatNumber(stats.totalPoints)}</p>
            <p className="text-sm text-muted-foreground">Total Points</p>
          </div>
        </div>

        <div className="p-6 rounded-xl bg-card border border-border">
          <div className="flex items-center justify-between">
            <div className="p-2 rounded-lg bg-yellow-500/10">
              <Award className="h-5 w-5 text-yellow-500" />
            </div>
          </div>
          <div className="mt-4">
            <p className="text-2xl font-bold text-foreground">{formatNumber(stats.totalBadges)}</p>
            <p className="text-sm text-muted-foreground">Total Badges Earned</p>
          </div>
        </div>

        <div className="p-6 rounded-xl bg-card border border-border">
          <div className="flex items-center justify-between">
            <div className="p-2 rounded-lg bg-orange-500/10">
              <Award className="h-5 w-5 text-orange-500" />
            </div>
          </div>
          <div className="mt-4">
            <p className="text-2xl font-bold text-foreground">{formatNumber(stats.badgeCatalogCount)}</p>
            <p className="text-sm text-muted-foreground">Badge Types</p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Leaderboard */}
        <div className="p-6 rounded-xl bg-card border border-border">
          <h3 className="text-lg font-semibold text-foreground mb-4 flex items-center gap-2">
            <Trophy className="h-5 w-5 text-yellow-500" />
            Top Players
          </h3>
          
          {leaderboardLoading ? (
            <div className="h-[400px] flex items-center justify-center">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : displayUsers.length === 0 ? (
            <div className="h-[200px] flex flex-col items-center justify-center text-muted-foreground">
              <Trophy className="h-8 w-8 mb-2 opacity-50" />
              <p className="text-sm">Henüz lider tablosu verisi yok</p>
              <p className="text-xs mt-1">Kullanıcılar puan kazandıkça burada görünecek</p>
            </div>
          ) : (
            <div className="space-y-3">
              {displayUsers.map((user) => (
                <div
                  key={user.rank}
                  className={`flex items-center justify-between p-3 rounded-lg ${
                    user.rank <= 3 ? 'bg-muted/50' : 'hover:bg-muted/30'
                  }`}
                >
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 flex items-center justify-center">
                      {getRankIcon(user.rank)}
                    </div>
                    <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
                      <span className="text-sm font-semibold text-primary">{user.avatar}</span>
                    </div>
                    <div>
                      <p className="font-medium text-foreground">{user.name}</p>
                      <p className="text-xs text-muted-foreground">Level {user.level}</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="font-semibold text-foreground">{formatNumber(user.points)}</p>
                    <p className="text-xs text-muted-foreground">{user.badges} badges</p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Charts */}
        <div className="space-y-6">
          {/* Activity Chart */}
          <div className="p-6 rounded-xl bg-card border border-border">
            <h3 className="text-lg font-semibold text-foreground mb-4">Points History</h3>
            {pointsHistoryLoading ? (
              <div className="h-[200px] flex items-center justify-center">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            ) : pointsHistory.length > 0 ? (
              <ResponsiveContainer width="100%" height={200}>
                <LineChart data={pointsHistory}>
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
                  <Line
                    type="monotone"
                    dataKey="points"
                    stroke="hsl(var(--primary))"
                    strokeWidth={2}
                    dot={{ fill: 'hsl(var(--primary))' }}
                  />
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <div className="h-[200px] flex items-center justify-center text-muted-foreground">
                No points history available
              </div>
            )}
          </div>

          {/* Badge Distribution */}
          <div className="p-6 rounded-xl bg-card border border-border">
            <h3 className="text-lg font-semibold text-foreground mb-4">Badge Rarity Distribution</h3>
            {badgeDistLoading ? (
              <div className="flex items-center justify-center h-[150px]">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            ) : badgeChartData.length > 0 ? (
              <div className="flex items-center gap-8">
                <ResponsiveContainer width={150} height={150}>
                  <PieChart>
                    <Pie
                      data={badgeChartData}
                      cx="50%"
                      cy="50%"
                      innerRadius={40}
                      outerRadius={60}
                      paddingAngle={5}
                      dataKey="value"
                    >
                      {badgeChartData.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={entry.color} />
                      ))}
                    </Pie>
                    <Tooltip
                      contentStyle={{
                        backgroundColor: 'hsl(var(--card))',
                        border: '1px solid hsl(var(--border))',
                        borderRadius: '8px',
                      }}
                    />
                  </PieChart>
                </ResponsiveContainer>
                <div className="space-y-2 max-h-[150px] overflow-y-auto w-full pr-2">
                  {badgeChartData.map((item) => (
                    <div key={item.name} className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-2">
                        <div className="w-3 h-3 rounded-full shrink-0" style={{ backgroundColor: item.color }} />
                        <span className="text-sm text-foreground truncate max-w-[120px]" title={item.name}>{item.name}</span>
                      </div>
                      <span className="text-sm font-medium text-muted-foreground">{formatNumber(item.value)} users</span>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
               <div className="h-[150px] flex flex-col items-center justify-center text-muted-foreground">
                 <Award className="h-8 w-8 mb-2 opacity-50" />
                 <p className="text-sm">No badges distributed yet</p>
               </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}