Go Project Layout (Recommended)
==============================

/cmd/api/main.go
/cmd/worker/main.go

/internal
  /config
  /models
  /api/http
  /api/ws
  /services
  /storage/redis
  /storage/mongo
  /ai
  /util

Key service ownership
---------------------
RoomService: CreateRoom, JoinRoom, Start/End, NextQuestion
AnswerService: Draft, SubmitAnswer, Skip, Idempotency, Queue ops
AnalyticsService: Update per-question profiles + room memory
PoolService: Generate/approve/edit follow-up pools
Evaluator (AI): Gemini client + prompt builders + JSON schema enforcement
