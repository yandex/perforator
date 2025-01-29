# orb/quadtree [![Godoc Reference](https://pkg.go.dev/badge/github.com/paulmach/orb)](https://pkg.go.dev/github.com/paulmach/orb/quadtree)

Package `quadtree` implements a quadtree using rectangular partitions.
Each point exists in a unique node. This implementation is based off of the
[d3 implementation](https://github.com/mbostock/d3/wiki/Quadtree-Geom).

## API

```go
func New(bound orb.Bound) *Quadtree
func (q *Quadtree) Bound() orb.Bound

func (q *Quadtree) Add(p orb.Pointer) error
func (q *Quadtree) Remove(p orb.Pointer, eq FilterFunc) bool

func (q *Quadtree) Find(p orb.Point) orb.Pointer
func (q *Quadtree) Matching(p orb.Point, f FilterFunc) orb.Pointer

func (q *Quadtree) KNearest(buf []orb.Pointer, p orb.Point, k int, maxDistance ...float64) []orb.Pointer
func (q *Quadtree) KNearestMatching(buf []orb.Pointer, p orb.Point, k int, f FilterFunc, maxDistance ...float64) []orb.Pointer

func (q *Quadtree) InBound(buf []orb.Pointer, b orb.Bound) []orb.Pointer
func (q *Quadtree) InBoundMatching(buf []orb.Pointer, b orb.Bound, f FilterFunc) []orb.Pointer
```

## Examples

```go
func ExampleQuadtree_Find() {
    r := rand.New(rand.NewSource(42)) // to make things reproducible

    qt := quadtree.New(orb.Bound{Min: orb.Point{0, 0}, Max: orb.Point{1, 1}})

    // add 1000 random points
    for i := 0; i < 1000; i++ {
        qt.Add(orb.Point{r.Float64(), r.Float64()})
    }

    nearest := qt.Find(orb.Point{0.5, 0.5})

    fmt.Printf("nearest: %+v\n", nearest)
    // Output:
    // nearest: [0.4930591659434973 0.5196585530161364]
}
```
