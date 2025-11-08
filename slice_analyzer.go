package onset

import (
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/go-audio/wav"
)

// SliceAnalyzerResult contains the results of slice analysis
type SliceAnalyzerResult struct {
	// Onsets contains the detected onset times in seconds
	Onsets []float64
	// Samples contains the audio samples (left channel only for stereo files)
	Samples []float64
	// SampleRate is the sample rate of the audio file
	SampleRate uint
}

// SliceAnalyzerOptions contains configuration options for slice analysis
type SliceAnalyzerOptions struct {
	// NumSlices specifies the number of slices to find.
	// If 0 (default), all onsets are detected.
	// If > 0, the best N onsets based on energy are selected.
	NumSlices int
	// Optimize enables optimization of onset positions using variance analysis.
	// Default is true.
	Optimize bool
	// OptimizeWindowMs specifies the window size in milliseconds for onset optimization.
	// Default is 100.0 ms.
	OptimizeWindowMs float64
}

// DefaultSliceAnalyzerOptions returns default options for slice analysis
func DefaultSliceAnalyzerOptions() SliceAnalyzerOptions {
	return SliceAnalyzerOptions{
		NumSlices:        0,
		Optimize:         true,
		OptimizeWindowMs: 100.0,
	}
}

// AnalyzeSlices performs onset detection and slice analysis on a WAV file.
// It returns the detected onset times along with audio samples and metadata.
//
// Parameters:
//   - wavFile: Path to the WAV file to analyze
//   - options: Configuration options for the analysis
//
// Returns:
//   - SliceAnalyzerResult containing onsets, samples, and sample rate
//   - error if the file cannot be read or processed
func AnalyzeSlices(wavFile string, options SliceAnalyzerOptions) (*SliceAnalyzerResult, error) {
	// Read audio file (left channel only)
	samples, sampleRate, err := readWavFileLeftChannel(wavFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file: %w", err)
	}

	var onsets []float64

	if options.NumSlices > 0 {
		// Find the best N onsets based on energy
		onsets = findBestOnsets(samples, sampleRate, options.NumSlices)
	} else {
		// Find all onsets
		onsets = findAllOnsets(samples, sampleRate)
	}

	// Optimize onset positions if requested
	if options.Optimize && len(onsets) > 0 {
		onsets = optimizeOnsetPositions(samples, sampleRate, onsets, options.OptimizeWindowMs)
	}

	return &SliceAnalyzerResult{
		Onsets:     onsets,
		Samples:    samples,
		SampleRate: sampleRate,
	}, nil
}

// readWavFileLeftChannel reads a WAV file and returns only the left channel (or mono)
func readWavFileLeftChannel(filename string) ([]float64, uint, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	decoder := wav.NewDecoder(f)
	if !decoder.IsValidFile() {
		return nil, 0, fmt.Errorf("invalid WAV file")
	}

	sampleRate := uint(decoder.SampleRate)

	// Read all audio data
	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read PCM data: %w", err)
	}

	numChannels := buf.Format.NumChannels
	numSamples := len(buf.Data) / numChannels
	samples := make([]float64, numSamples)

	// Extract left channel only (channel 0)
	for i := 0; i < numSamples; i++ {
		// Normalize int to float64 [-1.0, 1.0]
		samples[i] = float64(buf.Data[i*numChannels]) / 32768.0
	}

	return samples, sampleRate, nil
}

// onsetWithEnergy stores an onset time and its energy
type onsetWithEnergy struct {
	time   float64
	energy float64
}

// findBestOnsets uses onset detection to find the best N onsets in the audio.
// The "best" onsets are those with the highest energy/loudness.
func findBestOnsets(samples []float64, sampleRate uint, targetSlices int) []float64 {
	bufSize := uint(512)
	hopSize := uint(256)
	method := "hfc" // High Frequency Content method

	// Detect all onsets with relaxed parameters to get more candidates
	allOnsets := detectAllOnsets(samples, sampleRate, method, bufSize, hopSize)

	if len(allOnsets) == 0 {
		return []float64{}
	}

	// Calculate energy at each onset
	onsetsWithEnergy := make([]onsetWithEnergy, len(allOnsets))
	for i, onsetTime := range allOnsets {
		energy := calculateOnsetEnergy(samples, sampleRate, onsetTime)
		onsetsWithEnergy[i] = onsetWithEnergy{
			time:   onsetTime,
			energy: energy,
		}
	}

	// Sort by energy (descending)
	sort.Slice(onsetsWithEnergy, func(i, j int) bool {
		return onsetsWithEnergy[i].energy > onsetsWithEnergy[j].energy
	})

	// Take top N onsets
	numToSelect := targetSlices
	if numToSelect > len(onsetsWithEnergy) {
		numToSelect = len(onsetsWithEnergy)
	}
	bestOnsets := onsetsWithEnergy[:numToSelect]

	// Sort back by time for output
	sort.Slice(bestOnsets, func(i, j int) bool {
		return bestOnsets[i].time < bestOnsets[j].time
	})

	// Extract just the times
	result := make([]float64, len(bestOnsets))
	for i, onset := range bestOnsets {
		result[i] = onset.time
	}

	return result
}

