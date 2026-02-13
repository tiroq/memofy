# OBS WebSocket API Contract

**Protocol**: OBS WebSocket v5 (obs-websocket 5.x)  
**Transport**: WebSocket over TCP  
**Default Endpoint**: `ws://localhost:4455`  
**Message Format**: JSON-RPC 2.0

## Connection Flow

### 1. Initial Handshake

**Client → Server**: Connect to WebSocket endpoint

**Server → Client**: Hello message
```json
{
  "op": 0,
  "d": {
    "obsWebSocketVersion": "5.0.0",
    "rpcVersion": 1,
    "authentication": {
      "challenge": "base64-encoded-challenge",
      "salt": "base64-encoded-salt"
    }
  }
}
```

**Client → Server**: Identify message
```json
{
  "op": 1,
  "d": {
    "rpcVersion": 1,
    "authentication": "base64-encoded-auth-response-if-password-set",
    "eventSubscriptions": 33
  }
}
```

Note: `eventSubscriptions: 33` = RecordStateChanged events only (bitmask: 0x00000001 | 0x00000020)

**Server → Client**: Identified message
```json
{
  "op": 2,
  "d": {
    "negotiatedRpcVersion": 1
  }
}
```

---

## Required Operations

### GetRecordStatus

**Purpose**: Query current recording state before making changes

**Request** (op: 6 = Request):
```json
{
  "op": 6,
  "d": {
    "requestType": "GetRecordStatus",
    "requestId": "uuid-or-incrementing-id"
  }
}
```

**Response** (op: 7 = RequestResponse):
```json
{
  "op": 7,
  "d": {
    "requestType": "GetRecordStatus",
    "requestId": "matching-request-id",
    "requestStatus": {
      "result": true,
      "code": 100
    },
    "responseData": {
      "outputActive": true,
      "outputPaused": false,
      "outputTimecode": "00:15:23.450",
      "outputDuration": 923450,
      "outputBytes": 157286400
    }
  }
}
```

**Response Fields**:
- `outputActive`: Boolean, true if currently recording
- `outputPaused`: Boolean, true if recording is paused
- `outputTimecode`: String, current recording timecode
- `outputDuration`: Number, duration in milliseconds
- `outputBytes`: Number, file size in bytes

**Error Response**:
```json
{
  "op": 7,
  "d": {
    "requestType": "GetRecordStatus",
    "requestId": "matching-request-id",
    "requestStatus": {
      "result": false,
      "code": 600,
      "comment": "No recording active"
    }
  }
}
```

**Usage**: Call before StartRecord/StopRecord to verify state

---

### StartRecord

**Purpose**: Begin recording to file

**Request**:
```json
{
  "op": 6,
  "d": {
    "requestType": "StartRecord",
    "requestId": "uuid-or-incrementing-id"
  }
}
```

**Success Response**:
```json
{
  "op": 7,
  "d": {
    "requestType": "StartRecord",
    "requestId": "matching-request-id",
    "requestStatus": {
      "result": true,
      "code": 100
    },
    "responseData": {
      "outputPath": "/Users/user/Videos/2026-02-12 14-30-15.mp4"
    }
  }
}
```

**Response Fields**:
- `outputPath`: String, full path to recording file (OBS-managed naming)

**Error Response** (already recording):
```json
{
  "op": 7,
  "d": {
    "requestType": "StartRecord",
    "requestId": "matching-request-id",
    "requestStatus": {
      "result": false,
      "code": 501,
      "comment": "Recording already active"
    }
  }
}
```

**Usage**: Call only when GetRecordStatus confirms `outputActive: false`

**Note**: OBS controls filename. Application should rename file after recording stops if custom naming needed.

---

### StopRecord

**Purpose**: End current recording

**Request**:
```json
{
  "op": 6,
  "d": {
    "requestType": "StopRecord",
    "requestId": "uuid-or-incrementing-id"
  }
}
```

**Success Response**:
```json
{
  "op": 7,
  "d": {
    "requestType": "StopRecord",
    "requestId": "matching-request-id",
    "requestStatus": {
      "result": true,
      "code": 100
    },
    "responseData": {
      "outputPath": "/Users/user/Videos/2026-02-12 14-30-15.mp4"
    }
  }
}
```

**Response Fields**:
- `outputPath`: String, full path to completed recording file

**Error Response** (not recording):
```json
{
  "op": 7,
  "d": {
    "requestType": "StopRecord",
    "requestId": "matching-request-id",
    "requestStatus": {
      "result": false,
      "code": 502,
      "comment": "Recording not active"
    }
  }
}
```

**Usage**: Call only when GetRecordStatus confirms `outputActive: true`

---

## Events (Optional Subscription)

### RecordStateChanged

**Purpose**: Async notification when recording state changes (useful for detecting external control)

**Event** (op: 5 = Event):
```json
{
  "op": 5,
  "d": {
    "eventType": "RecordStateChanged",
    "eventIntent": 32,
    "eventData": {
      "outputActive": true,
      "outputPath": "/Users/user/Videos/2026-02-12 14-30-15.mp4",
      "outputState": "OBS_WEBSOCKET_OUTPUT_STARTED"
    }
  }
}
```

