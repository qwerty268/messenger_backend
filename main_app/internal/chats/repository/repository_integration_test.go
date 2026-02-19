//go:build integration
// +build integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

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

	_, err = s.DBSuite.DB.Exec("DELETE FROM chat_type")
	s.Require().NoError(err)

	_, err = s.DBSuite.DB.Exec("DELETE FROM contact")
	s.Require().NoError(err)

	_, err = s.DBSuite.DB.Exec("DELETE FROM public.user")
	s.Require().NoError(err)

	_, err = s.DBSuite.DB.Exec("DELETE FROM user_role")
	s.Require().NoError(err)
}

// func (s *RepoSuite) insertTypes() {
// 	query := `INSERT INTO chat_type (value) VALUES
// 				('personal'),
// 				('group'),
// 				('channel'),
// 				('branch')`

// 	_, err := s.DB.Exec(query)
// 	s.Require().NoError(err)
// }

// func (s *RepoSuite) insertUserRoles() {
// 	query := `INSERT INTO user_role ( value) VALUES
// 				('none'),
// 				('owner'),
// 				('admin')`

// 	_, err := s.DB.Exec(query)
// 	s.Require().NoError(err)
// }

// анти конфликтый барьер, писать под блоком

// дима
func (s *RepoSuite) TestStorage_GetStores() {
	// Arrange.
	ctx := context.Background()
	repo, _ := NewChatRepository(s.DB)

	ID := s.insertDummyStores()

	expectedType := "personal"

	// Action.
	chatType, err := repo.GetChatType(ctx, ID)

	// Assert.
	s.Require().NoError(err)
	s.Require().Equal(expectedType, chatType)
}

func (s *RepoSuite) insertDummyStores() uuid.UUID {
	ID := uuid.New()

	query := `INSERT INTO chat (chat_name, chat_type_id, id) VALUES
				(:chat_name, :chat_type_id, :id)`

	params := map[string]any{
		"id":           ID,
		"chat_name":    "test",
		"chat_type_id": 1,
	}

	_, err := s.DB.NamedExec(query, params)
	s.Require().NoError(err)

	return ID
}

// лев
