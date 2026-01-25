package control

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/internal/module"
)

const (
	stateFileName      = "state.json"
	stateSchemaVersion = 1
	maxHistoryEntries  = 25
)

type StateManager struct {
	mu       sync.Mutex
	dataPath string
	state    SystemState
}

type SystemState struct {
	SchemaVersion int                    `json:"schema_version"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Modules       map[string]ModuleState `json:"modules"`
	History       []StateSnapshot        `json:"history"`
}

type ModuleState struct {
	Name      string       `json:"name"`
	Version   string       `json:"version"`
	State     module.State `json:"state"`
	UpdatedAt time.Time    `json:"updated_at"`
}

type StateSnapshot struct {
	ID        string                 `json:"id"`
	Reason    string                 `json:"reason"`
	CreatedAt time.Time              `json:"created_at"`
	Modules   map[string]ModuleState `json:"modules"`
}

type StateCorrection struct {
	ModuleName string
	Previous   module.State
	Corrected  module.State
}

func NewStateManager(dataDir string) (*StateManager, error) {
	if dataDir == "" {
		return nil, i18n.Errorf("state data directory is empty")
	}

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, i18n.Errorf("create state directory: %w", err)
	}

	manager := &StateManager{
		dataPath: filepath.Join(dataDir, stateFileName),
		state: SystemState{
			SchemaVersion: stateSchemaVersion,
			UpdatedAt:     time.Now().UTC(),
			Modules:       map[string]ModuleState{},
			History:       []StateSnapshot{},
		},
	}

	if err := manager.Load(); err != nil {
		return nil, err
	}

	return manager, nil
}

func (m *StateManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return m.saveLocked()
		}
		return i18n.Errorf("read state file: %w", err)
	}

	if err := json.Unmarshal(data, &m.state); err != nil {
		return i18n.Errorf("parse state file: %w", err)
	}

	if m.state.Modules == nil {
		m.state.Modules = map[string]ModuleState{}
	}
	return nil
}

func (m *StateManager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *StateManager) saveLocked() error {
	m.state.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return i18n.Errorf("marshal state: %w", err)
	}

	tmpPath := m.dataPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return i18n.Errorf("write state temp file: %w", err)
	}
	if err := os.Rename(tmpPath, m.dataPath); err != nil {
		return i18n.Errorf("replace state file: %w", err)
	}
	return nil
}

func (m *StateManager) GetState() SystemState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state.clone()
}

func (m *StateManager) GetModule(name string) (ModuleState, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	moduleState, ok := m.state.Modules[name]
	return moduleState, ok
}

func (m *StateManager) UpdateModule(name, version string, state module.State) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == "" {
		return i18n.Errorf("module name is required")
	}

	m.pushSnapshotLocked(i18n.T("update module %s", name))
	m.state.Modules[name] = ModuleState{
		Name:      name,
		Version:   version,
		State:     state,
		UpdatedAt: time.Now().UTC(),
	}
	return m.saveLocked()
}

func (m *StateManager) RollbackTo(snapshotID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	index := -1
	for i, snapshot := range m.state.History {
		if snapshot.ID == snapshotID {
			index = i
			break
		}
	}
	if index == -1 {
		return i18n.Errorf("snapshot %s not found", snapshotID)
	}

	m.pushSnapshotLocked(i18n.T("pre-rollback"))
	m.state.Modules = cloneModules(m.state.History[index].Modules)
	return m.saveLocked()
}

func (m *StateManager) RollbackLast() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.state.History) == 0 {
		return i18n.Errorf("no snapshots available")
	}
	last := m.state.History[len(m.state.History)-1]
	m.pushSnapshotLocked(i18n.T("pre-rollback"))
	m.state.Modules = cloneModules(last.Modules)
	return m.saveLocked()
}

func (m *StateManager) Reconcile() ([]StateCorrection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var corrections []StateCorrection
	for name, moduleState := range m.state.Modules {
		if !isValidModuleState(moduleState.State) {
			corrections = append(corrections, StateCorrection{
				ModuleName: name,
				Previous:   moduleState.State,
				Corrected:  module.StateFailed,
			})
			moduleState.State = module.StateFailed
			moduleState.UpdatedAt = time.Now().UTC()
			m.state.Modules[name] = moduleState
		}
		if moduleState.Version == "" {
			moduleState.Version = "unknown"
			moduleState.UpdatedAt = time.Now().UTC()
			m.state.Modules[name] = moduleState
		}
	}

	if len(corrections) > 0 {
		m.pushSnapshotLocked(i18n.T("reconcile"))
		if err := m.saveLocked(); err != nil {
			return nil, err
		}
	}

	return corrections, nil
}

func (m *StateManager) pushSnapshotLocked(reason string) {
	snapshot := StateSnapshot{
		ID:        fmt.Sprintf("%d", time.Now().UTC().UnixNano()),
		Reason:    reason,
		CreatedAt: time.Now().UTC(),
		Modules:   cloneModules(m.state.Modules),
	}
	m.state.History = append(m.state.History, snapshot)
	if len(m.state.History) > maxHistoryEntries {
		m.state.History = m.state.History[len(m.state.History)-maxHistoryEntries:]
	}
}

func (s SystemState) clone() SystemState {
	clone := SystemState{
		SchemaVersion: s.SchemaVersion,
		UpdatedAt:     s.UpdatedAt,
		Modules:       cloneModules(s.Modules),
		History:       make([]StateSnapshot, len(s.History)),
	}
	copy(clone.History, s.History)
	for i, snapshot := range clone.History {
		snapshot.Modules = cloneModules(snapshot.Modules)
		clone.History[i] = snapshot
	}
	return clone
}

func cloneModules(source map[string]ModuleState) map[string]ModuleState {
	clone := make(map[string]ModuleState, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func isValidModuleState(state module.State) bool {
	switch state {
	case module.StateAvailable,
		module.StateResolved,
		module.StateInstalled,
		module.StateRunning,
		module.StateStopped,
		module.StateFailed,
		module.StateDeprecated:
		return true
	default:
		return false
	}
}
