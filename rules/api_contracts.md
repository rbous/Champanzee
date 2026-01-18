API Contracts (REST + WebSockets)
=================================

Auth
----
- Host: normal auth (JWT)
- Player: room-scoped token issued at join (JWT or opaque). Claims: roomCode, playerId, exp

Host (REST)
-----------
POST /v1/surveys
  body: {title, intentText, settings, questions[]}
  -> {surveyId}

GET /v1/surveys/{surveyId}
  -> survey

POST /v1/rooms
  body: {surveyId, settingsOverride?, hostContextText?, presentationText?}
  -> {roomCode, roomId}

POST /v1/rooms/{code}/start
POST /v1/rooms/{code}/end

GET /v1/rooms/{code}/leaderboard?top=20

POST /v1/rooms/{code}/ai/pools/generate
PATCH /v1/rooms/{code}/ai/pools/{Qk}

Player (REST)
-------------
POST /v1/rooms/{code}/join
  body: {nickname}
  -> {playerId, token, roomMeta, firstQuestion}

GET /v1/rooms/{code}/question/current
PUT /v1/rooms/{code}/questions/{questionKey}/draft
POST /v1/rooms/{code}/answers
POST /v1/rooms/{code}/questions/{questionKey}/skip

WebSockets
----------
GET /v1/ws/rooms/{code}/host?token=...
GET /v1/ws/rooms/{code}/player?token=...

Envelope:
{ "type": "...", "payload": {...} }

Host WS types:
- room_started, room_ended
- player_joined, player_left
- leaderboard_update
- player_progress_update
- analytics_update

Player WS types:
- next_question
- evaluation_result
- error
- room_ended

Idempotency
-----------
- clientAttemptId unique per submission; server dedupes per (roomCode, playerId, questionKey, clientAttemptId)
