package safety

import "time"

// Severity represents safety incident severity levels.
type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

// TelemetrySample captures a single telemetry snapshot from a connected vehicle or equipment.
type TelemetrySample struct {
	EquipmentID     string    `json:"equipment_id"`
	Timestamp       time.Time `json:"timestamp"`
	PitchDegrees    float64   `json:"pitch_degrees"`
	RollDegrees     float64   `json:"roll_degrees"`
	GroundSpeedKmh  float64   `json:"ground_speed_kmh"`
	EngineRPM       int       `json:"engine_rpm"`
	OperatorPresent bool      `json:"operator_present"`
	HydraulicPTO    bool      `json:"hydraulic_pto"`
	InSafetyZone    bool      `json:"in_safety_zone"`
}

// SafetyIncident captures an identified safety violation or risk condition.
type SafetyIncident struct {
	IncidentID      string    `json:"incident_id"`
	EquipmentID     string    `json:"equipment_id"`
	Timestamp       time.Time `json:"timestamp"`
	Severity        Severity  `json:"severity"`
	RuleName        string    `json:"rule_name"`
	Description     string    `json:"description"`
	MitigationOrder string    `json:"mitigation_order"`
}

// Config defines threshold bounds for safety envelope evaluation.
type Config struct {
	MaxRollDegrees    float64
	MaxPitchDegrees   float64
	MaxSpeedKmh       float64
	DebounceThreshold int
}

// DefaultConfig provides standard conservative thresholds for industrial equipment.
func DefaultConfig() Config {
	return Config{
		MaxRollDegrees:    25.0,
		MaxPitchDegrees:   20.0,
		MaxSpeedKmh:       30.0,
		DebounceThreshold: 2,
	}
}
