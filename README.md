# onsets

[![CI](https://github.com/schollz/onsets/actions/workflows/ci.yml/badge.svg)](https://github.com/schollz/onsets/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/schollz/onsets/branch/main/graph/badge.svg)](https://codecov.io/gh/schollz/onsets)
[![GoDoc](https://pkg.go.dev/badge/github.com/schollz/onsets)](https://pkg.go.dev/github.com/schollz/onsets)
[![Release](https://img.shields.io/github/v/release/schollz/onsets)](https://github.com/schollz/onsets/releases/latest)

A pure Go implementation of the aubio onset detection library for audio slice detection and analysis.

<img width="1400" height="600" alt="newplot" src="https://github.com/user-attachments/assets/49afea95-5922-4c5e-8f27-e4a50bc45b74" />


## Installation

```bash
go get github.com/schollz/onsets
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/schollz/onsets"
)

func main() {
    // Analyze a WAV file for onsets
    result, err := onset.AnalyzeSlices("audio.wav", onset.DefaultSliceAnalyzerOptions())
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d onsets:\n", len(result.Onsets))
    for i, onsetTime := range result.Onsets {
        fmt.Printf("  %d: %.4f seconds\n", i+1, onsetTime)
    }
}
```

## Customizing Options

```go
options := onset.SliceAnalyzerOptions{
    NumSlices:        8,      // Number of slices to find (0 = all)
    Method:           "hfc",  // Detection method
    Optimize:         true,   // Optimize onset positions
    OptimizeWindowMs: 15.0,   // Optimization window (ms)
}

result, err := onset.AnalyzeSlices("audio.wav", options)
```

### Recommended Settings for Best Results

For high-quality onset detection:
```go
options := onset.SliceAnalyzerOptions{
    NumSlices:        32,
    Method:           "hfc",
    Optimize:         true,
    OptimizeWindowMs: 15.0,
}
```

## Detection Methods

- **`hfc`** (recommended): High Frequency Content - best for percussive sounds
- **`consensus`**: Uses all methods and finds agreement (robust but slower)
- **`energy`**: Energy-based detection
- **`complex`**: Complex Domain Method
- **`phase`**: Phase-based detection
- **`wphase`**: Weighted Phase Deviation
- **`specdiff`**: Spectral Difference
- **`kl`**: Kullback-Liebler divergence
- **`mkl`**: Modified Kullback-Liebler
- **`specflux`**: Spectral Flux

### Consensus Method Options

The `consensus` method runs all detection methods and clusters their results:

```go
options := onset.SliceAnalyzerOptions{
    Method:                  "consensus",
    MinConsensusClusterSize: 3,  // Minimum methods that must agree (default: 3)
}
```

## Command-Line Tool

Build and use the slice analyzer tool:

```bash
cd examples/slice-analyzer
go build
./slice-analyzer -file audio.wav -slices 32 --method hfc --optimize --optimize-window 15
```

Options:
- `-file`: Path to WAV file (required)
- `-slices`: Number of slices to find (default: 8, 0 = all)
- `-method`: Detection method (default: hfc)
- `-optimize`: Optimize onset positions (default: true)
- `-optimize-window`: Optimization window in ms (default: 100.0)
- `-min-consensus-cluster`: Min cluster size for consensus method (default: 3)
- `-output`: Output HTML file (default: waveform.html)

## API Reference

### SliceAnalyzerOptions

```go
type SliceAnalyzerOptions struct {
    // Number of slices to find (0 = all onsets)
    NumSlices int

    // Optimize onset positions using variance analysis
    Optimize bool

    // Optimization window size in milliseconds
    OptimizeWindowMs float64

    // Detection method: "hfc", "energy", "consensus", etc.
    Method string

    // Minimum cluster size for consensus method (default: 3)
    MinConsensusClusterSize int
}
```

### SliceAnalyzerResult

```go
type SliceAnalyzerResult struct {
    // Detected onset times in seconds
    Onsets []float64

    // Audio samples (left channel)
    Samples []float64

    // Sample rate
    SampleRate uint
}
```

### Functions

```go
// Analyze a WAV file for onsets
func AnalyzeSlices(wavFile string, options SliceAnalyzerOptions) (*SliceAnalyzerResult, error)

// Get default options
func DefaultSliceAnalyzerOptions() SliceAnalyzerOptions
```

## Low-Level API

For streaming or custom audio processing:

```go
// Create onset detector
o := onset.NewOnset("hfc", 512, 256, 44100)
o.SetThreshold(0.3)
o.SetMinioiMs(50.0)

// Create buffers
input := onset.NewFvec(256)
output := onset.NewFvec(1)

// Process frames
for {
    // Fill input.Data with audio samples
    o.Do(input, output)

    if output.Data[0] > 0 {
        fmt.Printf("Onset at %.2f ms\n", o.GetLastMs())
    }
}
```

## Features

- **Pure Go**: No CGO dependencies, fully portable
- **High-level API**: Simple slice analysis with automatic optimization
- **Multiple detection methods**: 9 different onset detection algorithms
- **Consensus detection**: Combines all methods for robust results
- **Outlier removal**: Filters anomalous detections in consensus mode
- **Position optimization**: Refines onset positions using variance analysis
- **Energy-based selection**: Automatically selects the N best onsets by energy

## Testing

```bash
go test -v
```

## About

This library is a Go implementation of onset detection from [aubio](https://github.com/aubio/aubio), a library for audio and music analysis by Paul Brossier.

## License

GPL-3.0 (consistent with the original aubio library)
