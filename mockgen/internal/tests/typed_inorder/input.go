package typed_inorder

//go:generate mockgen -package typed_inorder -destination=mock.go github.com/canonical/gomock/mockgen/internal/tests/typed_inorder Animal
type Animal interface {
	GetSound() string
	Feed(string) error
}

func Interact(a Animal, food string) (string, error) {
	if err := a.Feed(food); err != nil {
		return "", err
	}
	return a.GetSound(), nil
}
