package cluster_top

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/yandex/perforator/perforator/internal/symbolizer/proxy/services/cluster_top/mocks"
	"github.com/yandex/perforator/perforator/pkg/storage/cluster_top/aggregated"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/lib/pagination"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

func TestMapEntriesWithZeroTotals(t *testing.T) {
	entries := MapEntries(big.NewInt(0), big.NewInt(0), []*aggregated.AggregationValue{
		{Name: "empty", CpuCycles: *big.NewInt(0), CumulativeCpuCycles: *big.NewInt(0)},
		{Name: "nonzero", CpuCycles: *big.NewInt(1), CumulativeCpuCycles: *big.NewInt(2)},
	})
	for _, entry := range entries {
		for _, percent := range []float64{entry.Count.SelfPct, entry.Count.CumulativePct} {
			if math.IsNaN(percent) || math.IsInf(percent, 0) || percent != 0 {
				t.Fatalf("%s: expected zero percent with zero denominator, got %v", entry.Name, percent)
			}
		}
	}
}

func TestClusterTopPagination(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			storage := mocks.NewMockStorage(gomock.NewController(t))
			entries := make([]*aggregated.AggregationValue, count)
			for i := range entries {
				entries[i] = &aggregated.AggregationValue{Name: fmt.Sprint(i)}
			}
			storage.EXPECT().AggregateClusterTop(gomock.Any(), uint32(42), gomock.Any(),
				aggregated.GroupByFunction, util.Pagination{Offset: 2, Limit: 2}, aggregated.SelfTimeSortOrder).
				Return(entries, nil)
			storage.EXPECT().CountTotalSelfCycles(gomock.Any(), uint32(42)).Return(big.NewInt(0), nil)
			resp, err := NewService(xlog.ForTest(t), storage).GetClusterTopAggregatedByFunction(t.Context(), &perforator.ClusterTopRequest{
				Generation: 42,
				Pagination: &pagination.Paginated{Offset: 2, Limit: 1},
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(resp.Instances) != min(count, 1) || resp.HasMore != (count > 1) {
				t.Fatalf("unexpected pagination response: %v", resp)
			}
			if count > 0 && resp.Instances[0].Name != "0" {
				t.Fatalf("pagination discarded the first entry: %v", resp)
			}
		})
	}
}

// oneCpuHourCycles is the minimum CPU cycle count that produces exactly 1.0 cpu-hour
// when passed through fromCpuCyclesToCpuHours. The function uses two integer big.Int
// divisions, so smaller values silently truncate to 0:
//
//	cycles * perforatorSamplingModulo (30)
//	────────────────────────────────── = cpuSeconds  (integer division)
//	     estimatedCpuFreq (2.6e9)
//
//	cpuSeconds / intervalSec (3600) = hours  (integer division)
//
// Solving for cycles == 1 hour: 3600 * 2_600_000_000 / 30 = 312_000_000_000.
const oneCpuHourCycles = int64(312_000_000_000)

// TestFromCpuCyclesToCpuHours directly exercises the cycle → cpu-hour conversion so
// that regressions in the formula (wrong constant, wrong order of operations, etc.)
// are caught independently of the storage layer.
func TestFromCpuCyclesToCpuHours(t *testing.T) {
	cases := []struct {
		name      string
		cyclesN   int64 // multiple of oneCpuHourCycles
		wantHours float64
	}{
		{"zero cycles", 0, 0},
		{"one hour", 1, 1},
		{"five hours", 5, 5},
		{"ten hours", 10, 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cycles := big.NewInt(tc.cyclesN * oneCpuHourCycles)
			got := fromCpuCyclesToCpuHours(cycles)
			if got != tc.wantHours {
				t.Errorf("fromCpuCyclesToCpuHours(%d) = %v, want %v", cycles, got, tc.wantHours)
			}
		})
	}
}

