package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *ChatRepositoryImpl) GetSendNotificationsForUser(ctx context.Context, chatId uuid.UUID, userId uuid.UUID) (bool, error) {
	var sendNotifications bool

	err := r.db.QueryRow(
		`SELECT 
		send_notifications
		FROM chat_user
		WHERE chat_id = $1 AND user_id = $2`,
		chatId,
		userId,
	).Scan(&sendNotifications)
	if err != nil {
		return false, err
	}
	return sendNotifications, nil
}

func (r *ChatRepositoryImpl) SetChatNotofications(ctx context.Context, chatUUID uuid.UUID, userId uuid.UUID, value bool) error {
	var sendNotifications bool

	err := r.db.QueryRow(
		`UPDATE chat_user SET
		send_notifications = $1
		WHERE chat_id = $2 AND user_id = $3 RETURNING send_notifications`,
		value,
		chatUUID,
		userId,
	).Scan(&sendNotifications)
	if err != nil {
		return err
	}

	if sendNotifications != value {
		return fmt.Errorf("не удалось обновить значение")
	}

	return nil
}
