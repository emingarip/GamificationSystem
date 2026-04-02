import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Search, Edit, Trash2, Sparkles, Loader2, X, Play, Zap, Info } from 'lucide-react'
import { getRules, createRule, updateRule, deleteRule, generateRule, testEvent, getEventTypes, getBadges, Rule, EventTypeInfo } from '@/lib/api'
import { toast } from 'sonner'

const ruleSchema = z.object({
  name: z.string().min(1, 'Kural adı gereklidir'),
  description: z.string().min(1, 'Açıklama gereklidir'),
  type: z.enum(['points', 'badge', 'level', 'streak']),
  trigger: z.string().min(1, 'Tetikleyici gereklidir'),
  conditions: z.string(),
  actions: z.string(),
  enabled: z.boolean(),
  priority: z.number().min(0),
})

type RuleFormData = z.infer<typeof ruleSchema>

interface RuleModalProps {
  rule?: Rule | null
  onClose: () => void
  eventTypes?: Array<{ key: string; name: string }>
}function RuleModal({ rule, onClose, eventTypes = [] }: RuleModalProps) {
  const queryClient = useQueryClient()
  const [currentStep, setCurrentStep] = useState(1)
  const totalSteps = 4
  const [trigger, setTrigger] = useState(rule?.trigger || '')
  const [customTrigger, setCustomTrigger] = useState('')
  
  // Use Query for Badges
  const { data: badgesResponse } = useQuery({
    queryKey: ['badges-rule-modal'],
    queryFn: () => getBadges({ limit: 100 }),
    staleTime: 60000,
  })
  const availableBadges = badgesResponse?.data || []

  // Visual Conditions State
  const parseInitialConditions = () => {
    try {
      const parsed = JSON.parse(rule?.conditions || '[]')
      return Array.isArray(parsed) && parsed.length > 0
        ? parsed
        : [{ field: '', operator: '==', value: '', evaluation_type: 'simple' }]
    } catch {
      return [{ field: '', operator: '==', value: '', evaluation_type: 'simple' }]
    }
  }
  const [conditions, setConditions] = useState<any[]>(parseInitialConditions())

  // Visual Actions State
  const parseInitialActions = () => {
    try {
      const parsed = JSON.parse(rule?.actions || '[]')
      return Array.isArray(parsed) ? parsed : []
    } catch {
      return []
    }
  }
  const [actions, setActions] = useState<any[]>(parseInitialActions())
  
  // Hidden values for Hook Form sync
  const { register, handleSubmit, formState: { errors }, setValue, trigger: formTrigger } = useForm<RuleFormData>({
    resolver: zodResolver(ruleSchema),
    defaultValues: rule || {
      name: '',
      description: '',
      type: 'points',
      trigger: '',
      conditions: '{}',
      actions: '[]',
      enabled: true,
      priority: 0,
    },
  })

  const createMutation = useMutation({
    mutationFn: createRule,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      toast.success('Kural başarıyla oluşturuldu')
      onClose()
    },
    onError: () => toast.error('Kural oluşturulamadı'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Rule> }) => updateRule(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      toast.success('Kural başarıyla güncellendi')
      onClose()
    },
    onError: () => toast.error('Kural güncellenemedi'),
  })

  const onSubmit = (data: RuleFormData) => {
    // Sync visual states into JSON strings before submission
    const cleanConditions = conditions.filter(c => c.field && c.value !== '')
    data.conditions = JSON.stringify(cleanConditions)
    
    // Actions are directly mapped
    data.actions = JSON.stringify(actions.filter(a => !!a.action_type))

    if (rule) {
      updateMutation.mutate({ id: rule.id, data })
    } else {
      createMutation.mutate(data)
    }
  }

  const isLoading = createMutation.isPending || updateMutation.isPending

  const handleNext = async () => {
    let isValid = false
    if (currentStep === 1) {
      isValid = await formTrigger(['name', 'description', 'type', 'priority'])
    } else if (currentStep === 2) {
      if (trigger === '') {
        toast.error('Lütfen bir tetikleyici (trigger) seçin')
        return
      }
      if (trigger === 'custom' && customTrigger === '') {
        toast.error('Lütfen özel tetikleyici adını girin')
        return
      }
      isValid = true
    } else if (currentStep === 3) {
      isValid = true 
    }

    if (isValid) {
      setCurrentStep((prev) => Math.min(prev + 1, totalSteps))
    }
  }

  const handlePrev = () => {
    setCurrentStep((prev) => Math.max(prev - 1, 1))
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-card rounded-xl border border-border w-full max-w-2xl max-h-[90vh] overflow-y-auto flex flex-col relative">
        <div className="flex items-center justify-between p-6 border-b border-border sticky top-0 bg-card z-20">
          <h3 className="text-lg font-semibold text-foreground">
            {rule ? 'Kuralı Düzenle' : 'Yeni Kural Oluştur'}
          </h3>
          <button type="button" onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Stepper Progress */}
        <div className="px-6 pt-6 pb-2 sticky top-[73px] bg-card z-10 border-b border-border/50 shadow-sm">
          <div className="flex items-center justify-between mb-2">
            {[1, 2, 3, 4].map((step) => (
              <div 
                key={step} 
                className={`flex-1 h-2 mx-1 rounded-full transition-colors duration-300 ${
                  currentStep >= step ? 'bg-primary' : 'bg-muted'
                }`} 
              />
            ))}
          </div>
          <div className="flex justify-between text-[11px] sm:text-xs px-1 font-medium select-none">
            <span className={currentStep >= 1 ? 'text-primary' : 'text-muted-foreground'}>Temel Bilgiler</span>
            <span className={currentStep >= 2 ? 'text-primary' : 'text-muted-foreground'}>Tetikleyici</span>
            <span className={currentStep >= 3 ? 'text-primary' : 'text-muted-foreground'}>Koşullar</span>
            <span className={currentStep >= 4 ? 'text-primary' : 'text-muted-foreground'}>Ödüller</span>
          </div>
        </div>

        <form 
          onSubmit={handleSubmit(onSubmit)} 
          onKeyDown={(e) => {
            if (e.key === 'Enter' && (e.target as HTMLElement).tagName !== 'TEXTAREA') {
              e.preventDefault()
            }
          }}
          className="p-6 flex-1 flex flex-col space-y-8"
        >
          <div className="flex-1">
          {/* Section 1: Basic Info */}
          {currentStep === 1 && (
            <div className="space-y-4 animate-in fade-in slide-in-from-right-4 duration-300">
              <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground uppercase tracking-wider mb-6">
                <span className="w-6 h-6 rounded bg-primary text-primary-foreground flex items-center justify-center text-xs shadow-sm">1</span>
                Adım 1: Temel Bilgiler
              </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-foreground mb-2">Kural Adı</label>
                <input
                  {...register('name')}
                  placeholder="Örn: Günlük Giriş Bonusu"
                  className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                />
                {errors.name && <p className="mt-1 text-sm text-destructive">{errors.name.message}</p>}
              </div>

              <div>
                <label className="block text-sm font-medium text-foreground mb-2">Açıklama</label>
                <input
                  {...register('description')}
                  placeholder="Kısa bir açıklama..."
                  className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                />
                {errors.description && <p className="mt-1 text-sm text-destructive">{errors.description.message}</p>}
              </div>

              <div>
                <label className="block text-sm font-medium text-foreground mb-2">Kural Türü</label>
                <select
                  {...register('type')}
                  className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                >
                  <option value="points">Puan Ver</option>
                  <option value="badge">Rozet Ver</option>
                  <option value="level">Seviye Yükselt</option>
                  <option value="streak">Seri Oluştur</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-foreground mb-2">Öncelik (Opsiyonel)</label>
                <input
                  type="number"
                  {...register('priority', { valueAsNumber: true })}
                  placeholder="0"
                  className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                />
                <p className="mt-1 text-xs text-muted-foreground">Küçük sayılar daha önce çalışır.</p>
              </div>
            </div>
          </div>
          )}

          {/* Section 2: Trigger */}
          {currentStep === 2 && (
            <div className="space-y-4 animate-in fade-in slide-in-from-right-4 duration-300">
              <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground uppercase tracking-wider mb-6">
                <span className="w-6 h-6 rounded bg-primary text-primary-foreground flex items-center justify-center text-xs shadow-sm">2</span>
                Adım 2: Neyden Sonra Çalışsın? (Tetikleyici)
              </div>
            <div>
              <select
                value={trigger}
                onChange={(e) => {
                  setTrigger(e.target.value)
                  if (e.target.value !== 'custom') {
                    setValue('trigger', e.target.value)
                  }
                }}
                className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
              >
                <option value="">Olay (Event) Seçin...</option>
                {eventTypes.map((et) => (
                  <option key={et.key} value={et.key}>
                    {et.name} ({et.key})
                  </option>
                ))}
                <option value="custom">Özel (Custom Event)...</option>
              </select>
              {trigger === 'custom' && (
                <div className="mt-2">
                  <input
                    type="text"
                    value={customTrigger}
                    onChange={(e) => {
                      setCustomTrigger(e.target.value)
                      setValue('trigger', e.target.value)
                    }}
                    placeholder="Örn: daily_login"
                    className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                  />
                </div>
              )}
              <input type="hidden" {...register('trigger')} />
              {errors.trigger && <p className="mt-1 text-sm text-destructive">{errors.trigger.message}</p>}
            </div>
          </div>
          )}

          {/* Section 3: Visual Conditions */}
          {currentStep === 3 && (
            <div className="space-y-4 animate-in fade-in slide-in-from-right-4 duration-300">
              <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground uppercase tracking-wider">
                  <span className="w-6 h-6 rounded bg-primary text-primary-foreground flex items-center justify-center text-xs shadow-sm">3</span>
                  Adım 3: Hangi Koşullarda? (Conditions)
                </div>
              <button
                type="button"
                onClick={() => setConditions([...conditions, { field: '', operator: '==', value: '', evaluation_type: 'simple' }])}
                className="flex items-center gap-1 text-xs px-2 py-1 bg-secondary rounded text-secondary-foreground hover:bg-secondary/80"
              >
                <Plus className="w-3 h-3" /> Koşul Ekle
              </button>
            </div>
            
            <div className="space-y-3">
              {conditions.map((cond, idx) => (
                <div key={idx} className="flex flex-wrap items-start gap-3 p-3 bg-muted/30 border border-border rounded-lg relative">
                  <div className="flex-1 min-w-[120px]">
                    <label className="flex items-center gap-1 text-[10px] uppercase text-muted-foreground mb-1">
                      Field (Alan)
                      <div className="group relative">
                        <Info className="w-3 h-3 hover:text-foreground cursor-help" />
                        <div className="absolute left-1/2 -translate-x-1/2 bottom-full mb-1 hidden group-hover:block w-48 p-2 bg-foreground text-background text-xs rounded shadow-lg z-50 normal-case">
                          Kontrol edilecek veri alanı. (örn: global_count veya context.amount)
                        </div>
                      </div>
                    </label>
                    <input 
                      type="text"
                      list="field-suggestions"
                      value={cond.field}
                      onChange={(e) => {
                        const newConds = [...conditions]
                        newConds[idx].field = e.target.value
                        setConditions(newConds)
                      }}
                      placeholder="Örn: global_count"
                      className="w-full px-3 py-1.5 text-sm rounded border border-input bg-background focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  
                  <div className="w-[100px]">
                    <label className="flex items-center gap-1 text-[10px] uppercase text-muted-foreground mb-1">
                      Operatör
                      <div className="group relative">
                        <Info className="w-3 h-3 hover:text-foreground cursor-help" />
                        <div className="absolute left-1/2 -translate-x-1/2 bottom-full mb-1 hidden group-hover:block w-40 p-2 bg-foreground text-background text-xs rounded shadow-lg z-50 normal-case">
                          Alanın değeriyle yapılacak karşılaştırma (örn: Eşit, Büyük).
                        </div>
                      </div>
                    </label>
                    <select 
                      value={cond.operator}
                      onChange={(e) => {
                        const newConds = [...conditions]
                        newConds[idx].operator = e.target.value
                        setConditions(newConds)
                      }}
                      className="w-full px-2 py-1.5 text-sm rounded border border-input bg-background focus:ring-1 focus:ring-primary"
                    >
                      <option value="==">== (Eşit)</option>
                      <option value="!=">!= (Eşit Değil)</option>
                      <option value=">">&gt; (Büyük)</option>
                      <option value=">=">&gt;= (Büyük Eşit)</option>
                      <option value="<">&lt; (Küçük)</option>
                      <option value="<=">&lt;= (Küçük Eşit)</option>
                      <option value="contains">İçerir</option>
                      <option value="every">Her (Modül)</option>
                    </select>
                  </div>

                  <div className="flex-1 min-w-[100px]">
                    <label className="flex items-center gap-1 text-[10px] uppercase text-muted-foreground mb-1">
                      Değer (Value)
                      <div className="group relative">
                        <Info className="w-3 h-3 hover:text-foreground cursor-help" />
                        <div className="absolute left-1/2 -translate-x-1/2 bottom-full mb-1 hidden group-hover:block w-48 p-2 bg-foreground text-background text-xs rounded shadow-lg z-50 normal-case">
                          Koşulun sağlanması için hedeflenen sayı veya metin.
                        </div>
                      </div>
                    </label>
                    <input 
                      type="text"
                      value={cond.value}
                      onChange={(e) => {
                        const newConds = [...conditions]
                        // Try parse float for numbers
                        const val = e.target.value
                        newConds[idx].value = !isNaN(Number(val)) && val !== '' ? Number(val) : val
                        setConditions(newConds)
                      }}
                      placeholder="Değer..."
                      className="w-full px-3 py-1.5 text-sm rounded border border-input bg-background focus:ring-1 focus:ring-primary"
                    />
                  </div>

                  <div className="w-[130px]">
                    <label className="flex items-center gap-1 text-[10px] uppercase text-muted-foreground mb-1">
                      Türü
                      <div className="group relative">
                        <Info className="w-3 h-3 hover:text-foreground cursor-help" />
                        <div className="absolute right-0 bottom-full mb-1 hidden group-hover:block w-52 p-2 bg-foreground text-background text-xs rounded shadow-lg z-50 normal-case whitespace-normal">
                          <span className="font-semibold">Basit:</span> O anki değer<br/>
                          <span className="font-semibold">Toplu:</span> Eski değerlerle toplanır<br/>
                          <span className="font-semibold">Zamanlı:</span> Süreye bağlı işlem
                        </div>
                      </div>
                    </label>
                    <select 
                      value={cond.evaluation_type}
                      onChange={(e) => {
                        const newConds = [...conditions]
                        newConds[idx].evaluation_type = e.target.value
                        setConditions(newConds)
                      }}
                      className="w-full px-2 py-1.5 text-sm rounded border border-input bg-background focus:ring-1 focus:ring-primary"
                    >
                      <option value="simple">Basit (Simple)</option>
                      <option value="aggregation">Toplu (Aggregation)</option>
                      <option value="temporal">Zamanlı (Temporal)</option>
                    </select>
                  </div>

              {conditions.length > 1 && (
                <button 
                  type="button"
                  onClick={() => setConditions(conditions.filter((_, i) => i !== idx))}
                  className="absolute -right-2 -top-2 bg-destructive text-destructive-foreground rounded-full p-1 shadow hover:bg-destructive/90"
                >
                  <X className="w-3 h-3" />
                </button>
              )}
            </div>
          ))}
        </div>
        
        <datalist id="field-suggestions">
          <option value="global_count" />
          <option value="daily_count" />
          <option value="weekly_count" />
          <option value="monthly_count" />
          <option value="streak_count" />
          <option value="player_id" />
          <option value="context.amount" />
          <option value="metadata.score" />
        </datalist>
      </div>
          )}

          {/* Section 4: Visual Rewards */}
          {currentStep === 4 && (
            <div className="space-y-8 animate-in fade-in slide-in-from-right-4 duration-300">
              <div className="flex flex-col gap-4">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground uppercase tracking-wider">
                    <span className="w-6 h-6 rounded bg-primary text-primary-foreground flex items-center justify-center text-xs shadow-sm">4</span>
                    Adım 4: Ödüller (Aksiyonlar)
                  </div>
                  <button
                    type="button"
                    onClick={() => setActions([...actions, { action_type: 'award_points', params: { points: 10, reason: '' } }])}
                    className="flex items-center gap-2 text-xs font-medium text-primary hover:text-primary/80 transition-colors"
                  >
                    <Plus className="w-4 h-4" /> Aksiyon Ekle
                  </button>
                </div>

                {actions.length === 0 && (
                  <div className="text-center p-8 border rounded-lg bg-muted/20 border-dashed text-muted-foreground text-sm">
                    Henüz aksiyon eklenmedi. Kural tetiklendiğinde hiçbir şey olmayacak.
                  </div>
                )}

                {actions.map((act, idx) => (
                  <div key={idx} className="relative flex flex-wrap sm:flex-nowrap gap-3 p-4 border border-input bg-muted/10 rounded-xl items-end group">
                    <div className="w-[180px]">
                      <label className="block text-xs text-muted-foreground mb-1">Aksiyon Tipi</label>
                      <select
                        value={act.action_type}
                        onChange={(e) => {
                          const newActions = [...actions]
                          if (e.target.value === 'award_points') {
                            newActions[idx] = { action_type: 'award_points', params: { points: 10, reason: '' } }
                          } else if (e.target.value === 'grant_badge') {
                            newActions[idx] = { action_type: 'grant_badge', params: { badge_id: '' } }
                          }
                          setActions(newActions)
                        }}
                        className="w-full px-3 py-2 text-sm rounded border border-input bg-background focus:ring-1 focus:ring-primary"
                      >
                        <option value="award_points">Puan Ver</option>
                        <option value="grant_badge">Rozet Ver</option>
                      </select>
                    </div>

                    {act.action_type === 'award_points' && (
                      <>
                        <div className="w-[120px]">
                          <label className="block text-xs text-muted-foreground mb-1">Puan</label>
                          <input
                            type="number"
                            value={act.params?.points || ''}
                            onChange={(e) => {
                              const newActions = [...actions]
                              newActions[idx].params.points = parseInt(e.target.value) || 0
                              setActions(newActions)
                            }}
                            className="w-full px-3 py-2 text-sm rounded border border-input bg-background focus:ring-1 focus:ring-primary"
                          />
                        </div>
                        <div className="flex-1 min-w-[150px]">
                          <label className="block text-xs text-muted-foreground mb-1">Kazanım Nedeni</label>
                          <input
                            type="text"
                            value={act.params?.reason || ''}
                            onChange={(e) => {
                              const newActions = [...actions]
                              newActions[idx].params.reason = e.target.value
                              setActions(newActions)
                            }}
                            placeholder="Örn: Günlük Giriş"
                            className="w-full px-3 py-2 text-sm rounded border border-input bg-background focus:ring-1 focus:ring-primary"
                          />
                        </div>
                      </>
                    )}

                    {act.action_type === 'grant_badge' && (
                      <div className="flex-1 min-w-[200px]">
                        <label className="block text-xs text-muted-foreground mb-1">Verilecek Rozet</label>
                        <select
                          value={act.params?.badge_id || ''}
                          onChange={(e) => {
                            const newActions = [...actions]
                            newActions[idx].params.badge_id = e.target.value
                            setActions(newActions)
                          }}
                          className="w-full px-3 py-2 text-sm rounded border border-input bg-background focus:ring-1 focus:ring-primary"
                        >
                          <option value="">Rozet Seçin</option>
                          {availableBadges.map((b: any) => (
                            <option key={b.id} value={b.id}>{b.name} ({b.id})</option>
                          ))}
                        </select>
                      </div>
                    )}

                    <button
                      type="button"
                      onClick={() => setActions(actions.filter((_, i) => i !== idx))}
                      className="absolute -right-2 -top-2 bg-destructive text-destructive-foreground rounded-full p-1.5 shadow opacity-0 group-hover:opacity-100 transition-opacity"
                    >
                      <X className="w-3 h-3" />
                    </button>
                  </div>
                ))}
              </div>
              
              {/* Section 5: Enabled */}
              <div className="flex items-center gap-3 p-4 rounded-lg bg-muted/50 border border-border mt-4">
                <input
                  type="checkbox"
                  {...register('enabled')}
                  id="enabled"
                  className="w-5 h-5 rounded border-input text-primary focus:ring-primary cursor-pointer"
                />
                <div className="flex-1">
                  <label htmlFor="enabled" className="text-sm font-medium text-foreground cursor-pointer">Aksiyonlara Açık (Enabled)</label>
                  <p className="text-xs text-muted-foreground">İşaretli değilse kural sisteme kaydolur ancak tetiklenmez.</p>
                </div>
              </div>
            </div>
          )}
        </div>

          {/* Wizard Footer Controls */}
          <div className="flex justify-between items-center mt-auto pt-6 border-t border-border sticky bottom-0 bg-card z-20 pb-2">
            <div>
              {currentStep > 1 ? (
                <button
                  type="button"
                  onClick={handlePrev}
                  className="px-5 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors flex items-center gap-2 text-sm font-medium"
                >
                  Geri
                </button>
              ) : (
                <button
                  type="button"
                  onClick={onClose}
                  className="px-5 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors text-sm font-medium"
                >
                  İptal
                </button>
              )}
            </div>

            <div>
              {currentStep < totalSteps && (
                <button
                  key="next-btn"
                  type="button"
                  onClick={(e) => {
                    e.preventDefault()
                    handleNext()
                  }}
                  className="px-8 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors text-sm font-medium shadow-sm"
                >
                  İleri
                </button>
              )}
              
              {currentStep === totalSteps && (
                <button
                  key="submit-btn"
                  type="submit"
                  disabled={isLoading}
                  className="px-8 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors flex items-center gap-2 text-sm font-medium shadow-sm"
                >
                  {isLoading && <Loader2 className="h-4 w-4 animate-spin" />}
                  {rule ? 'Değişiklikleri Kaydet' : 'Kuralı Oluştur'}
                </button>
              )}
            </div>
          </div>
        </form>
      </div>
    </div>
  )
}
interface GenerateModalProps {
  onClose: () => void
}

