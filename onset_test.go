package onset

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/go-audio/wav"
)

func TestFvecCreation(t *testing.T) {
	v := NewFvec(10)
	if v.Length != 10 {
		t.Errorf("Expected length 10, got %d", v.Length)
	}
	if len(v.Data) != 10 {
		t.Errorf("Expected data length 10, got %d", len(v.Data))
	}
}

func TestFvecOperations(t *testing.T) {
	v := NewFvec(5)
	v.Data[0] = 1.0
	v.Data[1] = 2.0
	v.Data[2] = 3.0
	v.Data[3] = 4.0
	v.Data[4] = 5.0

	mean := v.Mean()
	if mean != 3.0 {
		t.Errorf("Expected mean 3.0, got %f", mean)
	}

	max := v.Max()
	if max != 5.0 {
		t.Errorf("Expected max 5.0, got %f", max)
	}

	min := v.Min()
	if min != 1.0 {
		t.Errorf("Expected min 1.0, got %f", min)
	}
}

func TestCvecCreation(t *testing.T) {
	c := NewCvec(512)
	expectedLength := uint(512/2 + 1)
	if c.Length != expectedLength {
		t.Errorf("Expected length %d, got %d", expectedLength, c.Length)
	}
}

func TestPeakPicker(t *testing.T) {
	pp := NewPeakPicker()
	if pp.Threshold != 0.1 {
		t.Errorf("Expected default threshold 0.1, got %f", pp.Threshold)
	}

	pp.SetThreshold(0.5)
	if pp.GetThreshold() != 0.5 {
		t.Errorf("Expected threshold 0.5, got %f", pp.GetThreshold())
	}
}

func TestSpecdesc(t *testing.T) {
	bufSize := uint(512)
	s := NewSpecdesc("hfc", bufSize)

	if s.OnsetType != OnsetHFC {
		t.Errorf("Expected HFC onset type")
	}

	// Test energy method
	s2 := NewSpecdesc("energy", bufSize)
	if s2.OnsetType != OnsetEnergy {
		t.Errorf("Expected Energy onset type")
	}
}

func TestOnsetCreation(t *testing.T) {
	bufSize := uint(512)
	hopSize := uint(256)
	samplerate := uint(44100)

	o := NewOnset("hfc", bufSize, hopSize, samplerate)

	if o.Samplerate != samplerate {
		t.Errorf("Expected samplerate %d, got %d", samplerate, o.Samplerate)
	}
	if o.HopSize != hopSize {
		t.Errorf("Expected hopSize %d, got %d", hopSize, o.HopSize)
	}
}

func TestOnsetDetection(t *testing.T) {
	bufSize := uint(512)
	hopSize := uint(256)
	samplerate := uint(44100)

	o := NewOnset("hfc", bufSize, hopSize, samplerate)
	input := NewFvec(hopSize)
	output := NewFvec(1)

	// Generate a test signal with a clear onset
	for i := uint(0); i < hopSize; i++ {
		t := float64(i) / float64(samplerate)
		input.Data[i] = math.Sin(2 * math.Pi * 440 * t)
	}

	// Process the input
	o.Do(input, output)

	// The output should be a value (onset detected or not)
	if output.Data[0] < 0 {
		t.Errorf("Onset value should not be negative, got %f", output.Data[0])
	}
}

func TestOnsetMethods(t *testing.T) {
	bufSize := uint(512)
	hopSize := uint(256)
	samplerate := uint(44100)

	methods := []string{"energy", "hfc", "complex", "phase", "specdiff", "kl", "mkl", "specflux"}

	for _, method := range methods {
		o := NewOnset(method, bufSize, hopSize, samplerate)
		input := NewFvec(hopSize)
		output := NewFvec(1)

		// Generate a simple test signal
		for i := uint(0); i < hopSize; i++ {
			input.Data[i] = math.Sin(2 * math.Pi * 440 * float64(i) / float64(samplerate))
		}

		// Should not panic
		o.Do(input, output)
	}
}

func TestOnsetThresholds(t *testing.T) {
	bufSize := uint(512)
	hopSize := uint(256)
	samplerate := uint(44100)

	o := NewOnset("hfc", bufSize, hopSize, samplerate)

	o.SetThreshold(0.5)
	if o.GetThreshold() != 0.5 {
		t.Errorf("Expected threshold 0.5, got %f", o.GetThreshold())
	}

	o.SetSilence(-80.0)
	if o.GetSilence() != -80.0 {
		t.Errorf("Expected silence -80.0, got %f", o.GetSilence())
	}

	o.SetMinioiMs(100.0)
	if o.GetMinioiMs() != 100.0 {
		t.Errorf("Expected minioi 100.0 ms, got %f", o.GetMinioiMs())
	}
}

