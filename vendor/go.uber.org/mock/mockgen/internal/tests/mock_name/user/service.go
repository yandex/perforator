package user

type User struct {
	Name string
}

type Service interface {
	Create(name string) (*User, error)
}
