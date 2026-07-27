package gmessages

import (
	"sync"

	"go.mau.fi/mautrix-gmessages/pkg/libgm"
)

type clientEventGate struct {
	mu     sync.RWMutex
	active bool
}

var clientEventGates sync.Map // map[*libgm.Client]*clientEventGate

func (m *Manager) registerClientEventHandler(client *libgm.Client) {
	gate := &clientEventGate{active: true}
	clientEventGates.Store(client, gate)
	client.SetEventHandler(func(raw any) {
		gate.mu.RLock()
		defer gate.mu.RUnlock()
		if !gate.active {
			return
		}
		m.handleClientEvent(client, raw)
	})
}

// retireClient waits for any event handler that already started for this
// client, disables future events, then closes the libgm transport. It must not
// be called from inside that same client's event handler.
func (m *Manager) retireClient(client *libgm.Client) {
	if client == nil {
		return
	}
	if value, ok := clientEventGates.LoadAndDelete(client); ok {
		gate := value.(*clientEventGate)
		gate.mu.Lock()
		gate.active = false
		gate.mu.Unlock()
	}
	client.Disconnect()
}