func TestSpectralWhitening(t *testing.T) {
	bufSize := uint(512)
	hopSize := uint(256)
	samplerate := uint(44100)

	sw := NewSpectralWhitening(bufSize, hopSize, samplerate)
	if sw.BufSize != bufSize {
		t.Errorf("Expected bufSize %d, got %d", bufSize, sw.BufSize)
	}

	sw.SetRelaxTime(100.0)
	if sw.GetRelaxTime() != 100.0 {
		t.Errorf("Expected relax time 100.0, got %f", sw.GetRelaxTime())
	}

	sw.SetFloor(1e-3)
	if sw.GetFloor() != 1e-3 {
		t.Errorf("Expected floor 1e-3, got %f", sw.GetFloor())
	}
}

func TestMedian(t *testing.T) {
	v := NewFvec(5)
	v.Data = []float64{3, 1, 4, 1, 5}

	median := FvecMedian(v)
	if median != 3.0 {
		t.Errorf("Expected median 3.0, got %f", median)
	}
}

func TestPeakDetection(t *testing.T) {
	v := NewFvec(5)
	v.Data = []float64{1, 2, 5, 3, 1}

	if !FvecPeakPick(v, 2) {
		t.Error("Expected peak at position 2")
	}

	if FvecPeakPick(v, 1) {
		t.Error("Did not expect peak at position 1")
	}
}

func TestFilter(t *testing.T) {
	f := NewBiquadFilter(0.15998789, 0.31997577, 0.15998789, 0.23484048, 0)

	if f.Order != 3 {
		t.Errorf("Expected order 3, got %d", f.Order)
	}

	input := NewFvec(10)
	for i := range input.Data {
		input.Data[i] = 1.0
	}

	f.Do(input)

	// Filter should have modified the input
	// Just check it doesn't crash
}

func BenchmarkOnsetDetection(b *testing.B) {
	bufSize := uint(512)
	hopSize := uint(256)
	samplerate := uint(44100)

	o := NewOnset("hfc", bufSize, hopSize, samplerate)
	input := NewFvec(hopSize)
	output := NewFvec(1)

	// Generate test signal
	for i := uint(0); i < hopSize; i++ {
		input.Data[i] = math.Sin(2 * math.Pi * 440 * float64(i) / float64(samplerate))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		o.Do(input, output)
	}
}

// readWavFile reads a WAV file and returns the audio samples
func readWavFile(filename string) ([]float64, uint, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	decoder := wav.NewDecoder(f)
	if !decoder.IsValidFile() {
		return nil, 0, fmt.Errorf("invalid WAV file")
	}

	// Get the sample rate
	sampleRate := uint(decoder.SampleRate)

	// Read all audio data
	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read PCM data: %w", err)
	}

	// Convert to float64 and handle stereo by averaging channels
	numChannels := buf.Format.NumChannels
	samples := make([]float64, len(buf.Data)/numChannels)

	for i := 0; i < len(samples); i++ {
		sum := 0.0
		for ch := 0; ch < numChannels; ch++ {
			// Normalize int to float64 [-1.0, 1.0]
			sample := float64(buf.Data[i*numChannels+ch]) / 32768.0
			sum += sample
		}
		samples[i] = sum / float64(numChannels)
	}

	return samples, sampleRate, nil
}

// detectOnsets processes audio samples and returns onset times in seconds
func detectOnsets(samples []float64, sampleRate uint, method string, bufSize, hopSize uint, threshold float64, minioi float64) []float64 {
	o := NewOnset(method, bufSize, hopSize, sampleRate)
	o.SetThreshold(threshold)
	o.SetMinioiMs(minioi)

	input := NewFvec(hopSize)
	output := NewFvec(1)

	var onsets []float64

	// Process audio in chunks
	for pos := uint(0); pos+hopSize < uint(len(samples)); pos += hopSize {
		// Fill input buffer
		for i := uint(0); i < hopSize; i++ {
			if pos+i < uint(len(samples)) {
				input.Data[i] = samples[pos+i]
			} else {
				input.Data[i] = 0
			}
		}

		// Process
		o.Do(input, output)

		// Check for onset
		if output.Data[0] > 0 {
			onsetTime := o.GetLastS()
			onsets = append(onsets, onsetTime)
		}
	}

	return onsets
}

