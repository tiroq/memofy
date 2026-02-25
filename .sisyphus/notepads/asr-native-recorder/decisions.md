# Decisions — asr-native-recorder

## [2026-02-25] Orchestrator: Active decisions from plan

- AD-1: ASR first — Phase 1 only (no Swift)
- AD-2: Native recorder additive/opt-in when built
- AD-3: ASR internal only — no UI changes
- AD-4: Batch transcription first (live deferred)
- AD-5: recorder.Recorder interface created (done in T025)
- AD-6: ASR config extends detection-rules.json (ASR *ASRConfig `json:"asr,omitempty"`)
- AD-7: No context.Context — use http.Client.Timeout for HTTP, manual timer for exec
