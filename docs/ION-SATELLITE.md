## ION Satellite Profile

Defines a remote voice satellite that streams microphone audio to a server
(Home Assistant or other controller) over the ION protocol.

**Status:** Draft

---

## Goals

- Transport-agnostic satellite for Home Assistant-like setups
- Supports wake word, VAD, ASR, and TTS event flows
- Minimal framing and raw PCM audio

---

## Roles

- **Satellite**: captures microphone audio and plays back audio from server.
- **Server**: performs wake/ASR/TTS and controls the session.

---

## Bootstrap

1) Satellite connects over a reliable byte stream (TCP/stdio).
2) Satellite sends `describe`.
3) Server responds with `ready` (PCM format).
4) Satellite sends `satellite.hello` with metadata/capabilities.

### `satellite.hello`

```json
{
  "type": "satellite.hello",
  "name": "kitchen-mic",
  "sample_rate": 16000,
  "channels": 1,
  "format": "s16le",
  "wake": true,
  "vad": true,
  "asr": true,
  "tts": true
}
```

---

## State

The satellite may report state changes:

```json
{ "type": "satellite.state", "state": "idle" }
```

Recommended states: `idle`, `listening`, `streaming`, `speaking`, `error`.

---

## Wake word / VAD

Wake word detection (local or remote) can be reported with:

```json
{ "type": "wake.detected", "name": "ok_nabu" }
{ "type": "wake.reset" }
```

Voice activity detection events:

```json
{ "type": "vad.start" }
{ "type": "vad.stop" }
```

---

## ASR flow

- Satellite sends `asr.start` when it begins streaming audio for recognition.
- Satellite sends raw PCM audio frames (`0x02`).
- Server returns `asr.partial` and `asr.result`.
- Satellite sends `asr.stop` when speech ends.

---

## TTS flow

- Satellite sends `tts.start` (text) to request synthesis, or the server can
  respond to an external request and stream audio.
- Server sends `tts.ready` and then PCM audio frames.
- Server sends `tts.done`.

---

## Audio rules

- Audio frames are raw PCM (`s16le`), as defined by `ready`.
- Microphone audio flows **satellite → server**.
- TTS audio flows **server → satellite**.

---

## Error handling

Any endpoint may send:

```json
{ "type": "error", "message": "reason" }
```

Unknown events MUST be ignored.
