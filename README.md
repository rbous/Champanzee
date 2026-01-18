# 2026 Champs - Development Setup

This project uses Docker for development to ensure all developers have the same environment.

## Services

- **MongoDB**: Database on port 27017
- **Redis**: Cache on port 6379
- **API**: Go backend on port 8080 (with hot reloading)
- **Web**: Next.js frontend on port 3000 (when added)

## Getting Started

1. Ensure Docker and Docker Compose are installed.

2. Clone the repository and navigate to the project directory.

3. Run the development environment:
   ```bash
   docker-compose up --build
   ```

4. The services will be available at:
   - API: http://localhost:8080
   - MongoDB: localhost:27017
   - Redis: localhost:6379
   - Web: http://localhost:3000 (once frontend is added)

## Development

- The API uses Air for hot reloading, so changes to Go files will automatically rebuild.
- Add your Next.js frontend in the `web/` folder with a proper `package.json`.
- Database data persists in Docker volumes.

## Stopping

To stop the services:
```bash
docker-compose down
```

To stop and remove volumes (reset data):
```bash
docker-compose down -v
```
