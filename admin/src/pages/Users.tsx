import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Search, Edit, Trash2, Loader2, X, Award, Star, Eye } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { getUsers, updateUser, deleteUser, User } from '@/lib/api'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

interface RichBadgeInfo {
  id: string
  name: string
  description: string
  icon: string
  points: number
  earned_at: string
  reason: string
}

interface UserModalProps {
  user: User | null
  onClose: () => void
}

function UserModal({ user, onClose }: UserModalProps) {
  const queryClient = useQueryClient()

  // Rich badge info from backend
  const richBadgeInfo: RichBadgeInfo[] = (user as any)?.rich_badge_info || []
  const recentActivity = (user as any)?.recent_activity || []

  const [name, setName] = useState(user?.name || '')
  const [email, setEmail] = useState(user?.email || '')
  const [points, setPoints] = useState(user?.points || 0)
  const [level, setLevel] = useState(user?.level || 1)

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<User> }) => updateUser(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      toast.success('User updated successfully')
      onClose()
    },
    onError: () => toast.error('Failed to update user'),
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (user) {
      updateMutation.mutate({ id: user.id, data: { name, email, points, level } })
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-card rounded-xl border border-border w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-6 border-b border-border">
          <h3 className="text-lg font-semibold text-foreground">Kullanıcıyı Düzenle</h3>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Points</label>
              <input
                type="number"
                value={points}
                onChange={(e) => setPoints(parseInt(e.target.value) || 0)}
                className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Level</label>
              <input
                type="number"
                value={level}
                onChange={(e) => setLevel(parseInt(e.target.value) || 1)}
                min={1}
                className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
              />
            </div>
          </div>

          {/* Badge History from rich_badge_info */}
          {richBadgeInfo.length > 0 && (
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Badge History</label>
              <div className="space-y-2 max-h-40 overflow-y-auto">
                {richBadgeInfo.map((badge) => (
                  <div
                    key={badge.id}
                    className="flex items-center justify-between p-2 rounded-lg bg-muted"
                  >
                    <div className="flex items-center gap-2">
                      <Award className="h-4 w-4 text-yellow-500" />
                      <span className="text-sm text-foreground">{badge.name}</span>
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {formatDate(badge.earned_at)}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Recent Activity */}
          {recentActivity.length > 0 && (
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Recent Activity</label>
              <div className="space-y-2 max-h-40 overflow-y-auto">
                {recentActivity.slice(0, 5).map((activity: any, index: number) => (
                  <div
                    key={index}
                    className="flex items-center justify-between p-2 rounded-lg bg-muted"
                  >
                    <div>
                      <span className="text-sm text-foreground">{activity.action_type}</span>
                      {activity.reason && (
                        <p className="text-xs text-muted-foreground">{activity.reason}</p>
                      )}
                    </div>
                    <div className="text-right">
                      <span className="text-sm font-medium text-green-500">+{activity.points}</span>
                      <p className="text-xs text-muted-foreground">{formatDate(activity.timestamp)}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="flex justify-end gap-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={updateMutation.isPending}
              className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {updateMutation.isPending && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Save Changes
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default function Users() {
  const [search, setSearch] = useState('')
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)

  const queryClient = useQueryClient()
  const navigate = useNavigate()

  const { data, isLoading } = useQuery({
    queryKey: ['users', { search }],
    queryFn: () => getUsers({ search }),
  })

  const deleteMutation = useMutation({
    mutationFn: deleteUser,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      toast.success('Kullanıcı başarıyla silindi')
      setDeleteConfirmId(null)
    },
    onError: () => toast.error('Kullanıcı silinemedi'),
  })

  const users: User[] = data?.data || []

  const filteredUsers = users.filter(
    (user: User) =>
      user.name.toLowerCase().includes(search.toLowerCase()) ||
      user.email.toLowerCase().includes(search.toLowerCase())
  )

  const handleEdit = (user: User) => {
    setSelectedUser(user)
    setIsModalOpen(true)
  }

  const handleDelete = (id: string) => {
    setDeleteConfirmId(id)
  }

  const confirmDelete = () => {
    if (deleteConfirmId) {
      deleteMutation.mutate(deleteConfirmId)
    }
  }

  const cancelDelete = () => {
    setDeleteConfirmId(null)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-foreground">Kullanıcı Yönetimi</h2>
          <p className="text-muted-foreground">Kullanıcıları, puanları ve rozetleri yönet</p>
        </div>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <input
          type="text"
          placeholder="Search users..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full pl-10 pr-4 py-2 rounded-lg border border-input bg-background text-foreground"
        />
      </div>

      {/* Users Table */}
      {isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : filteredUsers.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">Kullanıcı bulunamadı</div>
      ) : (
        <div className="border border-border rounded-xl bg-card overflow-hidden shadow-sm">
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="text-xs uppercase bg-muted/40 text-muted-foreground border-b border-border">
                <tr>
                  <th className="px-6 py-4 font-medium">Kullanıcı (User)</th>
                  <th className="px-6 py-4 font-medium">Seviye & Puan</th>
                  <th className="px-6 py-4 font-medium">Kayıt Tarihi</th>
                  <th className="px-6 py-4 font-medium text-right">Aksiyon</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {filteredUsers.map((user: User) => (
                  <tr key={user.id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                          <span className="text-sm font-semibold text-primary">
                            {user.name.charAt(0).toUpperCase()}
                          </span>
                        </div>
                        <div className="min-w-0">
                          <h3 className="font-semibold text-foreground truncate">{user.name}</h3>
                          <p className="text-xs text-muted-foreground truncate">{user.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex flex-col gap-1">
                        <div className="flex items-center gap-1.5">
                          <Star className="h-3.5 w-3.5 text-yellow-500 fill-yellow-500/20" />
                          <span className="font-medium text-foreground">Level {user.level}</span>
                        </div>
                        <div className="text-xs font-semibold text-green-500">{user.points.toLocaleString()} pts</div>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-xs text-muted-foreground whitespace-nowrap">
                      {formatDate(user.createdAt)}
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex gap-1 justify-end">
                        <button
                          onClick={() => navigate(`/users/${user.id}`)}
                          className="p-1.5 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                          title="Görüntüle"
                        >
                          <Eye className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleEdit(user)}
                          className="p-1.5 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                          title="Düzenle"
                        >
                          <Edit className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(user.id)}
                          className="p-1.5 rounded-md hover:bg-red-500/10 text-muted-foreground hover:text-destructive transition-colors"
                          title="Sil"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Modal */}
      {isModalOpen && selectedUser && (
        <UserModal user={selectedUser} onClose={() => { setIsModalOpen(false); setSelectedUser(null) }} />
      )}

      {/* Delete Confirmation Dialog */}
      {deleteConfirmId && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-card rounded-xl border border-border w-full max-w-md p-6">
            <div className="flex items-center gap-4 mb-4">
              <div className="p-3 rounded-full bg-red-500/10">
                <Trash2 className="h-6 w-6 text-red-500" />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-foreground">Kullanıcıyı Sil</h3>
                <p className="text-sm text-muted-foreground">Bu işlem geri alınamaz</p>
              </div>
            </div>
            <p className="text-sm text-muted-foreground mb-6">
              Bu kullanıcıyı silmek istediğinizden emin misiniz? Tüm verileri kalıcı olarak silinecek.
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={cancelDelete}
                className="px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors"
              >
                İptal
              </button>
              <button
                onClick={confirmDelete}
                disabled={deleteMutation.isPending}
                className="px-4 py-2 rounded-lg bg-red-500 text-white hover:bg-red-600 disabled:opacity-50 transition-colors flex items-center gap-2"
              >
                {deleteMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                Sil
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}