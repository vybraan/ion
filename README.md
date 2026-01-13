## ION

**ION** is a minimal framed binary protocol for **streaming raw PCM audio** with **JSON control events** over a reliable byte stream.

ION defines:

- framing
- control events
- audio payload semantics

ION does **not** define:

- codecs
- compression
- security
- authentication
- transports

---

## Core concepts

- Binary framing
- JSON for control
- Raw PCM for audio
- Transport-agnostic
- Low latency

---

## Frame types

| Type   | Meaning            |
| ------ | ------------------ |
| `0x01` | JSON control event |
| `0x02` | PCM audio payload  |

---

## Typical flow

```
client -> describe
server -> ready
client -> start
server -> audio frames
client -> stop
```

---

## Reference implementation

This repository provides:

- ION framing in Go
- TCP transport
- PipeWire/PulseAudio PCM capture
- Streaming microphone audio

---

## Quick test

TCP:

```sh
go run . --mode=server --transport=tcp --addr :10300
go run . --mode=client --transport=tcp --addr :10300
```

Stdio:

```sh
go run . --mode=server --transport=stdio < /dev/stdin > /dev/stdout
go run . --mode=client --transport=stdio
```

---

## Files

- `SPEC.md` — core protocol specification
- `ION-ASR.md` — ASR profile
- `ION-TTS.md` — TTS profile
