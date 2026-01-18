Redis Key Map (Live State)
=========================

Room
----
room:{code} (HASH)
  - surveyId
  - hostId
  - status = LOBBY|ACTIVE|ENDED
  - createdAt
  - settingsJson
  - scopeSummary (short)

room:{code}:players (HASH)
  field: playerId
  value: {"nickname","score","currentKey","followUpsUsed","lastActiveAt"}

room:{code}:lb (ZSET)
  member: playerId
  score: totalScore

room:{code}:events (PUBSUB channel name, e.g. "room:{code}:events")

Host context + pools
-------------------
room:{code}:hostctx (JSON)
  - scopeAnchor: {intentBullets[], constraints[], glossary[]}
  - presentationSummary (short)

room:{code}:q:{Qk}:pool (JSON)
  - clarify[] deepen[] branch[] challenge[] compare[]
Each item is a Question object (prompt, type, rubric, pointsMax, threshold?)

Per-player state
----------------
room:{code}:p:{pid}:q (LIST)
  - questionKeys in order: Q1, Q1.1, Q2...

room:{code}:p:{pid}:current (STRING)
  - current questionKey

room:{code}:p:{pid}:qmap (HASH)
  field: questionKey
  value: Question JSON (needed for follow-ups / overrides)

room:{code}:p:{pid}:closedParents (SET)
  - base question keys whose follow-up chain is closed (skip)

room:{code}:p:{pid}:attempt:{questionKey} (HASH/JSON)
  - draftAnswer
  - submittedAnswer
  - status = DRAFT|SUBMITTED|EVALUATED
  - resolution = SAT|UNSAT|SKIPPED|ABANDONED
  - tries
  - evalSummary (small)
  - updatedAt

Analytics (Level 3–4)
---------------------
room:{code}:memory (JSON)
  - globalThemesTop[] contrasts[] frictionPoints[] recommendedProbes[]

room:{code}:q:{Qk}:profile (JSON/HASH)
  - themeCounts missingCounts misunderstandings[]
  - satCount unsatCount skipCount
  - ratingHist ratingMean ratingMedian ratingVar
  - followupHelpedCount followupTotalCount
  - clusters[] (optional small buckets)

Streams (recommended for eval/jobs)
----------------------------------
answers:stream (STREAM)
  fields: roomCode, playerId, questionKey, type, answerRefId, createdAt
Consumer group: ai-eval-workers
