package alias

//go:generate mockgen -typed -package=mock -destination=mock/interfaces.go . Fooer,FooerAlias,Barer,BarerAlias,Bazer,QuxerConsumer,QuuxerConsumer

import "go.uber.org/mock/mockgen/internal/tests/alias/subpkg"

// Case 1: A interface that has alias references in this package
//         should still be generated for its underlying name, i.e., MockFooer,
//         even though we have the alias replacement logic.
type Fooer interface {
	Foo()
}

// Case 2: Generating a mock for an alias type.
type FooerAlias = Fooer

// Case 3: Generate mock for an interface that takes in alias parameters
//         and returns alias results.
type Barer interface{
	Bar(FooerAlias) FooerAlias
}

// Case 4: Combination of cases 2 & 3.
type BarerAlias = Barer

// Case 5: Generate mock for an interface that actually returns
//         the underlying type. This will generate mocks that use the alias,
//         but that should be fine since they should be interchangeable.
type Bazer interface{
	Baz(Fooer) Fooer
}

// Case 6: Generate mock for a type that refers to an alias defined in this package
//         for a type from another package.
//         The generated methods should use the alias defined here.
type QuxerAlias = subpkg.Quxer

type QuxerConsumer interface{
	Consume(QuxerAlias) QuxerAlias
}

// Case 7: Generate mock for a type that refers to an alias defined in another package
//         for an unexported type in that other package.
//         The generated method should only use the alias, not the unexported underlying name.
type QuuxerConsumer interface{
	Consume(subpkg.Quuxer) subpkg.Quuxer
}
