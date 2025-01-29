// Binary citygen generates
package main

import (
	"encoding/json"
	"math/rand"
	"os"

	"github.com/go-faster/city"
)

type Entry struct {
	Input             string
	City32            uint32
	City64            uint64
	City128           city.U128
	City128Seed       city.U128
	ClickHouse64      uint64
	ClickHouse128     city.U128
	ClickHouse128Seed city.U128
}

type Data struct {
	Seed    city.U128
	Entries []Entry
}

type testDataStruct struct {
	in string
}

var testData = []testDataStruct{
	{""},
	{"a"},
	{"ab"},
	{"abc"},
	{"abcd"},
	{"abcde"},
	{"abcdef"},
	{"abcdefg"},
	{"abcdefgh"},
	{"abcdefghi"},
	{"0123456789"},
	{"0123456789 "},
	{"0123456789-0"},
	{"0123456789~01"},
	{"0123456789#012"},
	{"0123456789@0123"},
	{"0123456789'01234"},
	{"0123456789=012345"},
	{"0123456789+0123456"},
	{"0123456789*01234567"},
	{"0123456789&012345678"},
	{"0123456789^0123456789"},
	{"0123456789%0123456789£"},
	{"0123456789$0123456789!0"},
	{"size:  a.out:  bad magic"},
	{"Nepal premier won't resign."},
	{"C is as portable as Stonehedge!!"},
	{"Discard medicine more than two years old."},
	{"I wouldn't marry him with a ten foot pole."},
	{"If the enemy is within range, then so are you."},
	{"The major problem is with sendmail.  -Mark Horton"},
	{"How can you write a big system without C++?  -Paul Glick"},
	{"He who has a shady past knows that nice guys finish last."},
	{"Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave"},
	{"His money is twice tainted: 'taint yours and 'taint mine."},
	{"The days of the digital watch are numbered.  -Tom Stoppard"},
	{"For every action there is an equal and opposite government program."},
	{"You remind me of a TV show, but that's all right: I watch it anyway."},
	{"It's well we cannot hear the screams/That we create in others' dreams."},
	{"Give me a rock, paper and scissors and I will move the world.  CCFestoon"},
	{"It's a tiny change to the code and not completely disgusting. - Bob Manchek"},
	{"There is no reason for any individual to have a computer in their home. -Ken Olsen, 1977"},
	{"Even if I could be Shakespeare, I think I should still choose to be Faraday. - A. Huxley"},
	{"The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule"},
	{"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."},
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ !@#$%^*()_-+=,.?")

func randStr(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func main() {
	rand.Seed(1)
	seed := city.U128{
		Low: uint64(rand.Int63()), High: uint64(rand.Int63()),
	}
	d := Data{
		Seed: seed,
	}
	inputs := []string{
		"Moscow",
		"ClickHouse",
	}

	for _, v := range testData {
		inputs = append(inputs, v.in)
	}

	for i := 0; i < 10; i++ {
		l := rand.Intn(256) + 1
		inputs = append(inputs, randStr(l))
	}

	for _, in := range inputs {
		s := []byte(in)
		e := Entry{
			Input:             in,
			City32:            city.Hash32(s),
			City64:            city.Hash64(s),
			City128:           city.Hash128(s),
			City128Seed:       city.Hash128Seed(s, seed),
			ClickHouse64:      city.CH64(s),
			ClickHouse128:     city.CH128(s),
			ClickHouse128Seed: city.CH128Seed(s, seed),
		}
		d.Entries = append(d.Entries, e)
	}

	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	if err := e.Encode(d); err != nil {
		panic(err)
	}
}