// TestGetClusterTopByFunction tests the "show cluster top with functions" user flow:
// general cluster top, grouped by function, without a search filter.
//
// Key assertions beyond the mock contract:
//   - cpu-hours are computed correctly from cycles
//   - CumulativePct uses totalSelfCycles as denominator (not totalCumulativeCycles,
//     which is never fetched for GroupByFunction)
func TestGetClusterTopByFunction(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := mocks.NewMockStorage(ctrl)

	svc := NewService(xlog.ForTest(t), storageMock)
	ctx := context.Background()

	var generation uint32 = 42

	entry := &aggregated.AggregationValue{Name: "main.expensiveFunction"}
	entry.CpuCycles.SetInt64(3 * oneCpuHourCycles)           // 3 hours
	entry.CumulativeCpuCycles.SetInt64(6 * oneCpuHourCycles) // 6 hours

	totalSelf := big.NewInt(10 * oneCpuHourCycles)

	storageMock.EXPECT().
		AggregateClusterTop(
			gomock.Any(),
			generation,
			gomock.Eq(&aggregated.Filter{
				FunctionFilter:          "",
				FunctionFilterMatchMode: aggregated.SubstringMatch,
			}),
			aggregated.GroupByFunction,
			util.Pagination{Offset: 0, Limit: aggregated.DefaultPageSize + 1},
			aggregated.SelfTimeSortOrder,
		).
		Return([]*aggregated.AggregationValue{entry}, nil)

	// No function-filter option is passed because FunctionFilterMatchMode is SubstringMatch, not ExactMatch.
	// CountTotalCumulativeCycles must NOT be called for GroupByFunction.
	storageMock.EXPECT().
		CountTotalSelfCycles(gomock.Any(), generation).
		Return(totalSelf, nil)

	req := &perforator.ClusterTopRequest{Generation: generation}

	resp, err := svc.GetClusterTopAggregatedByFunction(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(resp.Instances))
	}
	if resp.HasMore {
		t.Error("expected HasMore=false")
	}

	got := resp.Instances[0]
	if got.Name != "main.expensiveFunction" {
		t.Errorf("Name = %q, want %q", got.Name, "main.expensiveFunction")
	}
	if got.Count.Self != 3.0 {
		t.Errorf("Self = %v cpu-hours, want 3.0", got.Count.Self)
	}
	if got.Count.Cumulative != 6.0 {
		t.Errorf("Cumulative = %v cpu-hours, want 6.0", got.Count.Cumulative)
	}
	// For GroupByFunction the cumulative denominator is totalSelfCycles (10 * oneCpuHourCycles).
	// SelfPct = 3/10*100 = 30%, CumulativePct = 6/10*100 = 60%.
	if got.Count.SelfPct != 30.0 {
		t.Errorf("SelfPct = %v, want 30.0", got.Count.SelfPct)
	}
	if got.Count.CumulativePct != 60.0 {
		t.Errorf("CumulativePct = %v, want 60.0 (denominator must be totalSelf, not totalCumulative)", got.Count.CumulativePct)
	}
}

// TestGetClusterTopByServiceForFunction tests the "show biggest consumers of a specific function"
// user flow: exact-match search by function name, grouped by service (dropdown in UI).
//
// Key assertion: CumulativePct must use totalCumulativeCycles (fetched separately for
// GroupByService) rather than totalSelfCycles. The two totals are intentionally different
// so the test fails if the wrong one is chosen.
func TestGetClusterTopByServiceForFunction(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := mocks.NewMockStorage(ctrl)

	svc := NewService(xlog.ForTest(t), storageMock)
	ctx := context.Background()

	var generation uint32 = 42
	functionName := "runtime.mallocgc"

	entry := &aggregated.AggregationValue{Name: "some-service"}
	entry.CpuCycles.SetInt64(3 * oneCpuHourCycles)           // 3 hours
	entry.CumulativeCpuCycles.SetInt64(6 * oneCpuHourCycles) // 6 hours

	// Intentionally different totals: if code picks the wrong denominator for
	// CumulativePct (totalSelf instead of totalCumulative) the percentage will
	// be 60% instead of the expected 50%.
	totalSelf := big.NewInt(10 * oneCpuHourCycles)
	totalCumulative := big.NewInt(12 * oneCpuHourCycles)

	storageMock.EXPECT().
		AggregateClusterTop(
			gomock.Any(),
			generation,
			gomock.Eq(&aggregated.Filter{
				FunctionFilter:          functionName,
				FunctionFilterMatchMode: aggregated.ExactMatch,
			}),
			aggregated.GroupByService,
			util.Pagination{Offset: 0, Limit: aggregated.DefaultPageSize + 1},
			aggregated.SelfTimeSortOrder,
		).
		Return([]*aggregated.AggregationValue{entry}, nil)

	// For ExactMatch, CountTotalSelfCycles receives a WithFunctionFilter option.
	// We match it with gomock.Any() since function values cannot be compared directly.
	storageMock.EXPECT().
		CountTotalSelfCycles(gomock.Any(), generation, gomock.Any()).
		Return(totalSelf, nil)

	storageMock.EXPECT().
		CountTotalCumulativeCycles(gomock.Any(), generation, functionName).
		Return(totalCumulative, nil)

	req := &perforator.ClusterTopRequest{
		Generation:      generation,
		FunctionPattern: &functionName,
	}

	resp, err := svc.GetClusterTopAggregatedByService(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(resp.Instances))
	}
	if resp.HasMore {
		t.Error("expected HasMore=false")
	}

	got := resp.Instances[0]
	if got.Name != "some-service" {
		t.Errorf("Name = %q, want %q", got.Name, "some-service")
	}
	if got.Count.Self != 3.0 {
		t.Errorf("Self = %v cpu-hours, want 3.0", got.Count.Self)
	}
	if got.Count.Cumulative != 6.0 {
		t.Errorf("Cumulative = %v cpu-hours, want 6.0", got.Count.Cumulative)
	}
	// SelfPct uses totalSelf (10 units): 3/10*100 = 30%.
	if got.Count.SelfPct != 30.0 {
		t.Errorf("SelfPct = %v, want 30.0", got.Count.SelfPct)
	}
	// CumulativePct must use totalCumulative (12 units), not totalSelf (10 units):
	// 6/12*100 = 50%.  If totalSelf were used instead: 6/10*100 = 60%.
	if got.Count.CumulativePct != 50.0 {
		t.Errorf("CumulativePct = %v, want 50.0 (denominator must be totalCumulative=12, not totalSelf=10)", got.Count.CumulativePct)
	}
}
