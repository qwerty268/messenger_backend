package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	model "github.com/go-park-mail-ru/2024_2_EaglesDesigner/main_app/internal/chats/models"
	testingBoilerplate "github.com/go-park-mail-ru/2024_2_EaglesDesigner/main_app/internal/testing_boilerplate"
)

type RepoSuite struct {
	testingBoilerplate.DBSuite
}

func TestStorageSuite(t *testing.T) {
	suite.Run(t, &RepoSuite{DBSuite: testingBoilerplate.NewDBSuite()})
}

func (s *RepoSuite) SetupSuite() {
	s.DBSuite.SetupSuite()
}

func (s *RepoSuite) TearDownTest() {
	// Можно добавить еще таблицы если надо

	_, err := s.DBSuite.DB.Exec("DELETE FROM chat_user")
	s.Require().NoError(err)

	_, err = s.DBSuite.DB.Exec("DELETE FROM chat")
	s.Require().NoError(err)

	_, err = s.DBSuite.DB.Exec("DELETE FROM contact")
	s.Require().NoError(err)

	_, err = s.DBSuite.DB.Exec("DELETE FROM public.user")
	s.Require().NoError(err)
}

func (s *RepoSuite) insertTypes() {
	query := `INSERT INTO public.chat_type (value) VALUES
				('personal'),
				('group'),
				('channel'),
				('branch')`

	_, err := s.DB.Exec(query)
	s.Require().NoError(err)
}

func (s *RepoSuite) insertUserRoles() {
	query := `INSERT INTO  public.user_role ( value) VALUES
				('none'),
				('owner'),
				('admin')`

	_, err := s.DB.Exec(query)
	s.Require().NoError(err)
}

// анти конфликтый барьер, писать под блоком

// дима
func (s *RepoSuite) TestStorage_GetStores() {
	// Arrange.
	ctx := context.Background()
	repo, _ := NewChatRepository(s.DB)

	s.insertTypes()
	s.insertUserRoles()

	expectedType := "1"

	uuidStr := "123e4567-e89b-12d3-a456-426614174000"

	ID, _ := uuid.Parse(uuidStr)

	// Action.
	chatType, err := repo.GetChatType(ctx, ID)

	// Assert.
	s.Require().NoError(err)
	s.Require().Equal(expectedType, chatType)
}

// лев

func (s *RepoSuite) TestStorageSuite_GetCountOfUsersInChat_Success() {
	repo, err := NewChatRepository(s.DB)
	s.Require().NoError(err)

	chat := model.Chat{
		ChatId:            uuid.New(),
		ChatName:          "test",
		ChatType:          "group",
		SendNotifications: true,
	}

	err = repo.CreateNewChat(context.Background(), chat)
	s.Require().NoError(err)
	count, err := repo.GetCountOfUsersInChat(context.Background(), chat.ChatId)
	s.Require().NoError(err)
	s.Require().Equal(0, count)
}

func (s *RepoSuite) TestStorageSuite_GetChatById_NoRows() {
	repo, err := NewChatRepository(s.DB)
	s.Require().NoError(err)

	chat, err := repo.GetChatById(context.Background(), uuid.New())

	s.Require().Equal(model.Chat{}, chat)
	s.Require().ErrorIs(err, sql.ErrNoRows) // Надо бы возвращать ошибку.
}

func (s *RepoSuite) TestStorageSuite_GetChatById_Success() {
	repo, err := NewChatRepository(s.DB)
	s.Require().NoError(err)

	expectedChat := model.Chat{
		ChatId:   uuid.New(),
		ChatName: "test",
		ChatType: "group",
	}

	err = repo.CreateNewChat(context.Background(), expectedChat)
	s.Require().NoError(err)

	chat, err := repo.GetChatById(context.Background(), expectedChat.ChatId)

	s.Require().NoError(err)
	s.Require().Equal(expectedChat, chat)
}

func (s *RepoSuite) TestStorageSuite_GetNameAndAvatar_NoRows() {
	repo, err := NewChatRepository(s.DB)
	s.Require().NoError(err)

	name, avatar, err := repo.GetNameAndAvatar(context.Background(), uuid.New())

	s.Require().ErrorIs(err, sql.ErrNoRows) // А тут все ок) хотя код такой же. чзх.
	s.Require().Equal(name, "")
	s.Require().Equal(avatar, "")
}

// Тут типо для персонального чата берем аватар, но по названибю функции хуйня.
func (s *RepoSuite) TestStorageSuite_GetNameAndAvatar_Success() {
	repo, err := NewChatRepository(s.DB)
	s.Require().NoError(err)

	expectedChat := model.Chat{
		ChatId:    uuid.New(),
		ChatName:  "test",
		ChatType:  "group",
		AvatarURL: "test",
	}
	err = repo.CreateNewChat(context.Background(), expectedChat)
	s.Require().NoError(err)

	name, avatar, err := repo.GetNameAndAvatar(context.Background(), expectedChat.ChatId)

	s.Require().NoError(err)
	s.Require().Equal(expectedChat.ChatName, name)
	s.Require().Equal(expectedChat.AvatarURL, avatar)
}