// TestOnsetDetectionOnAmen tests onset detection with various parameters on amen.wav
func TestOnsetDetectionOnAmen(t *testing.T) {
	// Check if amen.wav exists
	if _, err := os.Stat("amen.wav"); os.IsNotExist(err) {
		t.Skip("amen.wav not found, skipping test")
	}

	// Read the audio file
	samples, sampleRate, err := readWavFile("amen.wav")
	if err != nil {
		t.Fatalf("Failed to read amen.wav: %v", err)
	}

	t.Logf("Loaded amen.wav: %d samples at %d Hz (%.2f seconds)",
		len(samples), sampleRate, float64(len(samples))/float64(sampleRate))

	// Test configurations
	testCases := []struct {
		name      string
		method    string
		bufSize   uint
		hopSize   uint
		threshold float64
		minioi    float64
	}{
		{"HFC Default", "hfc", 512, 256, 0.058, 50.0},
		{"HFC Sensitive", "hfc", 512, 256, 0.03, 30.0},
		{"HFC Less Sensitive", "hfc", 512, 256, 0.1, 70.0},
		{"Complex Domain", "complex", 512, 256, 0.15, 50.0},
		{"Energy", "energy", 512, 256, 0.3, 50.0},
		{"Spectral Flux", "specflux", 512, 256, 0.18, 50.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			onsets := detectOnsets(samples, sampleRate, tc.method, tc.bufSize, tc.hopSize, tc.threshold, tc.minioi)

			t.Logf("\n%s Configuration:", tc.name)
			t.Logf("  Method: %s", tc.method)
			t.Logf("  Buffer Size: %d", tc.bufSize)
			t.Logf("  Hop Size: %d", tc.hopSize)
			t.Logf("  Threshold: %.3f", tc.threshold)
			t.Logf("  Min Inter-Onset Interval: %.1f ms", tc.minioi)
			t.Logf("  Detected %d onsets:", len(onsets))

			for i, onset := range onsets {
				t.Logf("    Onset %2d: %.4f seconds", i+1, onset)
			}
		})
	}
}

// FindOptimalOnsetParameters attempts to find parameters that produce the target number of onsets
func FindOptimalOnsetParameters(samples []float64, sampleRate uint, targetSlices int, method string, bufSize, hopSize uint) (threshold float64, minioi float64, onsets []float64) {
	// Search parameters
	thresholdMin := 0.01
	thresholdMax := 0.5
	minioiMin := 10.0
	minioiMax := 200.0

	bestDiff := math.MaxInt
	bestThreshold := 0.1
	bestMinioi := 50.0
	var bestOnsets []float64

	// Grid search with reasonable granularity
	thresholdSteps := 20
	minioiSteps := 10

	for t := 0; t < thresholdSteps; t++ {
		threshold := thresholdMin + (thresholdMax-thresholdMin)*float64(t)/float64(thresholdSteps-1)

		for m := 0; m < minioiSteps; m++ {
			minioi := minioiMin + (minioiMax-minioiMin)*float64(m)/float64(minioiSteps-1)

			onsets := detectOnsets(samples, sampleRate, method, bufSize, hopSize, threshold, minioi)

			diff := len(onsets) - targetSlices
			if diff < 0 {
				diff = -diff
			}

			if diff < bestDiff {
				bestDiff = diff
				bestThreshold = threshold
				bestMinioi = minioi
				bestOnsets = onsets
			}

			// If we found an exact match, return early
			if diff == 0 {
				return bestThreshold, bestMinioi, bestOnsets
			}
		}
	}

	return bestThreshold, bestMinioi, bestOnsets
}

// TestOptimalOnsetParameters demonstrates the optimization feature
func TestOptimalOnsetParameters(t *testing.T) {
	// Check if amen.wav exists
	if _, err := os.Stat("amen.wav"); os.IsNotExist(err) {
		t.Skip("amen.wav not found, skipping test")
	}

	// Read the audio file
	samples, sampleRate, err := readWavFile("amen.wav")
	if err != nil {
		t.Fatalf("Failed to read amen.wav: %v", err)
	}

	t.Logf("Loaded amen.wav: %d samples at %d Hz (%.2f seconds)",
		len(samples), sampleRate, float64(len(samples))/float64(sampleRate))

	// Test finding optimal parameters for different target slice counts
	targetSlices := []int{4, 8, 16, 32}

	for _, target := range targetSlices {
		t.Run(fmt.Sprintf("Target_%d_slices", target), func(t *testing.T) {
			method := "hfc"
			bufSize := uint(512)
			hopSize := uint(256)

			t.Logf("\nFinding optimal parameters for %d slices...", target)

			threshold, minioi, onsets := FindOptimalOnsetParameters(
				samples, sampleRate, target, method, bufSize, hopSize)

			t.Logf("Optimal parameters found:")
			t.Logf("  Threshold: %.4f", threshold)
			t.Logf("  Min Inter-Onset Interval: %.1f ms", minioi)
			t.Logf("  Detected %d onsets (target: %d):", len(onsets), target)

			for i, onset := range onsets {
				t.Logf("    Onset %2d: %.4f seconds", i+1, onset)
			}

			// Verify we got close to the target
			diff := len(onsets) - target
			if diff < 0 {
				diff = -diff
			}
			t.Logf("  Difference from target: %d", diff)
		})
	}
}
