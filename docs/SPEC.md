## ION Protocol Specification

**Version:** 1
**Status:** Draft

---

## 1. Transport

ION requires:

- reliable delivery
- ordered delivery
- full-duplex

Transport boundaries are ignored.

---

## 2. Framing

Every message is a frame.

```
+---------+------+-------------------+
| u8 ver  | u8 t | u32 len (LE)      |
+---------+------+-------------------+
| payload (len bytes)               |
+-----------------------------------+
```

- Version: `0x01`
- Length: payload size in bytes

Unknown versions MUST be rejected.

---

## 3. Frame types

### 3.1 JSON (`0x01`)

- UTF-8 JSON object
- MUST contain `type`

### 3.2 Audio (`0x02`)

- Raw PCM bytes
- No metadata
- Format defined by `ready`

---

## 4. Events

All events are JSON frames.

### `describe` (client → server)

Requests capabilities.

```json
{ "type": "describe" }
```

---

### `ready` (server → client)

Defines audio format.

```json
{
  "type": "ready",
  "protocol": "ion",
  "sample_rate": 16000,
  "channels": 1,
  "format": "s16le"
}
```

Audio frames MUST follow these parameters.

---

### `start` (client → server)

Begins audio streaming.

```json
{ "type": "start" }
```

---

### `stop` (client → server)

Stops audio streaming.

```json
{ "type": "stop" }
```

---

### `error` (either)

Signals protocol error.

```json
{ "type": "error", "message": "reason" }
```

---

## 5. Audio rules

- Audio frames only after `start`
- Audio stops after `stop`
- Chunk size is implementation-defined
- No timestamps or sequencing

---

## 6. State model

```
CONNECT
 -> describe
 <- ready
 -> start
 <- audio*
 -> stop
```

---

## 7. Extensibility

- Unknown events MUST be ignored
- New events MAY be added
- New frame types MAY be added

---

## 8. Security

ION provides no security guarantees.
Security is transport-level.
