package dashboard

import (
	"github.com/jongio/azd-app/cli/src/internal/rpc"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// newLogsStoreFuncs builds the rpc.LogsStoreFuncs adapter that backs
// LogsService for the dashboard. It closes over server.go's existing
// helpers so the Connect handler shares state (mutexes, cached config
// clients, the LogManager singleton) with the legacy REST handlers
// during the parallel-stack window described in ADR-0001.
//
// The classifications path serialises behind classificationsMu (a
// package-level RWMutex shared with handleGetClassifications /
// handleCreateClassification / handleDeleteClassification) so concurrent
// Add/Delete calls across both surfaces don't lose writes. Preferences
// share s.configClient via getOrCreateConfigClient so REST writes and
// Connect writes target the same azdconfig instance.
func newLogsStoreFuncs(s *Server) rpc.LogsStoreFuncs {
	return rpc.LogsStoreFuncs{
		GetRecentFn: func(serviceName string, tail int) ([]service.LogEntry, bool) {
			lm := service.GetLogManager(s.projectDir)
			if lm == nil {
				return nil, false
			}
			buf, exists := lm.GetBuffer(serviceName)
			if !exists || buf == nil {
				return nil, false
			}
			return buf.GetRecent(tail), true
		},
		GetAllFn: func(tail int) []service.LogEntry {
			lm := service.GetLogManager(s.projectDir)
			if lm == nil {
				return nil
			}
			return lm.GetAllLogs(tail)
		},
		ServiceNamesFn: func() []string {
			lm := service.GetLogManager(s.projectDir)
			if lm == nil {
				return nil
			}
			return lm.GetServiceNames()
		},
		SubscribeFn: func(serviceName string) (chan service.LogEntry, bool) {
			lm := service.GetLogManager(s.projectDir)
			if lm == nil {
				return nil, false
			}
			buf, exists := lm.GetBuffer(serviceName)
			if !exists || buf == nil {
				return nil, false
			}
			return buf.Subscribe(), true
		},
		UnsubscribeFn: func(serviceName string, ch chan service.LogEntry) {
			lm := service.GetLogManager(s.projectDir)
			if lm == nil {
				return
			}
			buf, exists := lm.GetBuffer(serviceName)
			if !exists || buf == nil {
				return
			}
			buf.Unsubscribe(ch)
		},
		OnBufferAddedFn: func() <-chan string {
			lm := service.GetLogManager(s.projectDir)
			if lm == nil {
				// No manager: return nil so the stream's select never
				// fires on this channel. The stream still serves
				// already-registered services.
				return nil
			}
			return lm.OnBufferAdded()
		},
		RemoveBufferListenerFn: func(ch <-chan string) {
			lm := service.GetLogManager(s.projectDir)
			if lm == nil {
				return
			}
			lm.RemoveBufferListener(ch)
		},

		LoadClassificationsFn: func() ([]service.LogClassification, error) {
			classificationsMu.RLock()
			defer classificationsMu.RUnlock()
			ay, err := loadAzureYaml(s.projectDir)
			if err != nil {
				// Mirror handleGetClassifications: an azure.yaml that
				// can't be read or parsed is an empty list, not an
				// error. The Connect surface keeps this lenient
				// behaviour so the parallel REST and Connect surfaces
				// agree on "no project, no rules".
				return []service.LogClassification{}, nil
			}
			if ay == nil || ay.Logs == nil {
				return []service.LogClassification{}, nil
			}
			return ay.Logs.GetClassifications(), nil
		},
		SaveClassificationsFn: func(classifications []service.LogClassification) error {
			classificationsMu.Lock()
			defer classificationsMu.Unlock()
			ay, err := loadAzureYaml(s.projectDir)
			if err != nil {
				return err
			}
			if ay == nil {
				ay = &service.AzureYaml{}
			}
			if len(classifications) == 0 {
				// Match handleDeleteClassification's cleanup: if the
				// last rule is gone and no filters remain, drop the
				// logs section entirely so azure.yaml stays tidy.
				if ay.Logs != nil && ay.Logs.Filters == nil {
					ay.Logs = nil
				} else if ay.Logs != nil {
					ay.Logs.Classifications = nil
				}
			} else {
				if ay.Logs == nil {
					ay.Logs = &service.LogsConfig{}
				}
				ay.Logs.Classifications = classifications
			}
			return saveAzureYaml(s.projectDir, ay)
		},

		LoadPreferencesFn: func() ([]byte, error) {
			return s.getOrCreateConfigClient().GetPreferenceSection(prefsConfigKey)
		},
		SavePreferencesFn: func(data []byte) error {
			return s.getOrCreateConfigClient().SetPreferenceSection(prefsConfigKey, data)
		},
	}
}
