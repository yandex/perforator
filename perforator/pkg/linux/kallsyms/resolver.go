package kallsyms

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type KallsymsResolver struct {
	addresses []uint64
	symbols   []string
	modules   []string
	sorted    bool
}

func (k *KallsymsResolver) addSymbol(address uint64, name, module string) {
	if k.sorted && len(k.addresses) > 0 && k.addresses[len(k.addresses)-1] >= address {
		k.sorted = false
	}

	k.addresses = append(k.addresses, address)
	k.symbols = append(k.symbols, name)
	k.modules = append(k.modules, module)
}

func (k *KallsymsResolver) build() {
	if k.sorted {
		return
	}
	sortKallsyms(k.addresses, k.symbols, k.modules)
	k.sorted = true
}

// Find symbol name by address of the instruction inside the symbol.
func (k *KallsymsResolver) Resolve(address uint64) string {
	n := len(k.addresses)
	pos := sort.Search(n, func(i int) bool {
		return k.addresses[i] > address
	})

	if pos > n || pos == 0 {
		return "unknown"
	}

	sym := k.symbols[pos-1]
	mod := k.modules[pos-1]

	if mod == "" {
		return sym
	} else {
		return fmt.Sprintf("%s@%s", sym, mod)
	}
}

func (k *KallsymsResolver) LookupSymbolRegex(regex string) ([]string, error) {
	re, err := regexp.Compile(regex)
	if err != nil {
		return nil, err
	}

	symbols := make([]string, 0)
	for _, sym := range k.symbols {
		if re.MatchString(sym) {
			symbols = append(symbols, sym)
		}
	}

	return symbols, nil
}

// Get number of known symbols in the resolver.
// Meaningful for debugging/logging purposes only.
func (k *KallsymsResolver) Size() int {
	return len(k.symbols)
}

func NewKallsymsResolver(r io.Reader) (*KallsymsResolver, error) {
	res := &KallsymsResolver{sorted: true}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}

		if len(fields) != 3 && len(fields) != 4 {
			return nil, fmt.Errorf("malformed kallsyms line %v", fields)
		}

		if fields[1] != "t" && fields[1] != "T" {
			continue
		}

		address, err := strconv.ParseUint(fields[0], 16, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kallsyms symbol %v address: %w", fields, err)
		}

		symbol := fields[2]
		module := ""
		if len(fields) == 4 {
			module = fields[3]
		}

		res.addSymbol(address, symbol, module)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	res.build()

	return res, nil
}

func DefaultKallsymsResolver() (*KallsymsResolver, error) {
	r, err := os.Open("/proc/kallsyms")
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return NewKallsymsResolver(r)
}
