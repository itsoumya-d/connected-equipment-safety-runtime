package safety

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNormalOperation(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	now := time.Now()

	sample := TelemetrySample{
		EquipmentID:     "EQ-101",
		Timestamp:       now,
		PitchDegrees:    2.5,
		RollDegrees:     1.8,
		GroundSpeedKmh:  15.0,
		EngineRPM:       2100,
		OperatorPresent: true,
		HydraulicPTO:    true,
		InSafetyZone:    true,
	}

	incidents := engine.ProcessSample(sample)
	if len(incidents) != 0 {
		t.Fatalf("expected 0 incidents under normal conditions, got %d", len(incidents))
	}
}

func TestTiltDebounceAndIncident(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DebounceThreshold = 2
	engine := NewEngine(cfg)
	now := time.Now()

	// Sample 1: Transient spike - below debounce threshold
	sample1 := TelemetrySample{
		EquipmentID:     "EQ-102",
		Timestamp:       now,
		RollDegrees:     28.5, // exceeds 25.0
		OperatorPresent: true,
	}
	incidents1 := engine.ProcessSample(sample1)
	if len(incidents1) != 0 {
		t.Fatalf("expected 0 incidents on first spike due to debouncing, got %d", len(incidents1))
	}

	// Sample 2: Consecutive breach - reaches debounce threshold
	sample2 := TelemetrySample{
		EquipmentID:     "EQ-102",
		Timestamp:       now.Add(100 * time.Millisecond),
		RollDegrees:     29.0,
		OperatorPresent: true,
	}
	incidents2 := engine.ProcessSample(sample2)
	if len(incidents2) != 1 {
		t.Fatalf("expected 1 incident after debounce confirmation, got %d", len(incidents2))
	}
	if incidents2[0].Severity != SeverityCritical {
		t.Fatalf("expected SeverityCritical, got %s", incidents2[0].Severity)
	}
	if incidents2[0].MitigationOrder != "EMERGENCY_ENGINE_SHUTDOWN" {
		t.Fatalf("expected EMERGENCY_ENGINE_SHUTDOWN mitigation, got %s", incidents2[0].MitigationOrder)
	}
}

func TestUnattendedPTOInterlock(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	now := time.Now()

	sample := TelemetrySample{
		EquipmentID:     "EQ-103",
		Timestamp:       now,
		OperatorPresent: false, // operator leaves seat
		HydraulicPTO:    true,  // PTO left running
		GroundSpeedKmh:  0.0,
	}

	incidents := engine.ProcessSample(sample)
	if len(incidents) != 1 {
		t.Fatalf("expected 1 incident for unattended PTO, got %d", len(incidents))
	}
	if incidents[0].Severity != SeverityHigh {
		t.Fatalf("expected SeverityHigh, got %s", incidents[0].Severity)
	}
	if incidents[0].RuleName != "UNATTENDED_OPERATOR_INTERLOCK" {
		t.Fatalf("expected UNATTENDED_OPERATOR_INTERLOCK rule, got %s", incidents[0].RuleName)
	}
}

func TestSafetyZoneOverspeed(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxSpeedKmh = 20.0
	cfg.DebounceThreshold = 1
	engine := NewEngine(cfg)

	sample := TelemetrySample{
		EquipmentID:     "EQ-104",
		Timestamp:       time.Now(),
		InSafetyZone:    true,
		GroundSpeedKmh:  35.0, // exceeds 20.0
		OperatorPresent: true,
	}

	incidents := engine.ProcessSample(sample)
	if len(incidents) != 1 {
		t.Fatalf("expected 1 overspeed incident, got %d", len(incidents))
	}
	if incidents[0].Severity != SeverityMedium {
		t.Fatalf("expected SeverityMedium, got %s", incidents[0].Severity)
	}
	if incidents[0].MitigationOrder != "SPEED_GOVERNOR_LIMIT" {
		t.Fatalf("expected SPEED_GOVERNOR_LIMIT, got %s", incidents[0].MitigationOrder)
	}
}

func TestConcurrentTelemetryStream(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	var wg sync.WaitGroup
	workers := 50
	samplesPerWorker := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			eqID := fmt.Sprintf("EQ-WORKER-%d", workerID)
			for j := 0; j < samplesPerWorker; j++ {
				_ = engine.ProcessSample(TelemetrySample{
					EquipmentID:     eqID,
					Timestamp:       time.Now(),
					PitchDegrees:    1.0,
					RollDegrees:     1.0,
					GroundSpeedKmh:  10.0,
					OperatorPresent: true,
				})
			}
		}(i)
	}
	wg.Wait()
}

func BenchmarkProcessSample(b *testing.B) {
	engine := NewEngine(DefaultConfig())
	sample := TelemetrySample{
		EquipmentID:     "EQ-BENCH-1",
		Timestamp:       time.Now(),
		PitchDegrees:    5.0,
		RollDegrees:     6.0,
		GroundSpeedKmh:  12.0,
		OperatorPresent: true,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = engine.ProcessSample(sample)
	}
}
