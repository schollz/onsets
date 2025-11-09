package onset

import (
	"testing"
)

func TestAnalyzeSlices(t *testing.T) {
	wavFile := "amen.wav"

	t.Run("FindBestNSlices", func(t *testing.T) {
		options := SliceAnalyzerOptions{
			NumSlices:        8,
			Optimize:         true,
			OptimizeWindowMs: 100.0,
		}

		result, err := AnalyzeSlices(wavFile, options)
		if err != nil {
			t.Fatalf("AnalyzeSlices failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Samples) == 0 {
			t.Error("Expected samples, got empty array")
		}

		if result.SampleRate == 0 {
			t.Error("Expected non-zero sample rate")
		}

		if len(result.Onsets) == 0 {
			t.Error("Expected onsets, got empty array")
		}

		// Should find approximately the requested number of slices
		// (may be less if not enough onsets detected)
		if len(result.Onsets) > options.NumSlices {
			t.Errorf("Expected at most %d onsets, got %d", options.NumSlices, len(result.Onsets))
		}

		// Verify onsets are in chronological order
		for i := 1; i < len(result.Onsets); i++ {
			if result.Onsets[i] <= result.Onsets[i-1] {
				t.Errorf("Onsets not in chronological order at index %d: %f <= %f",
					i, result.Onsets[i], result.Onsets[i-1])
			}
		}
	})

	t.Run("FindAllSlices", func(t *testing.T) {
		options := SliceAnalyzerOptions{
			NumSlices:        0, // 0 means all slices
			Optimize:         false,
			OptimizeWindowMs: 100.0,
		}

		result, err := AnalyzeSlices(wavFile, options)
		if err != nil {
			t.Fatalf("AnalyzeSlices failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Onsets) == 0 {
			t.Error("Expected onsets, got empty array")
		}

		// When finding all slices, should typically find more than a specific small number
		if len(result.Onsets) < 5 {
			t.Errorf("Expected more onsets when finding all, got only %d", len(result.Onsets))
		}

		// Verify onsets are in chronological order
		for i := 1; i < len(result.Onsets); i++ {
			if result.Onsets[i] <= result.Onsets[i-1] {
				t.Errorf("Onsets not in chronological order at index %d: %f <= %f",
					i, result.Onsets[i], result.Onsets[i-1])
			}
		}
	})

	t.Run("WithOptimization", func(t *testing.T) {
		options := SliceAnalyzerOptions{
			NumSlices:        4,
			Optimize:         true,
			OptimizeWindowMs: 50.0, // Smaller window
		}

		result, err := AnalyzeSlices(wavFile, options)
		if err != nil {
			t.Fatalf("AnalyzeSlices failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Onsets) == 0 {
			t.Error("Expected onsets, got empty array")
		}
	})

	t.Run("WithoutOptimization", func(t *testing.T) {
		options := SliceAnalyzerOptions{
			NumSlices:        4,
			Optimize:         false,
			OptimizeWindowMs: 100.0, // Should be ignored when Optimize is false
		}

		result, err := AnalyzeSlices(wavFile, options)
		if err != nil {
			t.Fatalf("AnalyzeSlices failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Onsets) == 0 {
			t.Error("Expected onsets, got empty array")
		}
	})

	t.Run("InvalidFile", func(t *testing.T) {
		options := DefaultSliceAnalyzerOptions()

		_, err := AnalyzeSlices("nonexistent.wav", options)
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})
}

func TestDefaultSliceAnalyzerOptions(t *testing.T) {
	opts := DefaultSliceAnalyzerOptions()

	if opts.NumSlices != 0 {
		t.Errorf("Expected NumSlices to be 0, got %d", opts.NumSlices)
	}

	if !opts.Optimize {
		t.Error("Expected Optimize to be true")
	}

	if opts.OptimizeWindowMs != 100.0 {
		t.Errorf("Expected OptimizeWindowMs to be 100.0, got %f", opts.OptimizeWindowMs)
	}

	if opts.Method != "hfc" {
		t.Errorf("Expected Method to be 'hfc', got %s", opts.Method)
	}
}

func TestSliceAnalyzerResult(t *testing.T) {
	// Test that the result structure can be created and accessed
	result := &SliceAnalyzerResult{
		Onsets:     []float64{0.1, 0.5, 1.0},
		Samples:    []float64{0.0, 0.1, -0.1},
		SampleRate: 44100,
	}

	if len(result.Onsets) != 3 {
		t.Errorf("Expected 3 onsets, got %d", len(result.Onsets))
	}

	if len(result.Samples) != 3 {
		t.Errorf("Expected 3 samples, got %d", len(result.Samples))
	}

	if result.SampleRate != 44100 {
		t.Errorf("Expected sample rate 44100, got %d", result.SampleRate)
	}
}

func TestAnalyzeSlicesWithDifferentMethods(t *testing.T) {
	wavFile := "amen.wav"

	methods := []string{"energy", "hfc", "complex", "phase", "wphase", "specdiff", "kl", "mkl", "specflux"}

	for _, method := range methods {
		t.Run("Method_"+method, func(t *testing.T) {
			options := SliceAnalyzerOptions{
				NumSlices:        0,
				Optimize:         false,
				OptimizeWindowMs: 100.0,
				Method:           method,
			}

			result, err := AnalyzeSlices(wavFile, options)
			if err != nil {
				t.Fatalf("AnalyzeSlices failed for method %s: %v", method, err)
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			if len(result.Onsets) == 0 {
				t.Errorf("Expected onsets for method %s, got empty array", method)
			}

			// Verify onsets are in chronological order
			for i := 1; i < len(result.Onsets); i++ {
				if result.Onsets[i] <= result.Onsets[i-1] {
					t.Errorf("Onsets not in chronological order at index %d for method %s: %f <= %f",
						i, method, result.Onsets[i], result.Onsets[i-1])
				}
			}
		})
	}
}

func TestAnalyzeSlicesWithConsensusMethod(t *testing.T) {
	wavFile := "amen.wav"

	t.Run("ConsensusAllOnsets", func(t *testing.T) {
		options := SliceAnalyzerOptions{
			NumSlices:        0, // Get all onsets
			Optimize:         false,
			OptimizeWindowMs: 100.0,
			Method:           "consensus",
		}

		result, err := AnalyzeSlices(wavFile, options)
		if err != nil {
			t.Fatalf("AnalyzeSlices failed for consensus method: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Onsets) == 0 {
			t.Error("Expected consensus onsets, got empty array")
		}

		// Verify onsets are in chronological order
		for i := 1; i < len(result.Onsets); i++ {
			if result.Onsets[i] <= result.Onsets[i-1] {
				t.Errorf("Onsets not in chronological order at index %d: %f <= %f",
					i, result.Onsets[i], result.Onsets[i-1])
			}
		}

		t.Logf("Consensus method detected %d onsets", len(result.Onsets))
	})

	t.Run("ConsensusBestN", func(t *testing.T) {
		options := SliceAnalyzerOptions{
			NumSlices:        8, // Get best 8 onsets
			Optimize:         true,
			OptimizeWindowMs: 100.0,
			Method:           "consensus",
		}

		result, err := AnalyzeSlices(wavFile, options)
		if err != nil {
			t.Fatalf("AnalyzeSlices failed for consensus method: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Onsets) == 0 {
			t.Error("Expected consensus onsets, got empty array")
		}

		if len(result.Onsets) > 8 {
			t.Errorf("Expected at most 8 onsets, got %d", len(result.Onsets))
		}

		// Verify onsets are in chronological order
		for i := 1; i < len(result.Onsets); i++ {
			if result.Onsets[i] <= result.Onsets[i-1] {
				t.Errorf("Onsets not in chronological order at index %d: %f <= %f",
					i, result.Onsets[i], result.Onsets[i-1])
			}
		}

		t.Logf("Consensus method with NumSlices=8 detected %d onsets", len(result.Onsets))
	})
}

func TestAnalyzeSlicesMethodComparison(t *testing.T) {
	wavFile := "amen.wav"

	// Test that different methods can produce different results
	hfcOptions := SliceAnalyzerOptions{
		NumSlices:        8,
		Optimize:         false,
		OptimizeWindowMs: 100.0,
		Method:           "hfc",
	}

	consensusOptions := SliceAnalyzerOptions{
		NumSlices:        8,
		Optimize:         false,
		OptimizeWindowMs: 100.0,
		Method:           "consensus",
	}

	hfcResult, err := AnalyzeSlices(wavFile, hfcOptions)
	if err != nil {
		t.Fatalf("AnalyzeSlices failed for hfc: %v", err)
	}

	consensusResult, err := AnalyzeSlices(wavFile, consensusOptions)
	if err != nil {
		t.Fatalf("AnalyzeSlices failed for consensus: %v", err)
	}

	// Both should detect onsets
	if len(hfcResult.Onsets) == 0 {
		t.Error("HFC method detected no onsets")
	}

	if len(consensusResult.Onsets) == 0 {
		t.Error("Consensus method detected no onsets")
	}

	t.Logf("HFC detected %d onsets, Consensus detected %d onsets",
		len(hfcResult.Onsets), len(consensusResult.Onsets))
}

func TestMinimumSpacing(t *testing.T) {
	wavFile := "amen.wav"

	t.Run("WithMinimumSpacing", func(t *testing.T) {
		// First, get all onsets without minimum spacing
		optionsWithout := SliceAnalyzerOptions{
			Method:            "hfc",
			UseMinimumSpacing: false,
			Optimize:          false, // Disable optimization for clearer results
		}

		resultWithout, err := AnalyzeSlices(wavFile, optionsWithout)
		if err != nil {
			t.Fatalf("AnalyzeSlices failed: %v", err)
		}

		// Now get onsets with minimum spacing of 80ms
		optionsWith := SliceAnalyzerOptions{
			Method:            "hfc",
			UseMinimumSpacing: true,
			MinimumSpacing:    80.0, // 80ms
			Optimize:          false, // Disable optimization for clearer results
		}

		resultWith, err := AnalyzeSlices(wavFile, optionsWith)
		if err != nil {
			t.Fatalf("AnalyzeSlices failed: %v", err)
		}

		t.Logf("Without minimum spacing: %d onsets", len(resultWithout.Onsets))
		t.Logf("With 80ms minimum spacing: %d onsets", len(resultWith.Onsets))

		// With minimum spacing should have fewer or equal onsets
		if len(resultWith.Onsets) > len(resultWithout.Onsets) {
			t.Errorf("Expected minimum spacing to reduce or maintain onset count, got %d onsets (was %d)",
				len(resultWith.Onsets), len(resultWithout.Onsets))
		}

		// Verify that all onsets in the filtered result have at least 80ms spacing
		minimumSpacingSec := 0.080 // 80ms in seconds
		for i := 1; i < len(resultWith.Onsets); i++ {
			spacing := resultWith.Onsets[i] - resultWith.Onsets[i-1]
			if spacing < minimumSpacingSec-0.001 { // Allow small floating point error
				t.Errorf("Onset at index %d has spacing of %.4fs (%.1fms), expected at least %.4fs (%.1fms)",
					i, spacing, spacing*1000, minimumSpacingSec, minimumSpacingSec*1000)
			}
		}
	})

	t.Run("WithDifferentSpacingValues", func(t *testing.T) {
		// Test with a very small spacing (should keep most onsets)
		optionsSmall := SliceAnalyzerOptions{
			Method:            "hfc",
			UseMinimumSpacing: true,
			MinimumSpacing:    10.0, // 10ms
			Optimize:          false,
		}

		resultSmall, err := AnalyzeSlices(wavFile, optionsSmall)
		if err != nil {
			t.Fatalf("AnalyzeSlices failed: %v", err)
		}

		// Test with a large spacing (should filter many onsets)
		optionsLarge := SliceAnalyzerOptions{
			Method:            "hfc",
			UseMinimumSpacing: true,
			MinimumSpacing:    200.0, // 200ms
			Optimize:          false,
		}

		resultLarge, err := AnalyzeSlices(wavFile, optionsLarge)
		if err != nil {
			t.Fatalf("AnalyzeSlices failed: %v", err)
		}

		t.Logf("With 10ms spacing: %d onsets", len(resultSmall.Onsets))
		t.Logf("With 200ms spacing: %d onsets", len(resultLarge.Onsets))

		// Larger spacing should result in fewer onsets
		if len(resultLarge.Onsets) > len(resultSmall.Onsets) {
			t.Errorf("Expected larger spacing to have fewer onsets, got %d (small spacing had %d)",
				len(resultLarge.Onsets), len(resultSmall.Onsets))
		}

		// Verify spacing constraint for large spacing
		minimumSpacingSec := 0.200 // 200ms in seconds
		for i := 1; i < len(resultLarge.Onsets); i++ {
			spacing := resultLarge.Onsets[i] - resultLarge.Onsets[i-1]
			if spacing < minimumSpacingSec-0.001 { // Allow small floating point error
				t.Errorf("Onset at index %d has spacing of %.4fs (%.1fms), expected at least %.4fs (%.1fms)",
					i, spacing, spacing*1000, minimumSpacingSec, minimumSpacingSec*1000)
			}
		}
	})

	t.Run("MinimumSpacingDisabled", func(t *testing.T) {
		// Verify that UseMinimumSpacing=false doesn't filter
		options := SliceAnalyzerOptions{
			Method:            "hfc",
			UseMinimumSpacing: false,
			MinimumSpacing:    80.0, // This should be ignored
			Optimize:          false,
		}

		result, err := AnalyzeSlices(wavFile, options)
		if err != nil {
			t.Fatalf("AnalyzeSlices failed: %v", err)
		}

		// Count how many consecutive onsets are closer than 80ms
		closePairs := 0
		minimumSpacingSec := 0.080
		for i := 1; i < len(result.Onsets); i++ {
			spacing := result.Onsets[i] - result.Onsets[i-1]
			if spacing < minimumSpacingSec {
				closePairs++
			}
		}

		// When minimum spacing is disabled, we expect some onsets to be closer than 80ms
		t.Logf("Found %d pairs of onsets closer than 80ms (minimum spacing disabled)", closePairs)
		if closePairs == 0 && len(result.Onsets) > 3 {
			t.Log("Warning: No onsets closer than 80ms found, but minimum spacing was disabled")
		}
	})
}
