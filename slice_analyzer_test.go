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
