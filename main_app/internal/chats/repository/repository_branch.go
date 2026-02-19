package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/go-park-mail-ru/2024_2_EaglesDesigner/global_utils/logger"
	chatModel "github.com/go-park-mail-ru/2024_2_EaglesDesigner/main_app/internal/chats/models"
)

func (r *ChatRepositoryImpl) AddBranch(ctx context.Context, chatId uuid.UUID, messageID uuid.UUID) (chatModel.AddBranch, error) {
	log := logger.LoggerWithCtx(ctx, logger.Log)

	var branch chatModel.AddBranch
	branch.ID = messageID

	_, err := r.db.Exec(
		`INSERT INTO public.chat 
		(id,
		chat_name,
		chat_type_id
		)
		VALUES ($1, 'branch', (SELECT id FROM public.chat_type WHERE value = 'branch'))`,
		branch.ID,
	)
	if err != nil {
		log.Errorf("Не удалось добавить ветку: %v", err)
		return chatModel.AddBranch{}, err
	}

	_, err = r.db.Exec(
		`UPDATE public.message 
		SET branch_id = $2
		WHERE id = $1;`,
		messageID,
		branch.ID,
	)
	if err != nil {
		log.Errorf("Не удалось привязать ветку к сообщению: %v", err)
		return chatModel.AddBranch{}, err
	}

	log.Debugf("вставка юзеров в ветку %s чата %s", branch.ID.String(), chatId)

	_, err = r.db.Exec(
		`INSERT INTO public.chat_user 
			(id, 
			user_role_id, 
			chat_id, 
			user_id)
		SELECT 
			gen_random_uuid(),
			(SELECT id FROM public.user_role WHERE value = 'none'), 
			$2, 
			cu.user_id 
		FROM public.chat_user cu
		WHERE cu.chat_id = $1;`,
		chatId,
		branch.ID,
	)
	if err != nil {
		log.Errorf("Не удалось добавить пользователей в ветку: %v", err)
		return chatModel.AddBranch{}, err
	}

	_, err = r.db.Exec(
		`INSERT INTO chat_branch VALUES ($1, $2, $3);`,
		uuid.New(),
		chatId,
		branch.ID,
	)
	if err != nil {
		log.Printf("Не удослоь добавить чату ветку: %v", err)
		return chatModel.AddBranch{}, err
	}

	return branch, nil
}

// GetBranchParent находит родительский чат. Рассчет идет из того, что branchId == chatId.
func (r *ChatRepositoryImpl) GetBranchParent(ctx context.Context, branchId uuid.UUID) (uuid.UUID, error) {
	log := logger.LoggerWithCtx(ctx, logger.Log)

	row := r.db.QueryRow(
		`SELECT
		m.chat_id
		FROM message AS m WHERE m.id = $1;`,
		branchId,
	)

	var chatId uuid.UUID

	row.Scan(&chatId)
	log.Println(chatId, "dew")
	return chatId, nil
}
