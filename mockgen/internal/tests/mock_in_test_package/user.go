package users

//go:generate mockgen --destination=mock_test.go --package=users_test github.com/canonical/gomock/mockgen/internal/tests/mock_in_test_package Finder

type User struct {
	Name string
}

type Finder interface {
	FindUser(name string) User
	Add(u User)
}