// findAllOnsets detects all onsets in the audio with default parameters
func findAllOnsets(samples []float64, sampleRate uint) []float64 {
	bufSize := uint(512)
	hopSize := uint(256)
	method := "hfc" // High Frequency Content method

	return detectAllOnsets(samples, sampleRate, method, bufSize, hopSize)
}

// detectAllOnsets detects all onsets with relaxed parameters
func detectAllOnsets(samples []float64, sampleRate uint, method string, bufSize, hopSize uint) []float64 {
	// Use low threshold and short minioi to detect all possible onsets
	threshold := 0.02
	minioi := 10.0 // milliseconds

	return detectOnsetsInternal(samples, sampleRate, method, bufSize, hopSize, threshold, minioi)
}

// calculateOnsetEnergy calculates the RMS energy around an onset
func calculateOnsetEnergy(samples []float64, sampleRate uint, onsetTime float64) float64 {
	// Calculate energy in a window around the onset
	windowMs := 50.0 // 50ms window
	windowSamples := int(windowMs * float64(sampleRate) / 1000.0)

	onsetSample := int(onsetTime * float64(sampleRate))

	// Window starts at onset and extends forward
	startSample := onsetSample
	endSample := onsetSample + windowSamples

	// Clamp to valid range
	if startSample < 0 {
		startSample = 0
	}
	if endSample > len(samples) {
		endSample = len(samples)
	}

	// Calculate RMS energy
	sumSquares := 0.0
	count := 0
	for i := startSample; i < endSample; i++ {
		sumSquares += samples[i] * samples[i]
		count++
	}

	if count == 0 {
		return 0.0
	}

	return math.Sqrt(sumSquares / float64(count))
}

// optimizeOnsetPositions refines onset positions by finding the point of maximum variance difference
// within a window around each detected onset
func optimizeOnsetPositions(samples []float64, sampleRate uint, onsets []float64, windowMs float64) []float64 {
	optimized := make([]float64, len(onsets))

	for i, onsetTime := range onsets {
		optimized[i] = findOptimalOnsetPosition(samples, sampleRate, onsetTime, windowMs)
	}

	return optimized
}

// findOptimalOnsetPosition finds the exact onset position by locating the midpoint
// with the maximum variance difference between right and left sides within a window
func findOptimalOnsetPosition(samples []float64, sampleRate uint, onsetTime float64, windowMs float64) float64 {
	// Convert onset time to sample index
	onsetSample := int(onsetTime * float64(sampleRate))

	// Calculate window size in samples (centered around onset)
	windowSamples := int(windowMs * float64(sampleRate) / 1000.0)
	halfWindow := windowSamples / 2

	// Define search window boundaries
	windowStart := onsetSample - halfWindow
	windowEnd := onsetSample + halfWindow

	// Clamp to valid range
	if windowStart < 0 {
		windowStart = 0
	}
	if windowEnd > len(samples) {
		windowEnd = len(samples)
	}

	// If window is too small, return original onset
	if windowEnd-windowStart < 10 {
		return onsetTime
	}

	// Search for the midpoint with maximum variance difference
	maxDiff := -math.MaxFloat64
	bestPosition := onsetSample

	// Try each position in the window as a potential midpoint
	// Leave some margin on both sides to calculate variance
	minMargin := 5 // minimum samples on each side
	for midpoint := windowStart + minMargin; midpoint < windowEnd-minMargin; midpoint++ {
		// Calculate variance of left side (from window start to midpoint)
		leftVariance := calculateVariance(samples, windowStart, midpoint)

		// Calculate variance of right side (from midpoint to window end)
		rightVariance := calculateVariance(samples, midpoint, windowEnd)

		// Calculate difference (right - left)
		// Positive difference means signal variance increases at this point (onset characteristic)
		diff := rightVariance - leftVariance

		// Track maximum difference
		if diff > maxDiff {
			maxDiff = diff
			bestPosition = midpoint
		}
	}

	// Convert best position back to time
	return float64(bestPosition) / float64(sampleRate)
}

// calculateVariance computes the variance of a sample range
func calculateVariance(samples []float64, start, end int) float64 {
	if start >= end || start < 0 || end > len(samples) {
		return 0.0
	}

	count := end - start
	if count == 0 {
		return 0.0
	}

	// Calculate mean
	sum := 0.0
	for i := start; i < end; i++ {
		sum += samples[i]
	}
	mean := sum / float64(count)

	// Calculate variance
	sumSquaredDiff := 0.0
	for i := start; i < end; i++ {
		diff := samples[i] - mean
		sumSquaredDiff += diff * diff
	}

	return sumSquaredDiff / float64(count)
}

// detectOnsetsInternal processes audio samples and returns onset times in seconds
func detectOnsetsInternal(samples []float64, sampleRate uint, method string, bufSize, hopSize uint, threshold float64, minioi float64) []float64 {
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
