# transcript — Transcript File Writers

## OVERVIEW

Writes ASR transcription results to `.txt`, `.srt`, and `.vtt` files alongside recording files. Consumes `asr.Transcript` from the unified ASR interface. T027 / FR-013.

## KEY FUNCTIONS

| Function | Purpose |
|----------|---------|
| `WriteText(path, transcript)` | Plain text with `[HH:MM:SS]` timestamps |
| `WriteSRT(path, transcript)` | SubRip subtitle format (`.srt`) |
| `WriteVTT(path, transcript)` | WebVTT subtitle format (`.vtt`) |
| `WriteAll(basePath, transcript, formats)` | Write multiple formats; basePath has no extension |

## FORMAT DETAILS

- **TXT**: `[HH:MM:SS] segment text` — one line per segment
- **SRT**: Numbered cues, `HH:MM:SS,mmm --> HH:MM:SS,mmm` timestamps, blank-line separated
- **VTT**: `WEBVTT` header, `HH:MM:SS.mmm --> HH:MM:SS.mmm` timestamps

## ATOMIC WRITES

All writes use temp file + `os.Rename` to prevent partial/corrupt output files. Parent directories are created automatically.

## ANTI-PATTERNS

- **Never import ASR backends** — only depends on `asr.Segment` and `asr.Transcript` types
- **No audio processing** — this is text output only
- **No stitching/merging** — deferred to Phase 1.5
- **No streaming** — operates on complete `asr.Transcript` only
