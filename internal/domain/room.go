package domain

import (
	"errors"
	"time"
)

type GameStatus string

const (
	GameStatusWaiting  GameStatus = "waiting"
	GameStatusActive   GameStatus = "active"
	GameStatusFinished GameStatus = "finished"
)

type Role string

const (
	RoleFactory     Role = "factory"
	RoleDistributor Role = "distributor"
	RoleWholesaler  Role = "wholesaler"
	RoleRetailer    Role = "retailer"
)

var AllRoles = []Role{
	RoleFactory,
	RoleDistributor,
	RoleWholesaler,
	RoleRetailer,
}

var (
	ErrInvalidMaxWeeks    = errors.New("max weeks must be positive")
	ErrEmptyPlayerName    = errors.New("player name must not be empty")
	ErrRoomFull           = errors.New("room already has 4 players")
	ErrRoomNotFound       = errors.New("room not found")
	ErrPlayerAlreadyIn    = errors.New("player already joined the room")
	ErrCannotJoinStarted  = errors.New("cannot join a started room")
	ErrPlayerNotFound     = errors.New("player not found")
	ErrInvalidRole        = errors.New("invalid role")
	ErrRoleAlreadyTaken   = errors.New("role already taken")
	ErrRoomNotReady       = errors.New("room is not ready to start")
	ErrGameAlreadyStarted = errors.New("game already started")
)

type Player struct {
	ID        string
	Name      string
	Role      Role
	Connected bool
	JoinedAt  time.Time
}

type Room struct {
	ID          string
	Status      GameStatus
	CurrentWeek int
	MaxWeeks    int
	Players     []Player
	ScenarioID  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewRoom(id string, maxWeeks int, now time.Time) (Room, error) {
	if maxWeeks <= 0 {
		return Room{}, ErrInvalidMaxWeeks
	}

	return Room{
		ID:          id,
		Status:      GameStatusWaiting,
		CurrentWeek: 0,
		MaxWeeks:    maxWeeks,
		Players:     make([]Player, 0, len(AllRoles)),
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (r *Room) AddPlayer(player Player, now time.Time) error {
	if player.Name == "" {
		return ErrEmptyPlayerName
	}
	if r.Status != GameStatusWaiting {
		return ErrCannotJoinStarted
	}
	if len(r.Players) >= len(AllRoles) {
		return ErrRoomFull
	}

	for _, existing := range r.Players {
		if existing.Name == player.Name {
			return ErrPlayerAlreadyIn
		}
	}

	player.JoinedAt = now
	player.Connected = true
	r.Players = append(r.Players, player)
	r.UpdatedAt = now

	return nil
}

func (r *Room) AssignRole(playerID string, role Role, now time.Time) error {
	if r.Status != GameStatusWaiting {
		return ErrGameAlreadyStarted
	}
	if !isKnownRole(role) {
		return ErrInvalidRole
	}

	playerIndex := -1
	for i := range r.Players {
		if r.Players[i].ID == playerID {
			playerIndex = i
			continue
		}
		if r.Players[i].Role == role {
			return ErrRoleAlreadyTaken
		}
	}
	if playerIndex == -1 {
		return ErrPlayerNotFound
	}

	r.Players[playerIndex].Role = role
	r.UpdatedAt = now

	return nil
}

func (r *Room) EnsureRolesAssigned(now time.Time) error {
	if len(r.Players) != len(AllRoles) {
		return ErrRoomNotReady
	}

	used := make(map[Role]bool, len(AllRoles))
	for _, player := range r.Players {
		if player.Role == "" {
			continue
		}
		if !isKnownRole(player.Role) {
			return ErrInvalidRole
		}
		if used[player.Role] {
			return ErrRoleAlreadyTaken
		}
		used[player.Role] = true
	}

	availableRoles := make([]Role, 0, len(AllRoles))
	for _, role := range AllRoles {
		if !used[role] {
			availableRoles = append(availableRoles, role)
		}
	}

	nextRole := 0
	for i := range r.Players {
		if r.Players[i].Role != "" {
			continue
		}
		r.Players[i].Role = availableRoles[nextRole]
		nextRole++
	}

	r.UpdatedAt = now

	return nil
}

func (r *Room) Start(now time.Time) error {
	if r.Status != GameStatusWaiting {
		return ErrGameAlreadyStarted
	}
	if len(r.Players) != len(AllRoles) {
		return ErrRoomNotReady
	}
	if err := r.EnsureRolesAssigned(now); err != nil {
		return err
	}

	r.Status = GameStatusActive
	r.CurrentWeek = 1
	r.UpdatedAt = now

	return nil
}

func isKnownRole(role Role) bool {
	for _, existing := range AllRoles {
		if existing == role {
			return true
		}
	}

	return false
}
