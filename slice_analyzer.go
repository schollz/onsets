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
	// Method specifies the onset detection method to use.
	// Supported methods: "hfc", "energy", "complex", "phase", "wphase", "specdiff", "kl", "mkl", "specflux", "consensus"
	// Default is "hfc" if empty.
	// The special "consensus" method uses all methods and generates consensus markers.
	Method string
	// MinConsensusClusterSize specifies the minimum number of onset markers required
	// for a cluster to be considered valid when using the "consensus" method.
	// Default is 3. Only applies when Method is "consensus".
	MinConsensusClusterSize int
	// UseMinimumSpacing enables minimum spacing filter between slices.
	// When true, if multiple slices fall within MinimumSpacing window, only the first is kept.
	// Default is true.
	UseMinimumSpacing bool
	// MinimumSpacing specifies the minimum spacing in milliseconds between slices.
	// If multiple slices fall within this window, only the first is kept.
	// Default is 80.0 ms. Only applies when UseMinimumSpacing is true.
	MinimumSpacing float64
}

// DefaultSliceAnalyzerOptions returns default options for slice analysis
func DefaultSliceAnalyzerOptions() SliceAnalyzerOptions {
	return SliceAnalyzerOptions{
		NumSlices:               0,
		Optimize:                true,
		OptimizeWindowMs:        100.0,
		Method:                  "hfc",
		MinConsensusClusterSize: 3,
		UseMinimumSpacing:       true,
		MinimumSpacing:          80.0,
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

	// Default to "hfc" if method is not specified
	method := options.Method
	if method == "" {
		method = "hfc"
	}

	var onsets []float64

	if method == "consensus" {
		// Use consensus method: run all methods and generate consensus
		onsets = findConsensusOnsets(samples, sampleRate, options)
	} else if options.NumSlices > 0 {
		// Find the best N onsets based on energy
		onsets = findBestOnsets(samples, sampleRate, options.NumSlices, method)
	} else {
		// Find all onsets
		onsets = findAllOnsets(samples, sampleRate, method)
	}

	// Optimize onset positions if requested
	if options.Optimize && len(onsets) > 0 {
		onsets = optimizeOnsetPositions(samples, sampleRate, onsets, options.OptimizeWindowMs)
	}

	// Apply minimum spacing filter if requested
	if options.UseMinimumSpacing && len(onsets) > 0 {
		onsets = applyMinimumSpacing(onsets, options.MinimumSpacing)
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
func findBestOnsets(samples []float64, sampleRate uint, targetSlices int, method string) []float64 {
	bufSize := uint(512)
	hopSize := uint(256)

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
func findAllOnsets(samples []float64, sampleRate uint, method string) []float64 {
	bufSize := uint(512)
	hopSize := uint(256)

	return detectAllOnsets(samples, sampleRate, method, bufSize, hopSize)
}

// findConsensusOnsets runs all detection methods and generates consensus markers
// by clustering nearby onsets and taking the midpoint of each cluster
func findConsensusOnsets(samples []float64, sampleRate uint, options SliceAnalyzerOptions) []float64 {
	bufSize := uint(512)
	hopSize := uint(256)

	// All available methods
	methods := []string{"energy", "hfc", "complex", "phase", "wphase", "specdiff", "kl", "mkl", "specflux"}

	// Collect all onsets from all methods
	var allOnsets []float64
	for _, method := range methods {
		methodOnsets := detectAllOnsets(samples, sampleRate, method, bufSize, hopSize)
		allOnsets = append(allOnsets, methodOnsets...)
	}

	if len(allOnsets) == 0 {
		return []float64{}
	}

	// Sort all onsets by time
	sort.Float64s(allOnsets)

	// Cluster nearby onsets together
	// Two onsets are in the same cluster if they're within clusterThreshold seconds
	clusterThreshold := 0.05 // 50ms threshold for clustering

	// Default minimum cluster size to 3 if not set
	minClusterSize := options.MinConsensusClusterSize
	if minClusterSize <= 0 {
		minClusterSize = 3
	}

	var consensusOnsets []float64
	currentCluster := []float64{allOnsets[0]}

	for i := 1; i < len(allOnsets); i++ {
		if allOnsets[i]-currentCluster[len(currentCluster)-1] <= clusterThreshold {
			// Add to current cluster
			currentCluster = append(currentCluster, allOnsets[i])
		} else {
			// Finalize current cluster if it meets minimum size requirement
			if len(currentCluster) >= minClusterSize {
				consensusOnsets = append(consensusOnsets, calculateClusterMidpoint(currentCluster))
			}
			currentCluster = []float64{allOnsets[i]}
		}
	}

	// Don't forget the last cluster if it meets minimum size requirement
	if len(currentCluster) >= minClusterSize {
		consensusOnsets = append(consensusOnsets, calculateClusterMidpoint(currentCluster))
	}

	// If targetSlices is specified, select the best N based on cluster size and energy
	if options.NumSlices > 0 && len(consensusOnsets) > options.NumSlices {
		// For consensus, we could rank by cluster size (more methods agreeing)
		// But for simplicity, we'll use energy like in findBestOnsets
		onsetsWithEnergy := make([]onsetWithEnergy, len(consensusOnsets))
		for i, onsetTime := range consensusOnsets {
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
		bestOnsets := onsetsWithEnergy[:options.NumSlices]

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

	return consensusOnsets
}

// calculateClusterMidpoint calculates the midpoint of a cluster of onset times
// after removing outliers using the IQR method
func calculateClusterMidpoint(cluster []float64) float64 {
	if len(cluster) == 0 {
		return 0.0
	}

	// For small clusters (< 4), don't remove outliers
	if len(cluster) < 4 {
		sum := 0.0
		for _, time := range cluster {
			sum += time
		}
		return sum / float64(len(cluster))
	}

	// Remove outliers using IQR method
	cleanedCluster := removeOutliers(cluster)

	// If all values were outliers (shouldn't happen), use original cluster
	if len(cleanedCluster) == 0 {
		cleanedCluster = cluster
	}

	sum := 0.0
	for _, time := range cleanedCluster {
		sum += time
	}

	return sum / float64(len(cleanedCluster))
}

// removeOutliers removes outliers from a cluster using the IQR (Interquartile Range) method
func removeOutliers(data []float64) []float64 {
	if len(data) < 4 {
		return data
	}

	// Create a sorted copy
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	// Calculate Q1, Q2 (median), and Q3
	q1 := calculatePercentile(sorted, 25)
	q3 := calculatePercentile(sorted, 75)

	// Calculate IQR
	iqr := q3 - q1

	// Define outlier bounds (using 1.5 * IQR, standard for outlier detection)
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	// Filter out outliers
	var result []float64
	for _, value := range data {
		if value >= lowerBound && value <= upperBound {
			result = append(result, value)
		}
	}

	return result
}

// calculatePercentile calculates the nth percentile of a sorted array
func calculatePercentile(sorted []float64, percentile float64) float64 {
	if len(sorted) == 0 {
		return 0.0
	}

	if len(sorted) == 1 {
		return sorted[0]
	}

	// Calculate the rank
	rank := (percentile / 100.0) * float64(len(sorted)-1)
	lowerIndex := int(math.Floor(rank))
	upperIndex := int(math.Ceil(rank))

	// Handle edge cases
	if lowerIndex < 0 {
		lowerIndex = 0
	}
	if upperIndex >= len(sorted) {
		upperIndex = len(sorted) - 1
	}

	// Linear interpolation between the two nearest ranks
	if lowerIndex == upperIndex {
		return sorted[lowerIndex]
	}

	weight := rank - float64(lowerIndex)
	return sorted[lowerIndex]*(1-weight) + sorted[upperIndex]*weight
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

// applyMinimumSpacing filters onsets to ensure minimum spacing between them.
// If multiple onsets fall within the minimum spacing window, only the first is kept.
func applyMinimumSpacing(onsets []float64, minimumSpacingMs float64) []float64 {
	if len(onsets) == 0 {
		return onsets
	}

	// Convert minimum spacing from milliseconds to seconds
	minimumSpacingSec := minimumSpacingMs / 1000.0

	// First onset is always kept
	filtered := []float64{onsets[0]}

	// Check each subsequent onset
	for i := 1; i < len(onsets); i++ {
		// Calculate the time difference from the last kept onset
		timeDiff := onsets[i] - filtered[len(filtered)-1]

		// Only keep this onset if it's far enough from the previous one
		if timeDiff >= minimumSpacingSec {
			filtered = append(filtered, onsets[i])
		}
		// Otherwise, skip this onset (it's too close to the previous one)
	}

	return filtered
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
