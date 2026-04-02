import { useQuery } from '@tanstack/react-query'
import { Bug, CheckCircle2, XCircle, Clock, AlertCircle, Loader2, ArrowRight } from 'lucide-react'
import { getEventLogs } from '@/lib/api'

export default function EventDebugger() {
  const { data: logsData, isLoading } = useQuery({
    queryKey: ['event-logs'],
    queryFn: getEventLogs,
    refetchInterval: 5000, // auto-refresh every 5s for real-time feel
  })

  const logs = logsData?.data || []

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-foreground">Event Debugger</h2>
          <p className="text-muted-foreground">Real-time view of rule engine evaluations</p>
        </div>
      </div>

      <div className="grid gap-4">
        {isLoading && logs.length === 0 ? (
          <div className="h-[400px] flex items-center justify-center">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : logs.length === 0 ? (
           <div className="h-[400px] flex flex-col items-center justify-center text-muted-foreground">
             <Bug className="h-12 w-12 mb-4 opacity-20" />
             <p>No recent events found in the debug log.</p>
           </div>
        ) : (
          logs.map((log, i) => (
            <div key={i} className="rounded-xl border border-border bg-card overflow-hidden">
              <div className="p-4 border-b border-border bg-muted/20 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className={`w-8 h-8 rounded flex items-center justify-center ${log.success ? 'bg-green-500/10 text-green-500' : (log.skipped ? 'bg-amber-500/10 text-amber-500' : 'bg-red-500/10 text-red-500')}`}>
                    {log.success ? <CheckCircle2 className="w-4 h-4" /> : (log.skipped ? <AlertCircle className="w-4 h-4" /> : <XCircle className="w-4 h-4" />)}
                  </div>
                  <div>
                    <h4 className="font-semibold text-foreground flex items-center gap-2">
                      Event: <span className="font-mono text-xs px-2 py-0.5 rounded bg-muted">{log.event?.event_type || 'Unknown'}</span>
                    </h4>
                    <p className="text-xs text-muted-foreground flex items-center gap-1">
                      <Clock className="w-3 h-3" />
                      {log.timestamp ? new Date(log.timestamp).toLocaleString() : 'Just now'} 
                      <span className="mx-1">&bull;</span>
                      User: {log.event?.actor_id || log.event?.subject_id || log.event?.player_id || 'System'}
                      {log.event?.event_id && <><span className="mx-1">&bull;</span>ID: {log.event.event_id}</>}
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-xs font-mono text-muted-foreground">
                    {log.total_time_ms ? `${log.total_time_ms.toFixed(2)}ms` : '< 1ms'}
                  </div>
                </div>
              </div>

              <div className="p-4 bg-card">
                {log.error && (
                  <div className="mb-4 p-3 rounded bg-red-500/10 border border-red-500/20 text-red-500 text-sm">
                    {log.error}
                  </div>
                )}
                {log.skipped && (
                  <div className="mb-4 p-3 rounded bg-amber-500/10 border border-amber-500/20 text-amber-500 text-sm">
                    Skipped: {log.skip_reason || 'Unknown reason'}
                  </div>
                )}

                <div className="space-y-3">
                  <h5 className="text-sm font-medium text-foreground">Rule Evaluations</h5>
                  {(!log.triggered_rules || log.triggered_rules.length === 0) ? (
                    <p className="text-sm text-muted-foreground pl-4 border-l-2 border-border">
                      No rules matched this event type.
                    </p>
                  ) : (
                    <div className="space-y-2">
                      {log.triggered_rules.map((ruleResult: any, idx: number) => (
                        <div key={idx} className="flex flex-col gap-2 p-3 rounded bg-muted/10 border border-border">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                              {ruleResult.Matched ? <CheckCircle2 className="w-4 h-4 text-green-500" /> : <XCircle className="w-4 h-4 text-muted-foreground" />}
                              <span className="text-sm font-medium">{ruleResult.Rule?.name || ruleResult.Rule?.rule_id || 'Unknown Rule'}</span>
                            </div>
                            <span className="text-xs text-muted-foreground">
                              {ruleResult.EvalTimeMs ? `${ruleResult.EvalTimeMs.toFixed(2)}ms` : ''}
                            </span>
                          </div>
                          
                          {ruleResult.Matched && ruleResult.Actions?.length > 0 && (
                            <div className="pl-6 pt-1 flex items-start gap-2">
                              <ArrowRight className="w-4 h-4 text-muted-foreground shrink-0 mt-0.5" />
                              <div className="flex flex-wrap gap-2">
                                {ruleResult.Actions.map((action: any, aIdx: number) => (
                                  <span key={aIdx} className="text-xs px-2 py-1 rounded bg-primary/10 text-primary">
                                    {action.action_type || action.ActionType}
                                  </span>
                                ))}
                                {ruleResult.Users?.length > 0 && (
                                  <span className="text-xs px-2 py-1 rounded bg-muted text-muted-foreground">
                                    Affected: {ruleResult.Users.join(', ')}
                                  </span>
                                )}
                              </div>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                <div className="mt-4 pt-4 border-t border-border">
                  <details>
                    <summary className="text-sm text-muted-foreground cursor-pointer hover:text-foreground">
                      View Raw Payload
                    </summary>
                    <pre className="mt-2 p-4 rounded bg-muted/50 text-xs overflow-x-auto text-foreground font-mono">
                      {JSON.stringify(log.event, null, 2)}
                    </pre>
                  </details>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
