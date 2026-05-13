package dashboard

// prefsConfigKey is the azdconfig key under which the Logs preferences blob
// is stored. Referenced by rpc_logs_adapter.go when wiring the Connect
// LogsService preferences get/save RPCs to the shared config client.
const prefsConfigKey = "logs"
