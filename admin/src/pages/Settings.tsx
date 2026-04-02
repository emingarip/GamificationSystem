import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Save, Settings as SettingsIcon, Bell, Shield, Palette, Globe, Loader2 } from 'lucide-react'
import { updateConfig } from '@/lib/api'
import { toast } from 'sonner'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'

const settingsSchema = z.object({
  appName: z.string().min(1, 'App name is required'),
  apiUrl: z.string().url('Invalid API URL'),
  wsUrl: z.string().url('Invalid WebSocket URL'),
  language: z.string(),
  timezone: z.string(),
  allowRegistration: z.boolean(),
  requireEmailVerification: z.boolean(),
  maxPointsPerDay: z.number().min(0),
  pointsExpirationDays: z.number().min(0),
  enableNotifications: z.boolean(),
  enableEmailNotifications: z.boolean(),
  enablePushNotifications: z.boolean(),
  logLevel: z.enum(['debug', 'info', 'warn', 'error']),
  corsOrigins: z.string(),
})

type SettingsFormData = z.infer<typeof settingsSchema>

export default function Settings() {
  const [activeTab, setActiveTab] = useState('general')
  const origin = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:5173'
  const wsProtocol = typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'wss' : 'ws'
  const wsHost = typeof window !== 'undefined' ? window.location.host : 'localhost:5173'

  const { register, handleSubmit, formState: { errors, isDirty } } = useForm<SettingsFormData>({
    resolver: zodResolver(settingsSchema),
    defaultValues: {
      appName: 'Gamification Platform',
      apiUrl: `${origin}/api/v1`,
      wsUrl: `${wsProtocol}://${wsHost}/ws`,
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
    },
  })

  const updateMutation = useMutation({
    mutationFn: (data: SettingsFormData) => updateConfig(data),
    onSuccess: () => {
      toast.success('Settings saved successfully')
    },
    onError: () => {
      toast.error('Failed to save settings')
    },
  })

  const onSubmit = (data: SettingsFormData) => {
    updateMutation.mutate(data)
  }

  const tabs = [
    { id: 'general', label: 'General', icon: SettingsIcon },
    { id: 'notifications', label: 'Notifications', icon: Bell },
    { id: 'security', label: 'Security', icon: Shield },
    { id: 'appearance', label: 'Appearance', icon: Palette },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold text-foreground">Settings</h2>
        <p className="text-muted-foreground">Manage application configuration</p>
      </div>

      <div className="flex flex-col lg:flex-row gap-6">
        {/* Tabs */}
        <div className="lg:w-64 shrink-0">
          <nav className="flex lg:flex-col gap-1 p-1 rounded-xl bg-card border border-border">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-3 px-4 py-2 rounded-lg text-left transition-colors ${
                  activeTab === tab.id
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:text-foreground hover:bg-muted'
                }`}
              >
                <tab.icon className="h-4 w-4" />
                <span className="text-sm font-medium">{tab.label}</span>
              </button>
            ))}
          </nav>
        </div>

        {/* Content */}
        <div className="flex-1">
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
            {activeTab === 'general' && (
              <div className="p-6 rounded-xl bg-card border border-border space-y-6">
                <h3 className="text-lg font-semibold text-foreground flex items-center gap-2">
                  <Globe className="h-5 w-5" />
                  General Settings
                </h3>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-foreground mb-2">App Name</label>
                    <input
                      {...register('appName')}
                      className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
                    />
                    {errors.appName && <p className="mt-1 text-sm text-destructive">{errors.appName.message}</p>}
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-foreground mb-2">Language</label>
                    <select
                      {...register('language')}
                      className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
                    >
                      <option value="en">English</option>
                      <option value="es">Spanish</option>
                      <option value="fr">French</option>
                      <option value="de">German</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-foreground mb-2">API URL</label>
                    <input
                      {...register('apiUrl')}
                      className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
                    />
                    {errors.apiUrl && <p className="mt-1 text-sm text-destructive">{errors.apiUrl.message}</p>}
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-foreground mb-2">WebSocket URL</label>
                    <input
                      {...register('wsUrl')}
                      className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
                    />
                    {errors.wsUrl && <p className="mt-1 text-sm text-destructive">{errors.wsUrl.message}</p>}
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-foreground mb-2">Max Points Per Day</label>
                    <input
                      type="number"
                      {...register('maxPointsPerDay', { valueAsNumber: true })}
                      className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-foreground mb-2">Points Expiration (days)</label>
                    <input
                      type="number"
                      {...register('pointsExpirationDays', { valueAsNumber: true })}
                      className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
                    />
                  </div>
                </div>

                <div className="space-y-3">
                  <label className="flex items-center gap-3">
                    <input
                      type="checkbox"
                      {...register('allowRegistration')}
                      className="w-4 h-4 rounded border-input"
                    />
                    <span className="text-sm text-foreground">Allow user registration</span>
                  </label>

                  <label className="flex items-center gap-3">
                    <input
                      type="checkbox"
                      {...register('requireEmailVerification')}
                      className="w-4 h-4 rounded border-input"
                    />
                    <span className="text-sm text-foreground">Require email verification</span>
                  </label>
                </div>
              </div>
            )}

            {activeTab === 'notifications' && (
              <div className="p-6 rounded-xl bg-card border border-border space-y-6">
                <h3 className="text-lg font-semibold text-foreground flex items-center gap-2">
                  <Bell className="h-5 w-5" />
                  Notification Settings
                </h3>

                <div className="space-y-4">
                  <label className="flex items-center justify-between p-4 rounded-lg bg-muted/50">
                    <div>
                      <p className="font-medium text-foreground">Enable Notifications</p>
                      <p className="text-sm text-muted-foreground">Allow in-app notifications</p>
                    </div>
                    <input
                      type="checkbox"
                      {...register('enableNotifications')}
                      className="w-5 h-5 rounded border-input accent-primary"
                    />
                  </label>

                  <label className="flex items-center justify-between p-4 rounded-lg bg-muted/50">
                    <div>
                      <p className="font-medium text-foreground">Email Notifications</p>
                      <p className="text-sm text-muted-foreground">Receive notifications via email</p>
                    </div>
                    <input
                      type="checkbox"
                      {...register('enableEmailNotifications')}
                      className="w-5 h-5 rounded border-input accent-primary"
                    />
                  </label>

                  <label className="flex items-center justify-between p-4 rounded-lg bg-muted/50">
                    <div>
                      <p className="font-medium text-foreground">Push Notifications</p>
                      <p className="text-sm text-muted-foreground">Receive browser push notifications</p>
                    </div>
                    <input
                      type="checkbox"
                      {...register('enablePushNotifications')}
                      className="w-5 h-5 rounded border-input accent-primary"
                    />
                  </label>
                </div>
              </div>
            )}

            {activeTab === 'security' && (
              <div className="p-6 rounded-xl bg-card border border-border space-y-6">
                <h3 className="text-lg font-semibold text-foreground flex items-center gap-2">
                  <Shield className="h-5 w-5" />
                  Security Settings
                </h3>

                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-foreground mb-2">CORS Origins</label>
                    <input
                      {...register('corsOrigins')}
                      className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
                      placeholder="* or comma-separated URLs"
                    />
                    <p className="mt-1 text-xs text-muted-foreground">Use * to allow all origins</p>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-foreground mb-2">Log Level</label>
                    <select
                      {...register('logLevel')}
                      className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
                    >
                      <option value="debug">Debug</option>
                      <option value="info">Info</option>
                      <option value="warn">Warning</option>
                      <option value="error">Error</option>
                    </select>
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'appearance' && (
              <div className="p-6 rounded-xl bg-card border border-border space-y-6">
                <h3 className="text-lg font-semibold text-foreground flex items-center gap-2">
                  <Palette className="h-5 w-5" />
                  Appearance Settings
                </h3>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-foreground mb-2">Timezone</label>
                    <select
                      {...register('timezone')}
                      className="w-full px-4 py-2 rounded-lg border border-input bg-background text-foreground"
                    >
                      <option value="UTC">UTC</option>
                      <option value="America/New_York">Eastern Time</option>
                      <option value="America/Chicago">Central Time</option>
                      <option value="America/Denver">Mountain Time</option>
                      <option value="America/Los_Angeles">Pacific Time</option>
                      <option value="Europe/London">London</option>
                      <option value="Europe/Paris">Paris</option>
                      <option value="Asia/Tokyo">Tokyo</option>
                    </select>
                  </div>
                </div>

                <div className="p-4 rounded-lg bg-muted/50">
                  <p className="text-sm text-muted-foreground">
                    Theme customization is handled via the theme toggle in the sidebar.
                  </p>
                </div>
              </div>
            )}

            {/* Save Button */}
            <div className="flex justify-end gap-3">
              <button
                type="submit"
                disabled={updateMutation.isPending || !isDirty}
                className="flex items-center gap-2 px-6 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                {updateMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Save className="h-4 w-4" />
                )}
                Save Changes
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
