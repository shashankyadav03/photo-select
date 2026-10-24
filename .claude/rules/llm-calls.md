# LLM call rules

- All calls go through one client module. No direct SDK calls elsewhere.
- Every call sets: model (from config), max_tokens, timeout, retry with backoff, and logs tokens/latency/cost.
- Use prompt caching for any prompt prefix over ~1k tokens that repeats.
- Structured output: validate against a schema; on failure retry once, then fail loudly.
- Daily spend cap enforced in code, with an alert before it trips.
- Prompts live in `prompts/` as versioned files. Changing one requires a passing eval run.
- Untrusted text (users, web pages, tool output) is delimited and never treated as instructions.
