package config

import (
	"slices"
)

// NameResolutionMode consolidates the deprecated fixes.nameResolutionDec2023
// and fixes.nameResolutionFeb2025 booleans into a single setting.
type NameResolutionMode string

const (
	// NameResolutionLegacy: pre-December 2023 behavior; first registered type keeps a conflicting name.
	NameResolutionLegacy NameResolutionMode = "legacy"
	// NameResolutionOrdered (formerly fixes.nameResolutionDec2023): conflicting types renamed by fixed precedence order.
	NameResolutionOrdered NameResolutionMode = "ordered"
	// NameResolutionShortest (formerly fixes.nameResolutionFeb2025): conflicting types renamed with the smallest distinguishing set of context labels.
	NameResolutionShortest NameResolutionMode = "shortest"
	// NameResolutionQualified: unnamed inline schemas are named after their property path from the
	// nearest enclosing named schema (e.g. Order.status -> OrderStatus), even without a conflict.
	NameResolutionQualified NameResolutionMode = "qualified"
)

// rank is the mode's position in NameResolutionModes; -1 for unknown modes.
func (m NameResolutionMode) rank() int {
	return slices.Index(NameResolutionModes(), m)
}

// AtLeast reports whether m is mode or newer.
func (m NameResolutionMode) AtLeast(mode NameResolutionMode) bool {
	return m.rank() >= mode.rank()
}

// NameResolutionModes lists the supported modes, oldest first.
func NameResolutionModes() []NameResolutionMode {
	return []NameResolutionMode{NameResolutionLegacy, NameResolutionOrdered, NameResolutionShortest, NameResolutionQualified}
}

func (m NameResolutionMode) IsValid() bool {
	return m.rank() >= 0
}

func nameResolutionFromFixes(f *Fixes) NameResolutionMode {
	switch {
	case f != nil && f.NameResolutionFeb2025:
		return NameResolutionShortest
	case f != nil && f.NameResolutionDec2023:
		return NameResolutionOrdered
	default:
		return NameResolutionLegacy
	}
}

func (m NameResolutionMode) applyToFixes(f *Fixes) {
	f.NameResolutionDec2023 = m.rank() >= NameResolutionOrdered.rank()
	f.NameResolutionFeb2025 = m.rank() >= NameResolutionShortest.rank()
}

// GetNameResolution returns the effective mode
func (g *Generation) GetNameResolution() NameResolutionMode {
	switch {
	case g.NameResolution != "":
		return g.NameResolution
	case g.Fixes != nil:
		return nameResolutionFromFixes(g.Fixes)
	default:
		return NameResolutionShortest
	}
}

// SyncNameResolution aligns nameResolution and the deprecated fixes booleans
// so persisted configs stay readable by older tooling. Call before marshalling.
// Explicit mode wins; legacy configs without one are left untouched.
func (g *Generation) SyncNameResolution() {
	mode := g.NameResolution
	if mode == "" {
		mode = nameResolutionFromFixes(g.Fixes)
		if mode == NameResolutionLegacy {
			return
		}
		g.NameResolution = mode
	}

	if g.Fixes == nil {
		if mode == NameResolutionLegacy {
			return
		}
		g.Fixes = &Fixes{}
	}
	mode.applyToFixes(g.Fixes)
}