function GenerateModal({ onClose }: GenerateModalProps) {
  const queryClient = useQueryClient()
  const [prompt, setPrompt] = useState('')
  const [generatedRule, setGeneratedRule] = useState<Rule | null>(null)
  const [isGenerating, setIsGenerating] = useState(false)

  const generateMutation = useMutation({
    mutationFn: generateRule,
    onMutate: () => setIsGenerating(true),
    onSuccess: (data) => {
      setGeneratedRule(data.rule)
      toast.success('Kural başarıyla oluşturuldu')
    },
    onError: () => toast.error('Kural oluşturulamadı'),
    onSettled: () => setIsGenerating(false),
  })

  const saveMutation = useMutation({
    mutationFn: createRule,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      toast.success('Kural kaydedildi')
      onClose()
    },
    onError: () => toast.error('Kural kaydedilemedi'),
  })

  const handleGenerate = () => {
    generateMutation.mutate(prompt)
  }

  const handleSave = () => {
    if (generatedRule) {
      const { id, createdAt, updatedAt, ...rest } = generatedRule
      saveMutation.mutate(rest)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-card rounded-xl border border-border w-full max-w-lg">
        <div className="flex items-center justify-between p-6 border-b border-border">
          <h3 className="text-lg font-semibold text-foreground flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-yellow-500" />
            AI Rule Generator
          </h3>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">
              Describe your rule
            </label>
            <textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Create a rule that awards 100 points when a user completes 5 daily challenges in a row..."
              rows={4}
              className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
            />
          </div>

          <button
            onClick={handleGenerate}
            disabled={!prompt || isGenerating}
            className="w-full flex items-center justify-center gap-2 py-2 px-4 rounded-lg bg-gradient-to-r from-yellow-500 to-orange-500 text-white font-medium hover:opacity-90 disabled:opacity-50"
          >
            {isGenerating ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Sparkles className="h-4 w-4" />
            )}
            Generate Rule
          </button>

          {generatedRule && (
            <div className="mt-4 p-4 rounded-lg bg-muted border border-border">
              <h4 className="font-medium text-foreground mb-2">{generatedRule.name}</h4>
              <p className="text-sm text-muted-foreground mb-2">{generatedRule.description}</p>
              <div className="text-xs text-muted-foreground">
                <p>Type: {generatedRule.type}</p>
                <p>Trigger: {generatedRule.trigger}</p>
              </div>
              <button
                onClick={handleSave}
                disabled={saveMutation.isPending}
                className="mt-3 w-full py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90"
              >
                {saveMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin mx-auto" /> : 'Save Rule'}
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

interface TestEventModalProps {
  onClose: () => void
  eventTypes?: Array<{ key: string; name: string }>
}

function TestEventModal({ onClose, eventTypes = [] }: TestEventModalProps) {
  const queryClient = useQueryClient()
  const [eventId, setEventId] = useState('')
  const [eventType, setEventType] = useState('')
  const [customEventType, setCustomEventType] = useState('')
  const [matchId, setMatchId] = useState('')
  const [teamId, setTeamId] = useState('')
  const [playerId, setPlayerId] = useState('')
  const [minute, setMinute] = useState<number>(0)
  const [metadata, setMetadata] = useState('')
  const [metadataError, setMetadataError] = useState<string | null>(null)
  // Generic event fields for non-sport events
  const [subjectId, setSubjectId] = useState('')
  const [actorId, setActorId] = useState('')
  const [source, setSource] = useState('')
  const [context, setContext] = useState('')
  const [contextError, setContextError] = useState<string | null>(null)
  const [dryRun, setDryRun] = useState(true)
  const [result, setResult] = useState<any>(null)
  const [isTesting, setIsTesting] = useState(false)

  // Use custom event type if "custom" is selected
  const effectiveEventType = eventType === 'custom' ? customEventType : eventType

  const testMutation = useMutation({
    mutationFn: async () => {
      let parsedMetadata = {}
      if (metadata.trim()) {
        try {
          parsedMetadata = JSON.parse(metadata)
          setMetadataError(null)
        } catch (e: any) {
          setMetadataError('Geçersiz JSON formatı: ' + e.message)
          toast.error('Metadata geçersiz JSON')
          return
        }
      }
      return testEvent({
        event: {
          event_id: eventId || `test-${Date.now()}`,
          event_type: effectiveEventType || eventType,
          match_id: matchId,
          team_id: teamId,
          player_id: playerId,
          minute: minute,
          timestamp: new Date().toISOString(),
          metadata: parsedMetadata,
          // Generic fields for non-sport events
          subject_id: subjectId,
          actor_id: actorId,
          source: source,
          context: context ? (() => { try { return JSON.parse(context) } catch (e: any) { setContextError('Geçersiz JSON: ' + e.message); return undefined } })() : undefined,
        },
        dry_run: dryRun,
      })
    },
    onMutate: () => setIsTesting(true),
    onSuccess: (data) => {
      setResult(data)
      toast.success('Event başarıyla test edildi')
      queryClient.invalidateQueries({ queryKey: ['rules'] })
    },
    onError: (error: any) => {
      setResult({ error: error.response?.data?.error || error.message })
      toast.error('Event test edilemedi')
    },
    onSettled: () => setIsTesting(false),
  })

  const handleTest = () => {
    if (!eventType) {
      toast.error('Event tipi gereklidir')
      return
    }
    testMutation.mutate()
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-card rounded-xl border border-border w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-6 border-b border-border">
          <h3 className="text-lg font-semibold text-foreground flex items-center gap-2">
            <Zap className="h-5 w-5 text-blue-500" />
            Test Event
          </h3>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="p-6 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Event ID</label>
              <input
                type="text"
                value={eventId}
                onChange={(e) => setEventId(e.target.value)}
                placeholder="Event ID (opsiyonel)"
                className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Event Tipi</label>
              <select
                value={eventType}
                onChange={(e) => setEventType(e.target.value)}
                className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
              >
                <option value="">Event tipi seçin...</option>
                {eventTypes.map((et) => (
                  <option key={et.key} value={et.key}>
                    {et.name} ({et.key})
                  </option>
                ))}
                <option value="custom">Özel...</option>
              </select>
            </div>
          </div>

          {eventType === 'custom' && (
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Özel Event Tipi</label>
              <input
                type="text"
                value={customEventType}
                onChange={(e) => setCustomEventType(e.target.value)}
                placeholder="e.g., daily_login, app_shared"
                className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
              />
            </div>
          )}

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Maç ID</label>
              <input
                type="text"
                value={matchId}
                onChange={(e) => setMatchId(e.target.value)}
                placeholder="Match identifier"
                className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Takım ID</label>
              <input
                type="text"
                value={teamId}
                onChange={(e) => setTeamId(e.target.value)}
                placeholder="Team identifier"
                className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Oyuncu ID</label>
              <input
                type="text"
                value={playerId}
                onChange={(e) => setPlayerId(e.target.value)}
                placeholder="Player identifier"
                className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-2">Dakika</label>
              <input
                type="number"
                value={minute}
                onChange={(e) => setMinute(parseInt(e.target.value) || 0)}
                placeholder="Match minute"
                className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Meta Veri (JSON)</label>
            <textarea
              value={metadata}
              onChange={(e) => {
                setMetadata(e.target.value)
                setMetadataError(null)
              }}
              placeholder='{"key": "value"}'
              rows={3}
              className={`w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground font-mono text-sm ${metadataError ? 'border-destructive focus:ring-destructive' : ''}`}
            />
            {metadataError && <p className="mt-1 text-xs text-destructive">{metadataError}</p>}
            <p className="mt-1 text-xs text-muted-foreground">Opsiyonel: Event için ek veriler</p>
          </div>

          {/* Generic Event Fields (for non-sport events) */}
          <div className="border-t border-border pt-4">
            <h4 className="text-sm font-medium text-foreground mb-3">Generic Event Fields (optional)</h4>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-foreground mb-2">Subject ID</label>
                <input
                  type="text"
                  value={subjectId}
                  onChange={(e) => setSubjectId(e.target.value)}
                  placeholder="What the event is about"
                  className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-2">Actor ID</label>
                <input
                  type="text"
                  value={actorId}
                  onChange={(e) => setActorId(e.target.value)}
                  placeholder="Who triggered the event"
                  className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-2">Source</label>
                <input
                  type="text"
                  value={source}
                  onChange={(e) => setSource(e.target.value)}
                  placeholder="Where the event came from"
                  className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
                />
              </div>
            </div>
            <div className="mt-3">
              <label className="block text-sm font-medium text-foreground mb-2">Context (JSON)</label>
              <textarea
                value={context}
                onChange={(e) => {
                  setContext(e.target.value)
                  setContextError(null)
                }}
                placeholder='{"key": "value"}'
                rows={2}
                className={`w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground font-mono text-sm ${contextError ? 'border-destructive' : ''}`}
              />
              {contextError && <p className="mt-1 text-xs text-destructive">{contextError}</p>}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="dryRun"
              checked={dryRun}
              onChange={(e) => setDryRun(e.target.checked)}
              className="w-4 h-4 rounded border-input"
            />
            <label htmlFor="dryRun" className="text-sm text-foreground">Dry Run (don't actually award rewards)</label>
          </div>

          <button
            onClick={handleTest}
            disabled={!eventType || isTesting}
            className="w-full flex items-center justify-center gap-2 py-2 px-4 rounded-lg bg-blue-500 text-white font-medium hover:bg-blue-600 disabled:opacity-50"
          >
            {isTesting ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Play className="h-4 w-4" />
            )}
            Test Event
          </button>

          {result && (
            <div className="mt-4 p-4 rounded-lg bg-muted border border-border">
              <h4 className="font-medium text-foreground mb-2">Test Results</h4>
              {result.error ? (
                <p className="text-sm text-destructive">{result.error}</p>
              ) : (
                <div className="space-y-2 text-xs">
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">Executed:</span>
                    <span className={result.executed ? 'text-green-500' : 'text-yellow-500'}>
                      {result.executed ? 'Yes' : 'No (dry run)'}
                    </span>
                  </div>
                  {result.matches && result.matches.length > 0 && (
                    <div>
                      <span className="text-muted-foreground">Matched Rules:</span>
                      <div className="mt-1 space-y-1">
                        {result.matches.map((match: any, index: number) => (
                          <div key={index} className="flex items-center gap-2">
                            <span className={match.matched ? 'text-green-500' : 'text-muted-foreground'}>
                              {match.matched ? '✓' : '✗'}
                            </span>
                            <span className="text-foreground">{match.name}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                  {result.affected_users && result.affected_users.length > 0 && (
                    <div>
                      <span className="text-muted-foreground">Affected Users:</span>
                      <p className="text-foreground">{result.affected_users.join(', ')}</p>
                    </div>
                  )}
                  {result.actions && result.actions.length > 0 && (
                    <div>
                      <span className="text-muted-foreground">Actions:</span>
                      <div className="mt-1 space-y-1">
                        {result.actions.map((action: any, index: number) => (
                          <div key={index} className="text-foreground">
                            {action.action_type}: {JSON.stringify(action.params)}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default function Rules() {
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [isGenerateModalOpen, setIsGenerateModalOpen] = useState(false)
  const [isTestEventModalOpen, setIsTestEventModalOpen] = useState(false)
  const [selectedRule, setSelectedRule] = useState<Rule | null>(null)

  const queryClient = useQueryClient()

  const { data: rulesData, isLoading } = useQuery({
    queryKey: ['rules', { search, type: typeFilter }],
    queryFn: () => getRules({ type: typeFilter || undefined }),
  })

  // Fetch event types from registry
  const { data: eventTypesData } = useQuery({
    queryKey: ['event-types'],
    queryFn: getEventTypes,
    staleTime: 60000, // 1 minute
  })

  // Use enabled event types for selectors, fallback to all if none enabled
  const eventTypes = (eventTypesData || []).filter((et: EventTypeInfo) => et.enabled)

  const deleteMutation = useMutation({
    mutationFn: deleteRule,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      toast.success('Kural başarıyla silindi')
    },
    onError: () => toast.error('Kural silinemedi'),
  })

  const rules: Rule[] = rulesData?.data || []

  const filteredRules = rules.filter(
    (rule: Rule) => rule.name.toLowerCase().includes(search.toLowerCase()) ||
             rule.description.toLowerCase().includes(search.toLowerCase())
  )

  const handleEdit = (rule: Rule) => {
    setSelectedRule(rule)
    setIsModalOpen(true)
  }

  const [deleteId, setDeleteId] = useState<string | null>(null)

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
    setSelectedRule(null)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-foreground">Kural Yönetimi</h2>
          <p className="text-muted-foreground">Oyunlaştırma kurallarını oluştur ve yönet</p>
        </div>
        <div className="flex gap-2 flex-wrap">
          <button
            onClick={() => setIsTestEventModalOpen(true)}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-500/10 text-blue-600 border border-blue-500/20 hover:bg-blue-500/20 transition-colors font-medium"
          >
            <Zap className="h-4 w-4" />
            Test Et
          </button>
          <button
            onClick={() => setIsGenerateModalOpen(true)}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-gradient-to-r from-yellow-500 to-orange-500 text-white font-medium hover:opacity-90 transition-opacity"
          >
            <Sparkles className="h-4 w-4" />
            AI Oluştur
          </button>
          <button
            onClick={() => setIsModalOpen(true)}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            <Plus className="h-4 w-4" />
            Yeni Kural
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Kural ara..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-10 pr-4 py-2 rounded-lg border border-input bg-background text-foreground"
          />
        </div>
        <select
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value)}
          className="px-4 py-2 rounded-lg border border-input bg-background text-foreground min-w-[150px]"
        >
          <option value="">Tüm Türler</option>
          <option value="points">Puan</option>
          <option value="badge">Rozet</option>
          <option value="level">Seviye</option>
          <option value="streak">Seri</option>
        </select>
      </div>

      {/* Rules Table */}
      <div className="rounded-xl border border-border overflow-hidden">
        <table className="w-full">
          <thead className="bg-muted/50">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Name</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Type</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Trigger</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Ödül</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Priority</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-foreground">Status</th>
              <th className="px-4 py-3 text-right text-sm font-medium text-foreground">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {isLoading ? (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-muted-foreground">
                  <Loader2 className="h-6 w-6 animate-spin mx-auto" />
                </td>
              </tr>
            ) : filteredRules.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-muted-foreground">
                  No rules found
                </td>
              </tr>
            ) : (
              filteredRules.map((rule: Rule) => (
                <tr key={rule.id} className="hover:bg-muted/50">
                  <td className="px-4 py-3">
                    <div>
                      <p className="text-sm font-medium text-foreground">{rule.name}</p>
                      <p className="text-xs text-muted-foreground">{rule.description}</p>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span className="px-2 py-1 rounded-full text-xs font-medium bg-primary/10 text-primary">
                      {rule.type}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-muted-foreground font-mono">{rule.trigger}</td>
                  <td className="px-4 py-3">
                    {(() => {
                      try {
                        const a = JSON.parse(rule.actions)
                        if (!Array.isArray(a)) return <span className="text-xs text-muted-foreground">—</span>
                        
                        const parts = a.map((action: any) => {
                          if (action.action_type === 'grant_points') return `${action.params.amount} Puan`
                          if (action.action_type === 'grant_badge') return `🏅 ${action.params.badge_id}`
                          return action.action_type
                        })
                        return (
                          <span className="px-2 py-1 rounded-full text-xs font-semibold bg-yellow-500/10 text-yellow-600">
                            {parts.length > 0 ? parts.join(' + ') : '—'}
                          </span>
                        )
                      } catch {
                        return <span className="text-xs text-muted-foreground">—</span>
                      }
                    })()}
                  </td>
                  <td className="px-4 py-3 text-sm text-foreground">{rule.priority}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                      rule.enabled ? 'bg-green-500/10 text-green-500' : 'bg-muted text-muted-foreground'
                    }`}>
                      {rule.enabled ? 'Active' : 'Disabled'}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex justify-end gap-2">
                      <button
                        onClick={() => handleEdit(rule)}
                        className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground"
                      >
                        <Edit className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() => handleDelete(rule.id)}
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

      {/* Modals */}
      {isModalOpen && <RuleModal rule={selectedRule} onClose={handleCloseModal} eventTypes={eventTypes} />}
      {isGenerateModalOpen && <GenerateModal onClose={() => setIsGenerateModalOpen(false)} />}
      {isTestEventModalOpen && <TestEventModal onClose={() => setIsTestEventModalOpen(false)} eventTypes={eventTypes} />}

      {/* Delete Confirmation Modal */}
      {deleteId && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-card rounded-xl border border-border w-full max-w-sm">
            <div className="p-6 text-center">
              <div className="w-12 h-12 rounded-full bg-destructive/10 flex items-center justify-center mx-auto mb-4">
                <Trash2 className="h-6 w-6 text-destructive" />
              </div>
              <h3 className="text-lg font-semibold text-foreground mb-2">Kuralı Sil</h3>
              <p className="text-muted-foreground mb-6">Bu kuralı silmek istediğinizden emin misiniz? Bu işlem geri alınamaz.</p>
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