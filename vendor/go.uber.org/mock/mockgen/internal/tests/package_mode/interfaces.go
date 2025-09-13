package package_mode

//go:generate mockgen -typed -package=mock -destination=mock/interfaces.go . Food,Eater,Animal,Human,Primate,Car,Driver,UrbanResident,Farmer,Earth

import (
	"time"

	"go.uber.org/mock/mockgen/internal/tests/package_mode/cars"
	"go.uber.org/mock/mockgen/internal/tests/package_mode/fuel"
)

type Food interface {
	Calories() int
}

type Eater interface {
	Eat(foods ...Food)
}

type Animal interface {
	Eater
	Breathe()
	Sleep(duration time.Duration)
}

type Primate Animal

type Human = Primate

type Car[FuelType fuel.Fuel] interface {
	Brand() string
	FuelTank() cars.FuelTank[FuelType]
	Refuel(fuel FuelType, volume int) error
}

type Driver[FuelType fuel.Fuel, CarType Car[FuelType]] interface {
	Wroom() error
	Drive(car CarType)
}

type UrbanResident interface {
	Human
	Driver[fuel.Gasoline, cars.HyundaiSolaris]
	Do(work *Work) error
	LivesInACity()
}

type Farmer interface {
	Human
	Driver[fuel.Diesel, cars.FordF150]
	Do(work *Work) error
	LivesInAVillage()
}

type Work struct {
	Name string
}

type Counter interface {
	int
}

type HumansCount = int

type Earth interface {
	AddHumans(HumansCount) []Human
	HumanPopulation() HumansCount
}
