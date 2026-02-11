# Env Var Prefix Nesting

## What
An `EnvReader` scoped to a prefix, with a `WithPrefix` method that returns a new reader with an additional segment. Produces hierarchical env var names from composable parts.

## Why
When multiple config sections read from the environment, prefixes prevent name collisions (`PORT` could mean anything, `CANOPY_SERVER_PORT` is unambiguous). The nesting pattern lets the Loader automatically scope each section without the section knowing its full prefix — it just reads `PORT`, the reader handles the rest.

## Example
```go
root := config.NewEnvReader("CANOPY")          // reads CANOPY_*
server := root.WithPrefix("SERVER")            // reads CANOPY_SERVER_*
port := server.Int("PORT", 8080)               // reads CANOPY_SERVER_PORT
```

Composable: sections don't know their parent prefix.
```go
func (c *ServerConfig) Load(env config.EnvReader) {
    // This section doesn't know if it's CANOPY_SERVER or MYAPP_SERVER.
    // It just reads "PORT" — the caller scoped the reader.
    c.Port = env.Int("PORT", 8080)
}
```

## Gotchas
- Empty prefix reads variables without any prefix — useful for standalone microservices.
- `WithPrefix` creates a new reader, does not mutate the original — safe to reuse.
- Convention: prefixes use UPPER_SNAKE_CASE. Keys within sections also UPPER_SNAKE_CASE.

## Related
[[config-section-pattern]]
