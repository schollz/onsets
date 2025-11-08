package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/go-audio/wav"
	"github.com/schollz/goaubio-onset"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func main() {
	// Parse command-line arguments
	soundFile := flag.String("file", "", "Path to the sound file (required)")
	numSlices := flag.Int("slices", 8, "Number of slices to find (default: 8)")
	outputFile := flag.String("output", "waveform.png", "Output PNG file (default: waveform.png)")
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

	// Plot waveform with slice markers
	err = plotWaveform(samples, sampleRate, onsets, *outputFile)
	if err != nil {
		log.Fatalf("Failed to create plot: %v", err)
	}

	fmt.Printf("\nWaveform plot saved to: %s\n", *outputFile)
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

// findBestOnsets uses onset detection to find the best N onsets in the audio
func findBestOnsets(samples []float64, sampleRate uint, targetSlices int) []float64 {
	bufSize := uint(512)
	hopSize := uint(256)
	method := "hfc" // High Frequency Content method

	// Try to find the best parameters to get close to targetSlices
	threshold, minioi, onsets := findOptimalOnsetParameters(samples, sampleRate, targetSlices, method, bufSize, hopSize)

	fmt.Printf("  Using threshold: %.3f, minioi: %.1f ms\n", threshold, minioi)

	return onsets
}

// findOptimalOnsetParameters searches for parameters that produce the target number of onsets
func findOptimalOnsetParameters(samples []float64, sampleRate uint, targetSlices int, method string, bufSize, hopSize uint) (threshold float64, minioi float64, onsets []float64) {
	thresholdMin := 0.01
	thresholdMax := 0.5
	minioiMin := 10.0
	minioiMax := 200.0

	bestDiff := math.MaxInt
	bestThreshold := 0.058
	bestMinioi := 50.0
	var bestOnsets []float64

	// Grid search
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

			// If we found the exact number, we can stop
			if diff == 0 {
				return bestThreshold, bestMinioi, bestOnsets
			}
		}
	}

	return bestThreshold, bestMinioi, bestOnsets
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

// plotWaveform creates a waveform plot with onset markers
func plotWaveform(samples []float64, sampleRate uint, onsets []float64, outputFile string) error {
	p := plot.New()

	p.Title.Text = "Waveform with Onset Slices"
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "Amplitude"

	// Create waveform data points
	waveformPoints := make(plotter.XYs, len(samples))
	for i, sample := range samples {
		timeSeconds := float64(i) / float64(sampleRate)
		waveformPoints[i].X = timeSeconds
		waveformPoints[i].Y = sample
	}

	// Add waveform line
	waveformLine, err := plotter.NewLine(waveformPoints)
	if err != nil {
		return fmt.Errorf("failed to create waveform line: %w", err)
	}
	waveformLine.LineStyle.Color = color.RGBA{R: 200, G: 200, B: 200, A: 255} // Light gray
	waveformLine.LineStyle.Width = vg.Points(0.5)
	p.Add(waveformLine)

	// Add vertical lines for each onset
	for _, onset := range onsets {
		line, err := plotter.NewLine(plotter.XYs{
			{X: onset, Y: -1.0},
			{X: onset, Y: 1.0},
		})
		if err != nil {
			return fmt.Errorf("failed to create onset line: %w", err)
		}
		line.LineStyle.Color = color.White
		line.LineStyle.Width = vg.Points(2)
		p.Add(line)
	}

	// Set background to black for better contrast with white lines
	p.BackgroundColor = color.Black
	p.X.Tick.Label.Color = color.White
	p.Y.Tick.Label.Color = color.White
	p.X.Label.TextStyle.Color = color.White
	p.Y.Label.TextStyle.Color = color.White
	p.Title.TextStyle.Color = color.White

	// Save the plot
	if err := p.Save(12*vg.Inch, 4*vg.Inch, outputFile); err != nil {
		return fmt.Errorf("failed to save plot: %w", err)
	}

	return nil
}
