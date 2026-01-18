package app

import (
	"2026champs/internal/cache"
	"2026champs/internal/repository"
)

type App struct {
	PlayerRepo   repository.PlayerRepo
	RoomRepo     repository.RoomRepo
	SessionRepo  repository.SessionRepo
	AnswerRepo   repository.AnswerRepository
	SessionCache cache.SessionCache
}
