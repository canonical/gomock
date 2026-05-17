package users_test

type User struct {
	Name string
}

type Finder interface {
	FindUser(name string) User
	Add(u User)
}
