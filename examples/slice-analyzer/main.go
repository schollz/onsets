package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/schollz/onsets"
)

func main() {
	// Parse command-line arguments
	soundFile := flag.String("file", "", "Path to the sound file (required)")
	numSlices := flag.Int("slices", 8, "Number of slices to find (default: 8, 0 means all)")
	outputFile := flag.String("output", "waveform.html", "Output HTML file (default: waveform.html)")
	optimizeOnsets := flag.Bool("optimize", true, "Optimize onset positions using RMS differential (default: true)")
	optimizeWindowMs := flag.Float64("optimize-window", 100.0, "Window size in milliseconds for onset optimization (default: 100.0)")
	method := flag.String("method", "hfc", "Onset detection method: hfc, energy, complex, phase, wphase, specdiff, kl, mkl, specflux, consensus (default: hfc)")
	minConsensusClusterSize := flag.Int("min-consensus-cluster", 3, "Minimum cluster size for consensus method (default: 3)")
	useMinimumSpacing := flag.Bool("use-minimum-spacing", true, "Enable minimum spacing filter between slices (default: true)")
	minimumSpacing := flag.Float64("minimum-spacing", 80.0, "Minimum spacing in milliseconds between slices (default: 80.0)")
	flag.Parse()

	if *soundFile == "" {
		fmt.Println("Error: sound file is required")
		flag.Usage()
		os.Exit(1)
	}

	if *numSlices < 0 {
		fmt.Println("Error: number of slices must be 0 or greater")
		os.Exit(1)
	}

	// Use the slice analyzer API
	options := onset.SliceAnalyzerOptions{
		NumSlices:               *numSlices,
		Optimize:                *optimizeOnsets,
		OptimizeWindowMs:        *optimizeWindowMs,
		Method:                  *method,
		MinConsensusClusterSize: *minConsensusClusterSize,
		UseMinimumSpacing:       *useMinimumSpacing,
		MinimumSpacing:          *minimumSpacing,
	}

	result, err := onset.AnalyzeSlices(*soundFile, options)
	if err != nil {
		log.Fatalf("Failed to analyze slices: %v", err)
	}

	fmt.Printf("Loaded: %s\n", filepath.Base(*soundFile))
	fmt.Printf("  Samples: %d\n", len(result.Samples))
	fmt.Printf("  Sample Rate: %d Hz\n", result.SampleRate)
	fmt.Printf("  Duration: %.2f seconds\n", float64(len(result.Samples))/float64(result.SampleRate))
	fmt.Printf("  Method: %s\n", *method)
	if *numSlices > 0 {
		fmt.Printf("  Finding best %d slices...\n", *numSlices)
	} else {
		fmt.Printf("  Finding all slices...\n")
	}

	if len(result.Onsets) == 0 {
		log.Fatal("No onsets detected. Try adjusting parameters or using a different audio file.")
	}

	fmt.Printf("Found %d onsets:\n", len(result.Onsets))
	for i, onset := range result.Onsets {
		fmt.Printf("  %2d: %.4f seconds (sample %d)\n", i+1, onset, int(onset*float64(result.SampleRate)))
	}

	// Write data to JSON file
	dataFile := "waveform_data.json"
	err = writeDataToJSON(result.Samples, result.SampleRate, result.Onsets, dataFile)
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
