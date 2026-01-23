package module

import "fmt"

type InstallPlan struct {
	Order   []string
	Modules map[string]ModuleRecord
}

type Resolver struct {
	registry *Registry
}

func NewResolver(registry *Registry) *Resolver {
	return &Resolver{registry: registry}
}

func (r *Resolver) ResolveInstallPlan(targets []string) (InstallPlan, error) {
	if r.registry == nil {
		return InstallPlan{}, fmt.Errorf("registry is required")
	}
	plan := InstallPlan{
		Modules: make(map[string]ModuleRecord),
	}
	visited := make(map[string]bool)
	var stack []string

	for _, target := range targets {
		if target == "" {
			continue
		}
		name, constraint, err := ParseModuleDependency(target)
		if err != nil {
			return InstallPlan{}, err
		}
		if err := r.resolveModule(name, constraint, visited, &stack, &plan); err != nil {
			return InstallPlan{}, err
		}
	}

	plan.Order = append([]string(nil), stack...)
	return plan, nil
}

func (r *Resolver) resolveModule(name string, constraint *VersionConstraint, visited map[string]bool, stack *[]string, plan *InstallPlan) error {
	if existing, ok := plan.Modules[name]; ok {
		if constraint != nil && !constraint.Match(existing.Version) {
			return fmt.Errorf("version conflict for %s: selected %s does not satisfy %s%s", name, existing.Version, constraint.Operator, constraint.Version)
		}
		return nil
	}
	if visited[name] {
		return fmt.Errorf("circular dependency detected at %s", name)
	}
	visited[name] = true

	record, err := r.selectRecord(name, constraint)
	if err != nil {
		return err
	}
	for _, dep := range record.Manifest.Dependencies.Modules {
		depName, depConstraint, err := ParseModuleDependency(dep)
		if err != nil {
			return err
		}
		if err := r.resolveModule(depName, depConstraint, visited, stack, plan); err != nil {
			return err
		}
	}

	plan.Modules[name] = record
	*stack = append(*stack, name)
	visited[name] = false
	return nil
}

func (r *Resolver) selectRecord(name string, constraint *VersionConstraint) (ModuleRecord, error) {
	records := r.registry.Get(name)
	if len(records) == 0 {
		return ModuleRecord{}, fmt.Errorf("module %s not found in registry", name)
	}
	for _, record := range records {
		if constraint == nil || constraint.Match(record.Version) {
			return record, nil
		}
	}
	return ModuleRecord{}, fmt.Errorf("no available versions for %s satisfy constraint", name)
}
