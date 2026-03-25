package usecase

import (
	"context"

	"supply-chain-simulator/internal/domain"
)

type RoomService struct {
	roomStore   RoomStore
	idGenerator IDGenerator
	clock       Clock
}

func NewRoomService(roomStore RoomStore, idGenerator IDGenerator, clock Clock) *RoomService {
	return &RoomService{
		roomStore:   roomStore,
		idGenerator: idGenerator,
		clock:       clock,
	}
}

func (s *RoomService) CreateRoom(ctx context.Context, maxWeeks int) (domain.Room, error) {
	roomID, err := s.idGenerator.NewID()
	if err != nil {
		return domain.Room{}, err
	}

	room, err := domain.NewRoom(roomID, maxWeeks, s.clock.Now())
	if err != nil {
		return domain.Room{}, err
	}

	if err := s.roomStore.Save(ctx, room); err != nil {
		return domain.Room{}, err
	}

	return room, nil
}

func (s *RoomService) GetRoom(ctx context.Context, roomID string) (domain.Room, error) {
	return s.roomStore.GetByID(ctx, roomID)
}

func (s *RoomService) JoinRoom(ctx context.Context, roomID, playerName string) (domain.Room, error) {
	room, err := s.roomStore.GetByID(ctx, roomID)
	if err != nil {
		return domain.Room{}, err
	}

	playerID, err := s.idGenerator.NewID()
	if err != nil {
		return domain.Room{}, err
	}

	if err := room.AddPlayer(domain.Player{
		ID:   playerID,
		Name: playerName,
	}, s.clock.Now()); err != nil {
		return domain.Room{}, err
	}

	if err := s.roomStore.Save(ctx, room); err != nil {
		return domain.Room{}, err
	}

	return room, nil
}
