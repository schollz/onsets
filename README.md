# goaubio-onset

A pure Go implementation of the aubio onset detection library. This library transpiles the onset detection functionality from the [aubio](https://aubio.org) C library to pure Go, with no CGO dependencies.

## Features

- **Pure Go**: No CGO, fully portable Go implementation
- **Multiple onset detection methods**: 
  - Energy
  - High Frequency Content (HFC)
  - Complex Domain
  - Phase-based
  - Weighted Phase
  - Spectral Difference
  - Kullback-Liebler (KL)
  - Modified Kullback-Liebler (MKL)
  - Spectral Flux
- **Configurable parameters**: Threshold, silence detection, minimum inter-onset interval
- **Adaptive spectral whitening**: Optional preprocessing for improved detection
- **Peak picking**: Robust peak detection with filtering

## Installation

```bash
go get github.com/schollz/goaubio-onset
```

## Usage

Here's a simple example of how to use the library:

```go
package main

import (
    "fmt"
    "math"
    
    "github.com/schollz/goaubio-onset"
)

func main() {
    // Create onset detector
    bufSize := uint(512)
    hopSize := uint(256)
    samplerate := uint(44100)
    
    o := onset.NewOnset("hfc", bufSize, hopSize, samplerate)
    
    // Create buffers
    input := onset.NewFvec(hopSize)
    output := onset.NewFvec(1)
    
    // Process audio frames
    for {
        // Fill input buffer with audio data
        // ... (read from file, stream, etc.)
        
        // Detect onsets
        o.Do(input, output)
        
        // Check if onset was detected
        if output.Data[0] > 0 {
            fmt.Printf("Onset detected at %.2f ms\n", o.GetLastMs())
        }
    }
}
```

## Onset Detection Methods

- **`energy`**: Energy-based onset detection
- **`hfc`** (default): High Frequency Content - good for percussive sounds
- **`complex`**: Complex Domain Method - robust for various sound types
- **`phase`**: Phase-based detection
- **`wphase`**: Weighted Phase Deviation
- **`specdiff`**: Spectral Difference
- **`kl`**: Kullback-Liebler divergence
- **`mkl`**: Modified Kullback-Liebler
- **`specflux`**: Spectral Flux

## Configuration

The onset detector can be configured with various parameters:

```go
o := onset.NewOnset("hfc", bufSize, hopSize, samplerate)

// Set detection threshold (0.0 - 1.0)
o.SetThreshold(0.3)

// Set silence threshold in dB
o.SetSilence(-70.0)

// Set minimum inter-onset interval in milliseconds
o.SetMinioiMs(50.0)

// Enable adaptive whitening
o.SetAWhitening(true)

// Set compression (for logarithmic magnitude)
o.SetCompression(1.0)
```

## API Reference

### Main Types

#### `Onset`

The main onset detection object.

**Constructor:**
```go
func NewOnset(onsetMode string, bufSize, hopSize, samplerate uint) *Onset
```

**Methods:**
- `Do(input *Fvec, onset *Fvec)` - Process input and detect onsets
- `GetLast() uint` - Get last onset time in samples
- `GetLastS() float64` - Get last onset time in seconds
- `GetLastMs() float64` - Get last onset time in milliseconds
- `SetThreshold(threshold float64)` - Set peak picking threshold
- `SetSilence(silence float64)` - Set silence threshold in dB
- `SetMinioi(minioi uint)` - Set minimum inter-onset interval in samples
- `SetMinioiMs(minioi float64)` - Set minimum inter-onset interval in ms
- `SetAWhitening(enable bool)` - Enable/disable adaptive whitening
- `SetCompression(lambda float64)` - Set compression factor
- `Reset()` - Reset the detector state

#### `Fvec`

Vector of real-valued floating point data.

**Constructor:**
```go
func NewFvec(length uint) *Fvec
```

#### `Cvec`

Vector of complex-valued data stored in polar form (magnitude and phase).

**Constructor:**
```go
func NewCvec(length uint) *Cvec
```

## Testing

Run the tests with:

```bash
go test -v
```

Run benchmarks:

```bash
go test -bench=.
```

## Examples

### Basic Usage Example

See the [example](./example/main.go) directory for a complete working example of onset detection.

```bash
cd example
go run main.go
```

### Slice Analyzer with Visualization

The [slice-analyzer](./examples/slice-analyzer/) example demonstrates:
- Loading audio files (left channel only, no merging for stereo)
- Automatic parameter optimization to find N onset slices
- Generating waveform plots with onset markers

```bash
cd examples/slice-analyzer
go build
./slice-analyzer -file ../../amen.wav -slices 8 -output waveform.png
```

## About Aubio

This library is a transpilation of the onset detection functionality from [aubio](https://github.com/aubio/aubio), a library for audio and music analysis. The original aubio library is written in C and is used extensively in music information retrieval applications.

## License

This implementation follows the GPL-3.0 license, consistent with the original aubio library.

## Credits

- Original aubio library by Paul Brossier
- Go implementation/transpilation for this project