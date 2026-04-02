import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Search, Edit, Trash2, Loader2, X, Award } from 'lucide-react'
import { getBadges, createBadge, updateBadge, deleteBadge, Badge } from '@/lib/api'
import { toast } from 'sonner'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'

const badgeSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  description: z.string().min(1, 'Description is required'),
  icon: z.string().min(1, 'Icon is required'),
  color: z.string().min(1, 'Color is required'),
  points: z.number().min(0),
  rarity: z.enum(['common', 'rare', 'epic', 'legendary']),
  requirements: z.string(),
})

type BadgeFormData = z.infer<typeof badgeSchema>

interface BadgeModalProps {
  badge: Badge | null
  onClose: () => void
}

function BadgeModal({ badge, onClose }: BadgeModalProps) {
  const queryClient = useQueryClient()

  const { register, handleSubmit, formState: { errors } } = useForm<BadgeFormData>({
    resolver: zodResolver(badgeSchema),
    defaultValues: badge || {
      name: '',
      description: '',
      icon: 'award',
      color: '#3b82f6',
      points: 0,
      rarity: 'common',
      requirements: '{}',
    },
  })

  const createMutation = useMutation({
    mutationFn: createBadge,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['badges'] })
      toast.success('Badge created successfully')
      onClose()
    },
    onError: () => toast.error('Failed to create badge'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Badge> }) => updateBadge(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['badges'] })
      toast.success('Badge updated successfully')
      onClose()
    },
    onError: () => toast.error('Failed to update badge'),
  })

  const onSubmit = (data: BadgeFormData) => {
    if (badge) {
      updateMutation.mutate({ id: badge.id, data })
    } else {
      createMutation.mutate(data)
    }
  }

  const isLoading = createMutation.isPending || updateMutation.isPending

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-card rounded-xl border border-border w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-6 border-b border-border">
          <h3 className="text-lg font-semibold text-foreground">
            {badge ? 'Edit Badge' : 'Create Badge'}
          </h3>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Name</label>
            <input
              {...register('name')}
              className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
            />
            {errors.name && <p className="mt-1 text-sm text-destructive">{errors.name.message}</p>}
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Description</label>
            <textarea
              {...register('description')}
              rows={3}
              className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
            />
            {errors.description && <p className="mt-1 text-sm text-destructive">{errors.description.message}</p>}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Icon</label>
              <select
                {...register('icon')}
                className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
              >
                <option value="award">Award</option>
                <option value="star">Star</option>
                <option value="trophy">Trophy</option>
                <option value="medal">Medal</option>
                <option value="crown">Crown</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Color</label>
              <input
                type="color"
                {...register('color')}
                className="w-full h-10 rounded-lg border border-input"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Points</label>
              <input
                type="number"
                {...register('points', { valueAsNumber: true })}
                className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Rarity</label>
              <select
                {...register('rarity')}
                className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
              >
                <option value="common">Common</option>
                <option value="rare">Rare</option>
                <option value="epic">Epic</option>
                <option value="legendary">Legendary</option>
              </select>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Requirements (JSON)</label>
            <textarea
              {...register('requirements')}
              placeholder='{"minPoints": 500, "level": 5}'
              rows={2}
              className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground font-mono text-sm"
            />
          </div>

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
              disabled={isLoading}
              className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {isLoading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              {badge ? 'Update' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

const rarityColors = {
  common: 'bg-gray-500',
  rare: 'bg-blue-500',
  epic: 'bg-purple-500',
  legendary: 'bg-yellow-500',
}

export default function Badges() {
  const [search, setSearch] = useState('')
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [selectedBadge, setSelectedBadge] = useState<Badge | null>(null)
  const [deleteId, setDeleteId] = useState<string | null>(null)

  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['badges'],
    queryFn: () => getBadges(),
  })

  const deleteMutation = useMutation({
    mutationFn: deleteBadge,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['badges'] })
      toast.success('Badge deleted successfully')
    },
    onError: () => toast.error('Failed to delete badge'),
  })

  const badges: Badge[] = data?.data || []

  const filteredBadges = badges.filter(
    (badge: Badge) =>
      badge.name.toLowerCase().includes(search.toLowerCase()) ||
      badge.description.toLowerCase().includes(search.toLowerCase())
  )

  const handleEdit = (badge: Badge) => {
    setSelectedBadge(badge)
    setIsModalOpen(true)
  }

  const handleDelete = (id: string) => {
    setDeleteId(id)
  }

  const confirmDelete = () => {
    if (deleteId) {
      deleteMutation.mutate(deleteId)
      setDeleteId(null)
    }
  }

  const handleCloseModal = () => {
    setIsModalOpen(false)
    setSelectedBadge(null)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-foreground">Badges Management</h2>
          <p className="text-muted-foreground">Create and manage achievement badges</p>
        </div>
        <button
          onClick={() => setIsModalOpen(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          Add Badge
        </button>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <input
          type="text"
          placeholder="Search badges..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full pl-10 pr-4 py-2 rounded-lg border border-input bg-background text-foreground"
        />
      </div>

      {/* Badges Grid */}
      {isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : filteredBadges.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">No badges found</div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {filteredBadges.map((badge) => (
            <div
              key={badge.id}
              className="p-6 rounded-xl bg-card border border-border hover:border-primary/50 transition-colors"
            >
              <div className="flex items-start justify-between">
                <div
                  className="w-14 h-14 rounded-xl flex items-center justify-center"
                  style={{ backgroundColor: badge.color + '20' }}
                >
                  <Award className="h-7 w-7" style={{ color: badge.color }} />
                </div>
                <div className="flex gap-1">
                  <button
                    onClick={() => handleEdit(badge)}
                    className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground"
                  >
                    <Edit className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => handleDelete(badge.id)}
                    className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-destructive"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>

              <h3 className="mt-4 font-semibold text-foreground">{badge.name}</h3>
              <p className="mt-1 text-sm text-muted-foreground line-clamp-2">{badge.description}</p>

              <div className="mt-4 flex items-center justify-between">
                <span className={`px-2 py-1 rounded-full text-xs font-medium text-white ${rarityColors[badge.rarity] || ''}`}>
                  {badge.rarity}
                </span>
                <span className="text-sm font-semibold text-foreground">{badge.points} pts</span>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Modal */}
      {isModalOpen && <BadgeModal badge={selectedBadge} onClose={handleCloseModal} />}

      {/* Delete Confirmation Modal */}
      {deleteId && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-card rounded-xl border border-border w-full max-w-sm">
            <div className="p-6 text-center">
              <div className="w-12 h-12 rounded-full bg-destructive/10 flex items-center justify-center mx-auto mb-4">
                <Trash2 className="h-6 w-6 text-destructive" />
              </div>
              <h3 className="text-lg font-semibold text-foreground mb-2">Rozeti Sil</h3>
              <p className="text-muted-foreground mb-6">Bu rozeti silmek istediğinizden emin misiniz? Bu işlem geri alınamaz.</p>
              <div className="flex gap-3">
                <button
                  onClick={() => setDeleteId(null)}
                  className="flex-1 px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors"
                >
                  İptal
                </button>
                <button
                  onClick={confirmDelete}
                  disabled={deleteMutation.isPending}
                  className="flex-1 px-4 py-2 rounded-lg bg-destructive text-destructive-foreground hover:bg-destructive/90 disabled:opacity-50 transition-colors"
                >
                  {deleteMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin mx-auto" /> : 'Sil'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}