**Event Data Fields**:
- `outputActive`: Boolean, new recording state
- `outputPath`: String, recording file path
- `outputState`: Enum string
  - `OBS_WEBSOCKET_OUTPUT_STARTING`
  - `OBS_WEBSOCKET_OUTPUT_STARTED`
  - `OBS_WEBSOCKET_OUTPUT_STOPPING`
  - `OBS_WEBSOCKET_OUTPUT_STOPPED`
  - `OBS_WEBSOCKET_OUTPUT_PAUSED`
  - `OBS_WEBSOCKET_OUTPUT_RESUMED`

**Usage**: Update internal state if recording started/stopped externally (manual OBS control)

---

## Error Handling

### Status Codes

| Code | Category | Meaning |
|------|----------|---------|
| 100 | Success | Request succeeded |
| 203 | NoReponse | Request timed out |
| 600 | ResourceNotFound | Requested resource doesn't exist |
| 501 | OutputRunning | Output already active (StartRecord when recording) |
| 502 | OutputNotRunning | Output not active (StopRecord when not recording) |

### Connection Errors

**Scenarios**:
1. **OBS not running**: Connection refused (ECONNREFUSED)
2. **WebSocket disabled**: Connection refused (OBS setting)
3. **Wrong password**: Identified message not received, authentication error
4. **Connection lost during operation**: WebSocket close event

**Handling Strategy**:
- Connection refused → Exponential backoff retry (5s, 10s, 20s, max 60s)
- Authentication failed → Show error, don't retry
- Connection lost → Reconnect with backoff, re-subscribe to events
- Request timeout (>5s no response) → Mark OBS disconnected, attempt reconnect

---

## Implementation Requirements

### Required Functionality

1. **Connection Management**
   - Maintain persistent WebSocket connection
   - Handle reconnection with exponential backoff
   - Track connection state: Disconnected, Connecting, Connected

2. **Authentication**
   - Support password-less connection (most common)
   - Implement SHA-256 challenge-response if password configured
   - Formula: `base64(SHA256(base64(SHA256(password + salt)) + challenge))`

3. **Request/Response Matching**
   - Generate unique request IDs (UUID or incrementing)
   - Match responses to requests via ID
   - Timeout requests after 5 seconds

4. **State Verification**
   - Always call GetRecordStatus before StartRecord/StopRecord
   - Handle "already recording" and "not recording" errors gracefully
   - Update internal state from event subscriptions

5. **Error Recovery**
   - Log all errors with request context
   - Don't spam reconnection attempts (respect backoff)
   - Surface connection state to user (ERROR status)

### Non-Required Functionality

- Scene management (not needed for recording)
- Source control (OBS configuration assumed pre-set)
- Streaming operations (only recording needed)
- Advanced output settings (use OBS defaults)

---

## Testing Scenarios

### Happy Path

1. Connect to OBS → Receive Hello → Send Identify → Receive Identified
2. GetRecordStatus → outputActive: false
3. StartRecord → Success, receive outputPath
4. GetRecordStatus → outputActive: true
5. (Time passes)
6. StopRecord → Success, receive outputPath

### Error Paths

1. **OBS not running**: Connection fails → Retry with backoff
2. **Already recording**: StartRecord → Error 501 → Update state, don't retry
3. **Not recording**: StopRecord → Error 502 → Update state, don't retry
4. **Connection lost during recording**: WebSocket close → Reconnect → GetRecordStatus to verify state
5. **Authentication required**: Need password challenge-response

### Edge Cases

1. **Rapid start/stop**: Ensure GetRecordStatus checks prevent double-start/double-stop
2. **External control**: User starts recording in OBS UI → Event updates internal state
3. **OBS crash during recording**: Connection lost → Reconnect → GetRecordStatus shows recording stopped
4. **Network issue**: Timeout after 5s → Reconnect → Verify state

---

## Example Message Flow

```
Client                          OBS
  |                              |
  |-- WebSocket Connect -------->|
  |<-------- Hello --------------|
  |                              |
  |-- Identify (no auth) ------->|
  |<-------- Identified ---------|
  |                              |
  |-- GetRecordStatus ---------->|
  |<- outputActive: false -------|
  |                              |
  |-- StartRecord -------------->|
  |<- Success (outputPath) ------|
  |<-- RecordStateChanged -------|  (event subscription)
  |    (outputActive: true)      |
  |                              |
  (time passes)
  |                              |
  |-- GetRecordStatus ---------->|
  |<- outputActive: true --------|
  |                              |
  |-- StopRecord --------------->|
  |<- Success (outputPath) ------|
  |<-- RecordStateChanged -------|  (event subscription)
  |    (outputActive: false)     |
  |                              |
```

---

## Reference

- **Official Protocol**: https://github.com/obsproject/obs-websocket/blob/master/docs/generated/protocol.md
- **Request Types**: https://github.com/obsproject/obs-websocket/blob/master/docs/generated/protocol.md#requests
- **Event Types**: https://github.com/obsproject/obs-websocket/blob/master/docs/generated/protocol.md#events
- **OBS Version**: 28.0+ (includes obs-websocket v5 by default)
