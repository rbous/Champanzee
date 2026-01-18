# Frontend Integration Rules

## Base URL
```
http://localhost:8080/v1
```

---

## Authentication

### Host Login
```http
POST /auth/login
Content-Type: application/json

{"username": "admin", "password": "password123"}
```
Response: `{"token": "eyJ..."}`

Store token in localStorage. Use in all host requests:
```http
Authorization: Bearer <token>
```

### Player Join
```http
POST /rooms/{code}/join
Content-Type: application/json

{"nickname": "PlayerName"}
```
Response contains player token + first question. Store and use for all player requests.

---

## Host Flow

### 1. Create Survey
```http
POST /surveys
Authorization: Bearer <host_token>

{
  "title": "Product Feedback",
  "intent": "Understand user pain points",
  "settings": {"maxFollowUps": 2, "allowSkipAfter": 1},
  "questions": [
    {"key": "Q1", "type": "ESSAY", "prompt": "What frustrates you most?", "rubric": "...", "pointsMax": 100, "threshold": 0.6},
    {"key": "Q2", "type": "DEGREE", "prompt": "Rate your experience", "scaleMin": 1, "scaleMax": 5, "pointsMax": 20}
  ]
}
```

### 2. Create Room
```http
POST /rooms
Authorization: Bearer <host_token>

{"surveyId": "<id>"}
```
Response: `{"roomCode": "ABC123"}`

### 3. Start Room
```http
POST /rooms/{code}/start
Authorization: Bearer <host_token>
```

### 4. Connect WebSocket
```javascript
const ws = new WebSocket(`ws://localhost:8080/v1/ws/rooms/${code}/host?token=${hostToken}`);

ws.onmessage = (e) => {
  const msg = JSON.parse(e.data);
  // msg.type: "player_joined", "player_left", "leaderboard_update", "player_progress_update"
};
```

### 5. End Room & Get Reports
```http
POST /rooms/{code}/end
GET /reports/{code}/snapshot    # Instant dashboard
POST /reports/{code}/ai         # Trigger AI report generation
GET /reports/{code}/ai          # Check status / get report when ready
```

---

## Player Flow

### 1. Join Room
```http
POST /rooms/{code}/join
{"nickname": "Alice"}
```
Response:
```json
{
  "playerId": "p_abc123",
  "token": "eyJ...",
  "firstQuestion": {"key": "Q1", "type": "ESSAY", "prompt": "..."}
}
```

### 2. Connect WebSocket
```javascript
const ws = new WebSocket(`ws://localhost:8080/v1/ws/rooms/${code}/player?token=${playerToken}`);

ws.onmessage = (e) => {
  const msg = JSON.parse(e.data);
  // msg.type: "next_question", "evaluation_result", "error"
};
```

### 3. Get Current Question
```http
GET /rooms/{code}/question/current
Authorization: Bearer <player_token>
```

### 4. Save Draft (optional)
```http
PUT /rooms/{code}/questions/{questionKey}/draft
Authorization: Bearer <player_token>

{"draft": "My partial answer..."}
```

### 5. Submit Answer
```http
POST /rooms/{code}/answers
Authorization: Bearer <player_token>

{
  "questionKey": "Q1",
  "textAnswer": "Full answer here...",
  "clientAttemptId": "uuid-for-idempotency"
}
```
Response:
```json
{
  "status": "EVALUATED",
  "resolution": "SAT",          // or "UNSAT"
  "pointsEarned": 85,
  "evalSummary": "Good specifics, missing example",
  "followUp": null,             // or {key, prompt, type} if UNSAT
  "nextQuestion": {key, prompt} // if SAT
}
```

### 6. Skip Question
```http
POST /rooms/{code}/questions/{questionKey}/skip
Authorization: Bearer <player_token>
```
Skipping closes the follow-up chain for that parent question.

---

## Question Types

| Type | Input | Scoring |
|------|-------|---------|
| `ESSAY` | Text input | AI-evaluated, 0-pointsMax based on quality |
| `DEGREE` | Slider 1-5 | Fixed points (participation) |

---

## WebSocket Message Types

### Host receives:
- `player_joined` → `{playerId}`
- `player_left` → `{playerId}`
- `leaderboard_update` → `{leaderboard: [{playerId, score, rank}]}`
- `player_progress_update` → `{playerId, questionKey, status, resolution}`

### Player receives:
- `next_question` → `{question: {...}}`
- `evaluation_result` → `{resolution, points, summary}`
- `error` → `{message}`

---

## Idempotency

Always include `clientAttemptId` (UUID) in answer submissions. If network fails, retry with same ID - server will dedupe.

---

## Error Handling

All errors return:
```json
{"error": "message here"}
```
Status codes: 400 (bad request), 401 (unauthorized), 404 (not found), 500 (server error)

---

## Environment Variables (Backend)

```bash
MONGO_URI=mongodb://admin:password@mongodb:27017/champsdb?authSource=admin
REDIS_URI=redis:6379
HOST_USERNAME=admin
HOST_PASSWORD=password123
JWT_SECRET=your-secret-key
GEMINI_API_KEY=your-gemini-key
PORT=8080
```
