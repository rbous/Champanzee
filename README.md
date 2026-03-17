# champanzee

real-time multiplayer survey platform — built at [uottahack 8](https://2025.uottahack.ca/), where it placed **2nd out of 40+ teams**.

**[champanzee.tech](https://champanzee.tech)**

## what it does

champanzee turns surveys into live, multiplayer game sessions. hosts create survey-driven rooms, players join in real time, and responses are synced across all participants with sub-200ms latency. after each session, results are automatically pushed to surveymonkey for analysis.

## tech stack

| layer | tech |
|---|---|
| backend | go, redis, websockets |
| frontend | next.js, typescript, react |
| database | mongodb |
| ai | gemini api |
| infra | docker, docker compose |

## architecture

```
┌────────────┐     websockets     ┌────────────┐
│   next.js  │ ◄────────────────► │   go api   │
│   :3000    │                    │   :8080    │
└────────────┘                    └─────┬──────┘
                                       │
                          ┌────────────┼────────────┐
                          │            │            │
                     ┌────▼───┐  ┌────▼───┐  ┌────▼────┐
                     │ mongo  │  │ redis  │  │ gemini  │
                     │ :27017 │  │ :6379  │  │   api   │
                     └────────┘  └────────┘  └─────────┘
```

## key features

- **real-time sync** — websocket-based multiplayer state synchronization supporting 50+ concurrent users per session
- **surveymonkey integration** — bidirectional api integration that auto-creates targeted data collectors from game sessions, cutting manual survey setup by 90%+
- **ai-powered** — gemini api integration for intelligent question generation and response analysis
- **hot reloading** — go backend uses air for instant rebuilds during development

## getting started

```bash
git clone https://github.com/rbous/Champanzee.git
cd Champanzee
cp .env.example .env
docker-compose up --build
```

- frontend: `http://localhost:3000`
- api: `http://localhost:8080`

## team

built by the champanzee team at uottahack 8, january 2026.
