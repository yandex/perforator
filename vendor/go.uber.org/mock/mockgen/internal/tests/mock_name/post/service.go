package post

import (
	"go.uber.org/mock/mockgen/internal/tests/mock_name/user"
)

type Post struct {
	Title  string
	Body   string
	Author *user.User
}

type Service interface {
	Create(title, body string, author *user.User) (*Post, error)
}
