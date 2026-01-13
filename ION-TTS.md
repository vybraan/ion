## ION TTS Profile

Defines text-to-speech behavior over ION.

---

## Roles

- Client: submits text
- Synthesizer: emits audio

---

## Events

### `tts.start` (client → synthesizer)

```json
{
  "type": "tts.start",
  "text": "Hello world",
  "voice": "default",
  "language": "en"
}
```

---

### `tts.ready` (synthesizer → client)

```json
{ "type": "tts.ready" }
```

---

### `tts.done` (synthesizer → client)

```json
{ "type": "tts.done" }
```

---

### `tts.stop` (client → synthesizer)

```json
{ "type": "tts.stop" }
```

---

### `tts.error` (synthesizer → client)

```json
{ "type": "tts.error", "message": "failure" }
```

---

## Audio

- Audio frames sent after `tts.ready`
- Audio stops before `tts.done`
- Audio format defined by `ready`

---

## Cancellation

After `tts.stop`, no further audio MUST be sent.
