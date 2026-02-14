# 2026-02-14 — Synthesis LLM Integration

## What was done
- Modified `internal/synthesis/service/service.go` — added 4 cross-context ports (SourceLeafReader, SeedReader, SynthesisLeafCreator, WorkspaceMemberReader) plus their value types. Added `llm.Provider` field. Modified `StartSynthesis` to: load source leaves → load seed → build LLM prompt → call ChatCompletion → parse structured JSON response → create synthesis leaf via adapter → complete workflow. Added helpers: `failWorkflow`, `parseSynthesisResponse`, `buildSynthesisLLMPrompt`. CompleteSynthesis/FailSynthesis remain as manual override endpoints.
- Modified `internal/synthesis/provider.go` — updated Wire() to accept llm.Provider, SourceLeafReader, SeedReader, SynthesisLeafCreator
- Modified `cmd/server/adapters.go` — added 3 cross-context adapters: sourceLeafReaderAdapter (wraps LeafRepository.FindByIDs), seedReaderForSynthesisAdapter (wraps SeedRepository.FindByID), synthesisLeafCreatorAdapter (creates branch + synthesis leaf atomically via db.WithTx)
- Modified `cmd/server/main.go` — added exppostgres and expdomain imports, wired all new adapters into synthesis.Wire()

## Files created
- None

## Files modified
- `internal/synthesis/service/service.go`
- `internal/synthesis/provider.go`
- `cmd/server/adapters.go`
- `cmd/server/main.go`

## Decisions made
- Synchronous LLM call (blocks until response) for MVP simplicity; async can be added later
- If LLM or leaf creation fails, workflow is marked "failed" and returned (not an error) — client sees failure reason in the response
- Seed ID derived from first source leaf's seed
- LLM outputs structured JSON with title/summary/key_points/open_questions/tags
- Synthesis leaf creator adapter follows same branch+leaf transaction pattern as exploration's StartBranch
- ContextAssembler real implementation completed (separate from noop) — was done earlier this session
- noopContextAssembler removed, replaced with real contextAssembler in cmd/server/

## Open questions
- Should synthesis support async processing (goroutine + webhook/event on completion)?
- Should we validate that all source leaves share the same seed?
- Token budget management for large synthesis prompts

## Next steps
- Event subscribers for synthesis.completed / synthesis.failed
- Remaining handler smoke tests for synthesis endpoints
- LLM adapter improvements (token counting, retry logic, model selection)
