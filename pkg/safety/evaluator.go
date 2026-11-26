package safety

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type equipmentState struct {
	consecutiveTiltBreaches  int
	consecutiveSpeedBreaches int
	lastIncidentTime         time.Time
}

// Engine evaluates streaming equipment telemetry against configurable safety risk envelopes.
type Engine struct {
	mu     sync.RWMutex
	cfg    Config
	states map[string]*equipmentState
}

// NewEngine initializes a safety stream processor engine.
func NewEngine(cfg Config) *Engine {
	return &Engine{
		cfg:    cfg,
		states: make(map[string]*equipmentState),
	}
}

// ProcessSample evaluates a single telemetry snapshot and returns any detected safety incidents.
func (e *Engine) ProcessSample(sample TelemetrySample) []SafetyIncident {
	e.mu.Lock()
	defer e.mu.Unlock()

	st, exists := e.states[sample.EquipmentID]
	if !exists {
		st = &equipmentState{}
		e.states[sample.EquipmentID] = st
	}

	var incidents []SafetyIncident

	// Rule 1: Extreme Roll or Pitch (Rollover Hazard) with debouncing
	isTiltBreached := math.Abs(sample.RollDegrees) > e.cfg.MaxRollDegrees || math.Abs(sample.PitchDegrees) > e.cfg.MaxPitchDegrees
	if isTiltBreached {
		st.consecutiveTiltBreaches++
		if st.consecutiveTiltBreaches >= e.cfg.DebounceThreshold {
			incidents = append(incidents, SafetyIncident{
				IncidentID:      fmt.Sprintf("INC-TILT-%s-%d", sample.EquipmentID, sample.Timestamp.UnixNano()),
				EquipmentID:     sample.EquipmentID,
				Timestamp:       sample.Timestamp,
				Severity:        SeverityCritical,
				RuleName:        "ROLLOVER_RISK_ENVELOPE",
				Description:     fmt.Sprintf("Equipment tilt breached limit (Roll: %.1f deg, Pitch: %.1f deg)", sample.RollDegrees, sample.PitchDegrees),
				MitigationOrder: "EMERGENCY_ENGINE_SHUTDOWN",
			})
		}
	} else {
		st.consecutiveTiltBreaches = 0
	}

	// Rule 2: Unattended Active Equipment / Operator Interlock Bypass
	if !sample.OperatorPresent && (sample.HydraulicPTO || sample.GroundSpeedKmh > 1.0) {
		desc := "Operator seat switch open while hydraulic PTO active"
		if sample.GroundSpeedKmh > 1.0 {
			desc = fmt.Sprintf("Operator seat switch open while equipment moving at %.1f km/h", sample.GroundSpeedKmh)
		}
		incidents = append(incidents, SafetyIncident{
			IncidentID:      fmt.Sprintf("INC-INTERLOCK-%s-%d", sample.EquipmentID, sample.Timestamp.UnixNano()),
			EquipmentID:     sample.EquipmentID,
			Timestamp:       sample.Timestamp,
			Severity:        SeverityHigh,
			RuleName:        "UNATTENDED_OPERATOR_INTERLOCK",
			Description:     desc,
			MitigationOrder: "INTERLOCK_HYDRAULIC_DISENGAGE",
		})
	}

	// Rule 3: Restricted Safety Zone Speed Limit Violation
	if sample.InSafetyZone && sample.GroundSpeedKmh > e.cfg.MaxSpeedKmh {
		st.consecutiveSpeedBreaches++
		if st.consecutiveSpeedBreaches >= e.cfg.DebounceThreshold {
			incidents = append(incidents, SafetyIncident{
				IncidentID:      fmt.Sprintf("INC-OVERSPEED-%s-%d", sample.EquipmentID, sample.Timestamp.UnixNano()),
				EquipmentID:     sample.EquipmentID,
				Timestamp:       sample.Timestamp,
				Severity:        SeverityMedium,
				RuleName:        "SAFETY_ZONE_SPEED_VIOLATION",
				Description:     fmt.Sprintf("Ground speed %.1f km/h exceeds safety zone limit %.1f km/h", sample.GroundSpeedKmh, e.cfg.MaxSpeedKmh),
				MitigationOrder: "SPEED_GOVERNOR_LIMIT",
			})
		}
	} else {
		st.consecutiveSpeedBreaches = 0
	}

	if len(incidents) > 0 {
		st.lastIncidentTime = sample.Timestamp
	}

	return incidents
}

// Reset clears tracked state for a specific equipment unit.
func (e *Engine) Reset(equipmentID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.states, equipmentID)
}
