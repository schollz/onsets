#!/usr/bin/env python3
import json
import sys
import plotly.graph_objects as go
from plotly.subplots import make_subplots

def plot_waveform(data_file, output_file):
    # Read JSON data
    with open(data_file, 'r') as f:
        data = json.load(f)

    samples = data['samples']
    sample_rate = data['sample_rate']
    onsets = data['onsets']

    # Create time array
    time = [i / sample_rate for i in range(len(samples))]

    # Create figure
    fig = go.Figure()

    # Add waveform trace
    fig.add_trace(go.Scatter(
        x=time,
        y=samples,
        mode='lines',
        name='Waveform',
        line=dict(color='black', width=1),
        hovertemplate='Time: %{x:.4f}s<br>Amplitude: %{y:.4f}<extra></extra>',
        showlegend=False
    ))

    # Find min and max amplitude for vertical lines
    min_amp = min(samples)
    max_amp = max(samples)

    # Add vertical lines for each onset
    for i, onset in enumerate(onsets):
        fig.add_trace(go.Scatter(
            x=[onset, onset],
            y=[min_amp, max_amp],
            mode='lines',
            name=f'Onset {i+1}',
            line=dict(color='red', width=2),
            hovertemplate=f'Onset {i+1}<br>Time: {onset:.4f}s<extra></extra>',
            showlegend=False
        ))

    # Update layout
    fig.update_layout(
        title='Waveform with Onset Slices',
        xaxis_title='Time (seconds)',
        yaxis_title='Amplitude',
        template='plotly_white',
        hovermode='closest',
        width=1400,
        height=600,
        plot_bgcolor='white',
        paper_bgcolor='white'
    )

    # Save to HTML
    fig.write_html(output_file)
    print(f"Plot saved to {output_file}")

if __name__ == '__main__':
    if len(sys.argv) != 3:
        print("Usage: python3 plot_waveform.py <data_file.json> <output_file.html>")
        sys.exit(1)

    data_file = sys.argv[1]
    output_file = sys.argv[2]

    plot_waveform(data_file, output_file)
