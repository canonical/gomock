package users

//go:generate mockgen --destination=mock_test.go --package=users_test go.uber.org/mock/mockgen/internal/tests/mock_in_test_package Finder

type User struct {
	Name string
}

type Finder interface {
	FindUser(name string) User
	Add(u User)
}
