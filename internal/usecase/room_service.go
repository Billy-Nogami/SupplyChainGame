package usecase

import (
	"context"
	"log"

	"supply-chain-simulator/internal/domain"
)

type RoomService struct {
	roomStore      RoomStore
	idGenerator    IDGenerator
	clock          Clock
	eventPublisher RoomEventPublisher
}

func NewRoomService(roomStore RoomStore, idGenerator IDGenerator, clock Clock, eventPublisher RoomEventPublisher) *RoomService {
	if eventPublisher == nil {
		eventPublisher = NopRoomEventPublisher{}
	}

	return &RoomService{
		roomStore:      roomStore,
		idGenerator:    idGenerator,
		clock:          clock,
		eventPublisher: eventPublisher,
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
	_ = s.eventPublisher.PublishRoomEvent(ctx, RoomEvent{
		Type:       "room.created",
		RoomID:     room.ID,
		OccurredAt: room.UpdatedAt,
		Room:       &room,
	})
	log.Printf("room_created room_id=%s max_weeks=%d", room.ID, room.MaxWeeks)

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
	_ = s.eventPublisher.PublishRoomEvent(ctx, RoomEvent{
		Type:       "room.player_joined",
		RoomID:     room.ID,
		OccurredAt: room.UpdatedAt,
		Room:       &room,
	})
	log.Printf("room_player_joined room_id=%s player_id=%s player_name=%q players=%d", room.ID, playerID, playerName, len(room.Players))

	return room, nil
}

func (s *RoomService) AssignRole(ctx context.Context, roomID, playerID string, role domain.Role) (domain.Room, error) {
	room, err := s.roomStore.GetByID(ctx, roomID)
	if err != nil {
		return domain.Room{}, err
	}

	if err := room.AssignRole(playerID, role, s.clock.Now()); err != nil {
		return domain.Room{}, err
	}

	if err := s.roomStore.Save(ctx, room); err != nil {
		return domain.Room{}, err
	}
	_ = s.eventPublisher.PublishRoomEvent(ctx, RoomEvent{
		Type:       "room.role_assigned",
		RoomID:     room.ID,
		OccurredAt: room.UpdatedAt,
		Room:       &room,
	})
	log.Printf("room_role_assigned room_id=%s player_id=%s role=%s", room.ID, playerID, role)

	return room, nil
}
