# Slice Analyzer Example

This example program demonstrates how to use the goaubio-onset library to detect onset slices in audio files and visualize them.

## Features

- Analyzes audio files to find onset points (slices)
- Uses only the left channel for stereo files (no channel merging)
- Automatically finds optimal detection parameters to match the desired number of slices
- Generates an interactive waveform plot (HTML) with:
  - Light gray waveform visualization
  - Red vertical lines at each detected onset/slice point
  - Dark theme with hover information
  - Interactive zooming and panning capabilities

## Requirements

- Go 1.16 or higher
- Python 3.x with plotly installed:
  ```bash
  pip install plotly
  ```

## Building

```bash
go build
```

## Usage

```bash
./slice-analyzer -file <path-to-audio-file> [-slices N] [-output output.html]
```

### Arguments

- `-file` (required): Path to the audio file (WAV format)
- `-slices` (optional): Number of slices to find (default: 8)
- `-output` (optional): Output HTML file path (default: waveform.html)

### Examples

Find 8 slices in an audio file:
```bash
./slice-analyzer -file song.wav
```

Find 16 slices and save to a custom location:
```bash
./slice-analyzer -file song.wav -slices 16 -output my_slices.html
```

## How It Works

1. **Audio Loading**: The program reads the audio file and extracts only the left channel (or mono channel if the file is mono)
2. **Onset Detection**: Uses the High Frequency Content (HFC) method to detect all onsets, then selects the N strongest ones based on energy
3. **Data Export**: Writes the waveform samples and onset times to a JSON file
4. **Visualization**: Calls a Python script that uses Plotly to create an interactive HTML visualization with the detected onset points marked as red vertical lines

## Notes

- The program may not find exactly the requested number of slices due to the characteristics of the audio
- Different audio files may require different detection parameters for optimal results
- The algorithm focuses on finding the most prominent onsets/transients in the audio
