# Encryption at Rest (AES-256-GCM)

## What
Symmetric authenticated encryption for sensitive data stored in the database. The `Encryptor` interface abstracts the cipher so domain and service layers never depend on a specific algorithm.

## Why
API keys, tokens, and other secrets must not be stored as plaintext in Postgres. AES-256-GCM provides both confidentiality and integrity — if the ciphertext is tampered with, decryption fails. The interface allows swapping implementations (e.g., KMS-backed) without touching service code.

## Example
```go
// Port (defined in consumer package — workspace service)
type Encryptor interface {
    Encrypt(plaintext []byte) ([]byte, error)
    Decrypt(ciphertext []byte) ([]byte, error)
}

// Implementation (pkg/crypto)
cipher, _ := crypto.NewAES256GCM(keyBytes)  // [32]byte key

enc, _ := cipher.Encrypt([]byte("sk-my-api-key"))
// enc = nonce (12 bytes) || ciphertext || tag (16 bytes)

plain, _ := cipher.Decrypt(enc)
// plain = "sk-my-api-key"
```

The encrypted bytes are stored as `BYTEA` in Postgres. The domain entity treats them as opaque `[]byte` — it never decrypts. Only the service layer calls `Encrypt` (on write) and `Decrypt` (on read).

## Gotchas
- Key must be exactly 32 bytes (256 bits). Env var is hex-encoded (64 chars).
- Each Encrypt call generates a random 12-byte nonce — same plaintext produces different ciphertext every time.
- If the master key is lost, all encrypted data is unrecoverable. No key rotation mechanism yet.
- `NewAES256GCM` returns an error (not a panic) — cipher creation can fail on invalid key length.
- The Encryptor port lives in the service package (consumer-side), not in `pkg/crypto` — follows [[consumer-side-ports]].

## Related
[[consumer-side-ports]], [[config-section-pattern]], [[sqlc-mapper-pattern]]
