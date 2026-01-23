package module

var allowedTransitions = map[State]map[State]struct{}{
	StateAvailable: {
		StateResolved:   {},
		StateDeprecated: {},
	},
	StateResolved: {
		StateInstalled:  {},
		StateFailed:     {},
		StateDeprecated: {},
	},
	StateInstalled: {
		StateRunning:    {},
		StateStopped:    {},
		StateFailed:     {},
		StateDeprecated: {},
	},
	StateRunning: {
		StateStopped:    {},
		StateFailed:     {},
		StateDeprecated: {},
	},
	StateStopped: {
		StateRunning:    {},
		StateFailed:     {},
		StateDeprecated: {},
	},
	StateFailed: {
		StateResolved:   {},
		StateDeprecated: {},
	},
	StateDeprecated: {},
}

func CanTransition(from State, to State) bool {
	if from == to {
		return true
	}
	allowed, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	_, ok = allowed[to]
	return ok
}
