# Slice Analyzer Example

This example program demonstrates how to use the goaubio-onset library to detect onset slices in audio files and visualize them.

## Features

- Analyzes audio files to find onset points (slices)
- Uses only the left channel for stereo files (no channel merging)
- Automatically finds optimal detection parameters to match the desired number of slices
- Generates a waveform plot with:
  - Light gray waveform visualization
  - White vertical lines at each detected onset/slice point
  - Black background for contrast

## Building

```bash
go build
```

## Usage

```bash
./slice-analyzer -file <path-to-audio-file> [-slices N] [-output output.png]
```

### Arguments

- `-file` (required): Path to the audio file (WAV format)
- `-slices` (optional): Number of slices to find (default: 8)
- `-output` (optional): Output PNG file path (default: waveform.png)

### Examples

Find 8 slices in an audio file:
```bash
./slice-analyzer -file song.wav
```

Find 16 slices and save to a custom location:
```bash
./slice-analyzer -file song.wav -slices 16 -output my_slices.png
```

## How It Works

1. **Audio Loading**: The program reads the audio file and extracts only the left channel (or mono channel if the file is mono)
2. **Onset Detection**: Uses the High Frequency Content (HFC) method to detect onsets
3. **Parameter Optimization**: Automatically searches for optimal threshold and minimum inter-onset interval parameters to find approximately the requested number of slices
4. **Visualization**: Creates a waveform plot with the detected onset points marked as vertical white lines

## Notes

- The program may not find exactly the requested number of slices due to the characteristics of the audio
- Different audio files may require different detection parameters for optimal results
- The algorithm focuses on finding the most prominent onsets/transients in the audio
