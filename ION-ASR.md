## ION ASR Profile

Defines speech-to-text behavior over ION.

---

## Roles

- Client: sends audio
- Recognizer: emits text

---

## Events

### `asr.start` (client → recognizer)

```json
{ "type": "asr.start", "language": "en" }
```

---

### `asr.stop` (client → recognizer)

```json
{ "type": "asr.stop" }
```

---

### `asr.partial` (recognizer → client)

```json
{ "type": "asr.partial", "text": "hello wor" }
```

---

### `asr.result` (recognizer → client)

```json
{ "type": "asr.result", "text": "hello world" }
```

---

### `asr.error` (recognizer → client)

```json
{ "type": "asr.error", "message": "failure" }
```

---

## Audio

- Audio frames sent between `asr.start` and `asr.stop`
- Audio format defined by `ready`

---

## Completion

After `asr.result`, session is complete.
