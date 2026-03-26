package redis

import "fmt"

func roomKey(roomID string) string {
	return fmt.Sprintf("room:%s", roomID)
}

func sessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}

func roomSessionKey(roomID string) string {
	return fmt.Sprintf("room_session:%s", roomID)
}

func decisionsKey(roomID string, week int) string {
	return fmt.Sprintf("decisions:%s:%d", roomID, week)
}

func roomEventsChannel(roomID string) string {
	return fmt.Sprintf("room_events:%s", roomID)
}
