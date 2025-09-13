package fuel

type Fuel interface {
	EnergyCapacity() int
	Diesel | Gasoline
}

type Diesel struct{}

func (Diesel) EnergyCapacity() int {
	return 48
}

type Gasoline struct{}

func (Gasoline) EnergyCapacity() int {
	return 46
}
