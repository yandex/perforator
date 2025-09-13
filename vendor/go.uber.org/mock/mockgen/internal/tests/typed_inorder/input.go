package typed_inorder

//go:generate mockgen -package typed_inorder -source=input.go -destination=mock.go -typed
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
