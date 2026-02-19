package testing_boilerplate

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Для подключения к postgres.
	"github.com/stretchr/testify/suite"
)

type DBSuite struct {
	suite.Suite

	DB *sqlx.DB
}

func NewDBSuite() DBSuite {
	return DBSuite{}
}

func (s *DBSuite) SetupSuite() {
	cwd, err := os.Getwd()
	s.Require().NoError(err)

	path, err := findEnvFile(cwd)
	s.Require().NoErrorf(err, "could not load .env file: %v", err)

	err = godotenv.Load(path)

	s.Require().NoError(err)

	postgresDSN := os.Getenv("PG_DSN")
	s.Require().NotEmpty(postgresDSN, "postgres dsn not set")

	db, err := sqlx.Connect("postgres", postgresDSN)
	s.Require().NoErrorf(err, "failed sqlx connect: %v", err)

	s.DB = db
}

func (s *DBSuite) TearDownSuite() {
	// Проверяем остатки незакрытых соединений.
	stats := s.DB.DB.Stats()
	s.Zero(stats.InUse)

	// Закрываем подключение к БД.
	s.Require().NoError(s.DB.Close())
}

func findEnvFile(startPath string) (string, error) {
	currentPath := startPath

	for {
		envFilePath := filepath.Join(currentPath, ".env")

		if _, err := os.Stat(envFilePath); err == nil {
			return envFilePath, nil
		}

		parentPath := filepath.Dir(currentPath)
		if currentPath == parentPath {
			break
		}
		currentPath = parentPath
	}

	return "", errors.New(".env file not found")
}
