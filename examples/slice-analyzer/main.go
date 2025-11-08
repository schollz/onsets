package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/go-audio/wav"
	"github.com/schollz/goaubio-onset"
)

func main() {
	// Parse command-line arguments
	soundFile := flag.String("file", "", "Path to the sound file (required)")
	numSlices := flag.Int("slices", 8, "Number of slices to find (default: 8)")
	outputFile := flag.String("output", "waveform.html", "Output HTML file (default: waveform.html)")
	flag.Parse()

	if *soundFile == "" {
		fmt.Println("Error: sound file is required")
		flag.Usage()
		os.Exit(1)
	}

	if *numSlices < 1 {
		fmt.Println("Error: number of slices must be at least 1")
		os.Exit(1)
	}

	// Read audio file (left channel only)
	samples, sampleRate, err := readWavFileLeftChannel(*soundFile)
	if err != nil {
		log.Fatalf("Failed to read audio file: %v", err)
	}

	fmt.Printf("Loaded: %s\n", filepath.Base(*soundFile))
	fmt.Printf("  Samples: %d\n", len(samples))
	fmt.Printf("  Sample Rate: %d Hz\n", sampleRate)
	fmt.Printf("  Duration: %.2f seconds\n", float64(len(samples))/float64(sampleRate))
	fmt.Printf("  Finding best %d slices...\n", *numSlices)

	// Find the best N onsets
	onsets := findBestOnsets(samples, sampleRate, *numSlices)

	if len(onsets) == 0 {
		log.Fatal("No onsets detected. Try adjusting parameters or using a different audio file.")
	}

	fmt.Printf("Found %d onsets:\n", len(onsets))
	for i, onset := range onsets {
		fmt.Printf("  %2d: %.4f seconds (sample %d)\n", i+1, onset, int(onset*float64(sampleRate)))
	}

	// Write data to JSON file
	dataFile := "waveform_data.json"
	err = writeDataToJSON(samples, sampleRate, onsets, dataFile)
	if err != nil {
		log.Fatalf("Failed to write data file: %v", err)
	}

	// Run plotly visualization script
	fmt.Printf("\nGenerating visualization...\n")
	err = runPlotlyScript(dataFile, *outputFile)
	if err != nil {
		log.Fatalf("Failed to generate plot: %v", err)
	}

	fmt.Printf("Waveform plot saved to: %s\n", *outputFile)
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

// findBestOnsets uses onset detection to find the best N onsets in the audio
// The "best" onsets are those with the highest energy/loudness
func findBestOnsets(samples []float64, sampleRate uint, targetSlices int) []float64 {
	bufSize := uint(512)
	hopSize := uint(256)
	method := "hfc" // High Frequency Content method

	// Detect all onsets with relaxed parameters to get more candidates
	allOnsets := detectAllOnsets(samples, sampleRate, method, bufSize, hopSize)

	fmt.Printf("  Detected %d total onsets\n", len(allOnsets))

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

// detectAllOnsets detects all onsets with relaxed parameters
func detectAllOnsets(samples []float64, sampleRate uint, method string, bufSize, hopSize uint) []float64 {
	// Use low threshold and short minioi to detect all possible onsets
	threshold := 0.02
	minioi := 10.0 // milliseconds

	return detectOnsets(samples, sampleRate, method, bufSize, hopSize, threshold, minioi)
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


// detectOnsets processes audio samples and returns onset times in seconds
func detectOnsets(samples []float64, sampleRate uint, method string, bufSize, hopSize uint, threshold float64, minioi float64) []float64 {
	o := onset.NewOnset(method, bufSize, hopSize, sampleRate)
	o.SetThreshold(threshold)
	o.SetMinioiMs(minioi)

	input := onset.NewFvec(hopSize)
	output := onset.NewFvec(1)

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

// WaveformData represents the data structure for JSON export
type WaveformData struct {
	Samples    []float64 `json:"samples"`
	SampleRate uint      `json:"sample_rate"`
	Onsets     []float64 `json:"onsets"`
}

// writeDataToJSON writes the waveform and onset data to a JSON file
func writeDataToJSON(samples []float64, sampleRate uint, onsets []float64, filename string) error {
	data := WaveformData{
		Samples:    samples,
		SampleRate: sampleRate,
		Onsets:     onsets,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

// runPlotlyScript executes the Python plotly script to generate the visualization
func runPlotlyScript(dataFile, outputFile string) error {
	cmd := exec.Command("python3", "plot_waveform.py", dataFile, outputFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run plotly script: %w\nOutput: %s", err, string(output))
	}
	return nil
}
