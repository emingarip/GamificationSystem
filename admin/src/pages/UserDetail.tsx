import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft,
  Award,
  Star,
  Loader2,
  Calendar,
  Zap,
  Clock,
  Shield,
  Edit,
} from 'lucide-react'
import { getUser, updateUser, User } from '@/lib/api'
import { formatDate } from '@/lib/utils'
import { toast } from 'sonner'

export default function UserDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [isEditing, setIsEditing] = useState(false)
  const [name, setName] = useState('')
  const [points, setPoints] = useState(0)
  const [level, setLevel] = useState(1)

  const { data, isLoading, isError } = useQuery({
    queryKey: ['user', id],
    queryFn: () => getUser(id as string),
    enabled: !!id,
  })

  const updateMutation = useMutation({
    mutationFn: ({ userId, data }: { userId: string; data: Partial<User> }) =>
      updateUser(userId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user', id] })
      toast.success('Kullanıcı başarıyla güncellendi')
      setIsEditing(false)
    },
    onError: () => toast.error('Güncelleme başarısız oldu'),
  })

  // Set initial edit state when data loads
  const handleEditClick = (user: User) => {
    setName(user.name)
    setPoints(user.points)
    setLevel(user.level)
    setIsEditing(true)
  }

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault()
    if (id) {
      updateMutation.mutate({ userId: id, data: { name, points, level } })
    }
  }

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (isError || !data?.data) {
    return (
      <div className="text-center py-12">
        <p className="text-destructive mb-4">Kullanıcı bulunamadı veya bir hata oluştu.</p>
        <button
          onClick={() => navigate('/users')}
          className="px-4 py-2 bg-primary text-primary-foreground rounded-lg"
        >
          Geri Dön
        </button>
      </div>
    )
  }

  const user = data.data
  const richBadges = user.rich_badge_info || []
  const recentActivity = user.recent_activity || []

  return (
    <div className="space-y-6 max-w-6xl mx-auto pb-10">
      <div className="flex items-center justify-between border-b border-border pb-4">
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate('/users')}
            className="p-2 hover:bg-muted text-muted-foreground hover:text-foreground rounded-full transition-colors"
          >
            <ArrowLeft className="h-5 w-5" />
          </button>
          <div>
            <h2 className="text-2xl font-bold text-foreground">Kullanıcı Profili</h2>
            <p className="text-muted-foreground text-sm">Detaylı kullanıcı bilgileri ve geçmişi</p>
          </div>
        </div>
        <button
          onClick={() => handleEditClick(user)}
          className="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg border border-border bg-card hover:bg-muted transition-colors text-foreground"
        >
          <Edit className="h-4 w-4" /> Düzenle
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Left Column - User Info */}
        <div className="col-span-1 space-y-6">
          <div className="bg-card border border-border rounded-xl p-6 text-center shadow-sm">
            <div className="w-24 h-24 rounded-full bg-primary/10 text-primary flex items-center justify-center text-3xl font-bold mx-auto mb-4">
              {user.name.charAt(0).toUpperCase()}
            </div>
            <h3 className="text-xl font-bold text-foreground mb-1">{user.name}</h3>
            <p className="text-sm text-muted-foreground break-all mb-4">{user.email}</p>

            <div className="grid grid-cols-2 gap-4 mt-6">
              <div className="bg-muted rounded-lg p-3 text-center">
                <span className="block text-xl font-bold text-green-500 mb-1">
                  {user.points.toLocaleString()}
                </span>
                <span className="text-xs text-muted-foreground uppercase font-semibold">Puan</span>
              </div>
              <div className="bg-muted rounded-lg p-3 text-center">
                <span className="block text-xl font-bold text-yellow-500 mb-1">
                  Lvl {user.level}
                </span>
                <span className="text-xs text-muted-foreground uppercase font-semibold">
                  Seviye
                </span>
              </div>
            </div>

            <div className="mt-6 space-y-3 pt-6 border-t border-border text-sm">
              <div className="flex items-center justify-between text-muted-foreground">
                <div className="flex items-center gap-2">
                  <Calendar className="h-4 w-4" />
                  <span>Kayıt Tarihi</span>
                </div>
                <span className="text-foreground">{formatDate(user.createdAt)}</span>
              </div>
              <div className="flex items-center justify-between text-muted-foreground">
                <div className="flex items-center gap-2">
                  <Clock className="h-4 w-4" />
                  <span>Son Güncelleme</span>
                </div>
                <span className="text-foreground">{formatDate(user.updatedAt)}</span>
              </div>
              <div className="flex items-center justify-between text-muted-foreground">
                <div className="flex items-center gap-2">
                  <Shield className="h-4 w-4" />
                  <span>Rol</span>
                </div>
                <span className="text-foreground capitalize">Kullanıcı</span>
              </div>
            </div>
          </div>
        </div>

        {/* Right Column - Badges & Activity */}
        <div className="col-span-1 md:col-span-2 space-y-6">
          {/* Badges Section */}
          <div className="bg-card border border-border rounded-xl shadow-sm overflow-hidden">
            <div className="p-5 border-b border-border bg-muted/20 flex items-center gap-2">
              <Award className="h-5 w-5 text-yellow-500" />
              <h3 className="font-semibold text-foreground">Kazanılan Rozetler ({richBadges.length})</h3>
            </div>
            <div className="p-5">
              {richBadges.length > 0 ? (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  {richBadges.map((badge: any) => (
                    <div
                      key={badge.id}
                      className="flex items-center gap-4 bg-background border border-border p-3 rounded-xl hover:border-primary/50 transition-colors"
                    >
                      <div className="w-12 h-12 rounded-full bg-yellow-500/10 flex items-center justify-center shrink-0">
                        <Award className="h-6 w-6 text-yellow-500" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <h4 className="font-semibold text-foreground text-sm truncate">
                          {badge.name}
                        </h4>
                        <p className="text-xs text-muted-foreground truncate" title={badge.description}>
                          {badge.description}
                        </p>
                        <div className="flex items-center justify-between mt-1.5">
                          <span className="text-[10px] font-medium text-green-500 bg-green-500/10 px-2 py-0.5 rounded-full">
                            +{badge.points} Puan
                          </span>
                          <span className="text-[10px] text-muted-foreground">
                            {formatDate(badge.earned_at)}
                          </span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  <Award className="h-10 w-10 mx-auto text-muted mb-2" />
                  <p>Henüz rozet kazanılmadı</p>
                </div>
              )}
            </div>
          </div>

          {/* Activity Feed Section */}
          <div className="bg-card border border-border rounded-xl shadow-sm overflow-hidden">
            <div className="p-5 border-b border-border bg-muted/20 flex items-center gap-2">
              <Zap className="h-5 w-5 text-blue-500" />
              <h3 className="font-semibold text-foreground">Aktivite Geçmişi</h3>
            </div>
            <div className="p-0">
              {recentActivity.length > 0 ? (
                <div className="divide-y divide-border">
                  {recentActivity.map((activity: any, index: number) => (
                    <div key={index} className="p-4 hover:bg-muted/30 transition-colors flex items-start gap-4">
                      <div className="mt-1 w-8 h-8 rounded-full bg-blue-500/10 flex items-center justify-center shrink-0">
                        <Star className="h-4 w-4 text-blue-500" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between">
                          <p className="text-sm font-medium text-foreground capitalize">
                            {activity.action_type.replace(/_/g, ' ')}
                          </p>
                          <span
                            className={`text-sm font-bold ${
                              activity.points > 0 ? 'text-green-500' : 'text-red-500'
                            }`}
                          >
                            {activity.points > 0 ? '+' : ''}
                            {activity.points}
                          </span>
                        </div>
                        {activity.reason && (
                          <p className="text-xs text-muted-foreground mt-0.5">{activity.reason}</p>
                        )}
                        <span className="text-[10px] text-muted-foreground block mt-1">
                          {formatDate(activity.timestamp)}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-10 text-muted-foreground">
                  <Zap className="h-10 w-10 mx-auto text-muted mb-2" />
                  <p>Herhangi bir aktivite bulunamadı</p>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Edit Modal */}
      {isEditing && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-card rounded-xl border border-border w-full max-w-md p-6 shadow-xl">
            <div className="flex justify-between items-center mb-6">
              <h3 className="text-lg font-semibold text-foreground">Kullanıcıyı Düzenle</h3>
            </div>
            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">İsim</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-transparent outline-none transition-all"
                  required
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Puan</label>
                  <input
                    type="number"
                    value={points}
                    onChange={(e) => setPoints(parseInt(e.target.value) || 0)}
                    className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-transparent outline-none transition-all"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">Seviye</label>
                  <input
                    type="number"
                    value={level}
                    min={1}
                    onChange={(e) => setLevel(parseInt(e.target.value) || 1)}
                    className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-transparent outline-none transition-all"
                    required
                  />
                </div>
              </div>
              <div className="flex justify-end gap-3 pt-4">
                <button
                  type="button"
                  onClick={() => setIsEditing(false)}
                  className="px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors"
                >
                  İptal
                </button>
                <button
                  type="submit"
                  disabled={updateMutation.isPending}
                  className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors flex items-center gap-2"
                >
                  {updateMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                  Kaydet
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
