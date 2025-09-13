package cars

import (
	"go.uber.org/mock/mockgen/internal/tests/package_mode/fuel"
)

type FuelTank[FuelType fuel.Fuel] struct {
	Fuel     FuelType
	Capacity int
}

type HyundaiSolaris struct{}

func (HyundaiSolaris) Refuel(fuel.Gasoline, int) error {
	return nil
}

func (HyundaiSolaris) Brand() string {
	return "Hyundai"
}

func (HyundaiSolaris) FuelTank() FuelTank[fuel.Gasoline] {
	return FuelTank[fuel.Gasoline]{
		Fuel:     fuel.Gasoline{},
		Capacity: 50,
	}
}

type FordF150 struct{}

func (FordF150) Brand() string {
	return "Ford"
}

func (FordF150) Refuel(fuel.Diesel, int) error {
	return nil
}

func (FordF150) FuelTank() FuelTank[fuel.Diesel] {
	return FuelTank[fuel.Diesel]{
		Fuel:     fuel.Diesel{},
		Capacity: 136,
	}
}
