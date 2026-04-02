import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Search, Edit, Trash2, Loader2, X, Zap, ToggleLeft, ToggleRight } from 'lucide-react'
import { getEventTypes, createEventType, updateEventType, deleteEventType, EventTypeInfo } from '@/lib/api'
import { toast } from 'sonner'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'

// Helper to normalize category values from backend (e.g., "sports" -> "sport")
const normalizeCategory = (category?: string): string => {
  if (!category) return 'custom'
  if (category === 'sports') return 'sport'
  return category
}

const eventTypeSchema = z.object({
  key: z.string().min(1, 'Anahtar gereklidir').regex(/^[a-z_]+$/, 'Sadece küçük harf ve alt çizgi'),
  name: z.string().min(1, 'İsim gereklidir'),
  description: z.string().optional(),
  category: z.string().optional(),
  enabled: z.boolean().default(true),
})

type EventTypeFormData = z.infer<typeof eventTypeSchema>

interface EventTypeModalProps {
  eventType?: EventTypeInfo | null
  onClose: () => void
}

function EventTypeModal({ eventType, onClose }: EventTypeModalProps) {
  const queryClient = useQueryClient()
  
  const { register, handleSubmit, formState: { errors }, setValue, watch } = useForm<EventTypeFormData>({
    resolver: zodResolver(eventTypeSchema),
    defaultValues: {
      key: eventType?.key || '',
      name: eventType?.name || '',
      description: eventType?.description || '',
      category: normalizeCategory(eventType?.category) || 'custom',
      enabled: eventType?.enabled ?? true,
    },
  })

  const isEditing = !!eventType

  const createMutation = useMutation({
    mutationFn: createEventType,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['event-types'] })
      toast.success('Event type başarıyla oluşturuldu')
      onClose()
    },
    onError: () => toast.error('Event type oluşturulamadı'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ key, data }: { key: string; data: Partial<EventTypeFormData> }) => 
      updateEventType(key, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['event-types'] })
      toast.success('Event type başarıyla güncellendi')
      onClose()
    },
    onError: () => toast.error('Event type güncellenemedi'),
  })

  const onSubmit = (data: EventTypeFormData) => {
    if (isEditing && eventType) {
      updateMutation.mutate({ key: eventType.key, data })
    } else {
      createMutation.mutate(data)
    }
  }

  const isLoading = createMutation.isPending || updateMutation.isPending
  const enabled = watch('enabled')

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-card rounded-xl border border-border w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-6 border-b border-border">
          <h3 className="text-lg font-semibold text-foreground">
            {isEditing ? 'Event Type Düzenle' : 'Yeni Event Type'}
          </h3>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Anahtar (Key)</label>
            <input
              {...register('key')}
              disabled={isEditing}
              placeholder="örn: daily_login, goal_scored"
              className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground font-mono disabled:opacity-50"
            />
            {errors.key && <p className="mt-1 text-sm text-destructive">{errors.key.message}</p>}
            <p className="mt-1 text-xs text-muted-foreground">Benzersiz tanımlayıcı (sadece küçük harf ve alt çizgi)</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-2">İsim</label>
            <input
              {...register('name')}
              placeholder="örn: Günlük Giriş"
              className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
            />
            {errors.name && <p className="mt-1 text-sm text-destructive">{errors.name.message}</p>}
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Açıklama</label>
            <textarea
              {...register('description')}
              placeholder="Bu event type ne için kullanılır?"
              rows={2}
              className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Kategori</label>
            <select
              {...register('category')}
              className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
            >
              <option value="general">Genel</option>
              <option value="sport">Spor</option>
              <option value="custom">Özel</option>
              <option value="social">Sosyal</option>
              <option value="engagement">Etkileşim</option>
              <option value="achievement">Başarı</option>
            </select>
          </div>

          <div className="flex items-center gap-3 p-4 rounded-lg bg-muted/50 border border-border">
            <button
              type="button"
              onClick={() => setValue('enabled', !enabled)}
              className="text-muted-foreground hover:text-foreground"
            >
              {enabled ? (
                <ToggleRight className="h-6 w-6 text-green-500" />
              ) : (
                <ToggleLeft className="h-6 w-6" />
              )}
            </button>
            <div className="flex-1">
              <label className="text-sm font-medium text-foreground">Aktif</label>
              <p className="text-xs text-muted-foreground">Pasif event type'lar tetiklenmez</p>
            </div>
            <input type="hidden" {...register('enabled')} value={enabled ? 'true' : 'false'} />
          </div>

          <div className="flex justify-end gap-3 pt-4 border-t border-border">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted"
            >
              İptal
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="px-6 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 flex items-center gap-2"
            >
              {isLoading && <Loader2 className="h-4 w-4 animate-spin" />}
              {isEditing ? 'Güncelle' : 'Oluştur'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default function EventTypes() {
  const [search, setSearch] = useState('')
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [selectedEventType, setSelectedEventType] = useState<EventTypeInfo | null>(null)
  const [deleteId, setDeleteId] = useState<string | null>(null)

  const queryClient = useQueryClient()

  const { data: eventTypesData, isLoading } = useQuery({
    queryKey: ['event-types'],
    queryFn: getEventTypes,
  })

  const deleteMutation = useMutation({
    mutationFn: deleteEventType,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['event-types'] })
      toast.success('Event type silindi')
      setDeleteId(null)
    },
    onError: () => toast.error('Event type silinemedi'),
  })

  const toggleMutation = useMutation({
    mutationFn: ({ key, enabled }: { key: string; enabled: boolean }) => 
      updateEventType(key, { enabled }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['event-types'] })
      toast.success('Durum güncellendi')
    },
    onError: () => toast.error('Durum güncellenemedi'),
  })

  const eventTypes: EventTypeInfo[] = eventTypesData || []

  const filteredEventTypes = eventTypes.filter(
    (et) =>
      et.key.toLowerCase().includes(search.toLowerCase()) ||
      et.name.toLowerCase().includes(search.toLowerCase()) ||
      et.category?.toLowerCase().includes(search.toLowerCase())
  )

  const handleEdit = (eventType: EventTypeInfo) => {
    setSelectedEventType(eventType)
    setIsModalOpen(true)
  }

  const handleDelete = (key: string) => {
    setDeleteId(key)
  }

  const confirmDelete = () => {
    if (deleteId) {
      deleteMutation.mutate(deleteId)
    }
  }

  const handleToggle = (eventType: EventTypeInfo) => {
    toggleMutation.mutate({ key: eventType.key, enabled: !eventType.enabled })
  }

  const handleCloseModal = () => {
    setIsModalOpen(false)
    setSelectedEventType(null)
  }

  const categoryColors: Record<string, string> = {
    general: 'bg-gray-500',
    sport: 'bg-green-500',
    custom: 'bg-orange-500',
    social: 'bg-blue-500',
    engagement: 'bg-purple-500',
    achievement: 'bg-yellow-500',
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-foreground">Event Types</h2>
          <p className="text-muted-foreground">Sistem event type'larını yönet</p>
        </div>
        <button
          onClick={() => setIsModalOpen(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          Yeni Event Type
        </button>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <input
          type="text"
          placeholder="Event type ara..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full pl-10 pr-4 py-2 rounded-lg border border-input bg-background text-foreground"
        />
      </div>

      {/* Event Types Table */}
      <div className="rounded-xl border border-border overflow-hidden">
        <table className="w-full">
          <thead className="bg-muted/50">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Anahtar</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">İsim</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Kategori</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Durum</th>
              <th className="px-4 py-3 text-right text-sm font-medium text-foreground">İşlemler</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {isLoading ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                  <Loader2 className="h-6 w-6 animate-spin mx-auto" />
                </td>
              </tr>
            ) : filteredEventTypes.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                  <div className="flex flex-col items-center gap-2">
                    <Zap className="h-8 w-8 opacity-50" />
                    <p>Henüz event type yok</p>
                    <p className="text-sm">İlk event type'ı oluşturmak için yukarıdaki butona tıkla</p>
                  </div>
                </td>
              </tr>
            ) : (
              filteredEventTypes.map((et) => (
                <tr key={et.key} className="hover:bg-muted/50">
                  <td className="px-4 py-3">
                    <span className="font-mono text-sm text-foreground">{et.key}</span>
                  </td>
                  <td className="px-4 py-3">
                    <div>
                      <p className="text-sm font-medium text-foreground">{et.name}</p>
                      {et.description && (
                        <p className="text-xs text-muted-foreground line-clamp-1">{et.description}</p>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-1 rounded-full text-xs text-white ${categoryColors[normalizeCategory(et.category)]}`}>
                      {normalizeCategory(et.category)}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <button
                      onClick={() => handleToggle(et)}
                      disabled={toggleMutation.isPending}
                      className="flex items-center gap-1"
                    >
                      {et.enabled ? (
                        <span className="flex items-center gap-1 text-xs text-green-500">
                          <span className="w-2 h-2 rounded-full bg-green-500" />
                          Aktif
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 text-xs text-muted-foreground">
                          <span className="w-2 h-2 rounded-full bg-gray-400" />
                          Pasif
                        </span>
                      )}
                    </button>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex justify-end gap-2">
                      <button
                        onClick={() => handleEdit(et)}
                        className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground"
                      >
                        <Edit className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() => handleDelete(et.key)}
                        className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-destructive"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Modal */}
      {isModalOpen && (
        <EventTypeModal 
          eventType={selectedEventType} 
          onClose={handleCloseModal} 
        />
      )}

      {/* Delete Confirmation Modal */}
      {deleteId && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-card rounded-xl border border-border w-full max-w-sm">
            <div className="p-6 text-center">
              <div className="w-12 h-12 rounded-full bg-destructive/10 flex items-center justify-center mx-auto mb-4">
                <Trash2 className="h-6 w-6 text-destructive" />
              </div>
              <h3 className="text-lg font-semibold text-foreground mb-2">Event Type Sil</h3>
              <p className="text-muted-foreground mb-6">
                <span className="text-foreground font-medium">{deleteId}</span> event type'ını silmek istediğinizden emin misiniz?
                <br />
                <span className="text-xs text-yellow-500">Bu event type'a bağlı kurallar etkilenebilir.</span>
              </p>
              <div className="flex gap-3">
                <button
                  onClick={() => setDeleteId(null)}
                  className="flex-1 px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted"
                >
                  İptal
                </button>
                <button
                  onClick={confirmDelete}
                  disabled={deleteMutation.isPending}
                  className="flex-1 px-4 py-2 rounded-lg bg-destructive text-destructive-foreground hover:bg-destructive/90 disabled:opacity-50"
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