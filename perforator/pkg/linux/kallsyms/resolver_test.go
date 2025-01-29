package kallsyms

import (
	"bytes"
	"fmt"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
)

func TestGoodKallsyms(t *testing.T) {
	k, err := NewKallsymsResolver(bytes.NewReader([]byte(
		`0000000000000000 A fixed_percpu_data
0000000000000000 A __per_cpu_start
0000000000001000 A cpu_debug_store
0000000000002000 A irq_stack_backing_store
0000000000006000 A cpu_tss_rw
0000000000009000 A gdt_page
000000000000a000 A exception_stacks
0000000000014000 A entry_stack_storage
0000000000015000 A espfix_waddr
ffffffff81004570 t x86_pmu_del
ffffffff810046a0 T x86_reserve_hardware
ffffffff81004810 t x86_pmu_event_init
ffffffff81004a20 T x86_release_hardware
ffffffff81004ac0 t hw_perf_event_destroy
ffffffff81004ae0 T x86_add_exclusive
ffffffff81004ba0 T x86_del_exclusive
ffffffff81004bd0 T hw_perf_lbr_event_destroy
ffffffff81004bf0 T x86_setup_perfctr
ffffffff81004d20 T x86_pmu_max_precise
ffffffff81004d80 T x86_pmu_hw_config
ffffffff81004fb0 T x86_pmu_disable_all
ffffffffa01d5920 t mlx4_get_vf_indx     [mlx4_core]
ffffffffa01d1480 t mlx4_get_vf_stats    [mlx4_core]
ffffffffa01d5330 t mlx4_report_internal_err_comm_event  [mlx4_core]
ffffffffa01f1a90 t mlx4_qp_roce_entropy [mlx4_core]
	`)))
	require.NoError(t, err)

	require.Equal(t, k.Resolve(0x0000000000000000), "unknown")
	require.Equal(t, k.Resolve(0x0000000000015000), "unknown")
	require.Equal(t, k.Resolve(0x1111111111111111), "unknown")
	require.Equal(t, k.Resolve(0xffffffff8100456f), "unknown")
	require.Equal(t, k.Resolve(0xffffffff81004570), "x86_pmu_del")
	require.Equal(t, k.Resolve(0xffffffff8100469f), "x86_pmu_del")
	require.Equal(t, k.Resolve(0xffffffff810046a0), "x86_reserve_hardware")
	require.Equal(t, k.Resolve(0xffffffffa01f1a90), "mlx4_qp_roce_entropy@[mlx4_core]")
	require.Equal(t, k.Resolve(0xffffffffffffffff), "mlx4_qp_roce_entropy@[mlx4_core]")
}

func TestUnsortedKallsyms(t *testing.T) {
	k, err := NewKallsymsResolver(bytes.NewReader([]byte(`
ffffffffffffffff t f1
1 t f2
fffffffffffffffb t f3
fffffffffffffff0 t f4
	`)))
	require.NoError(t, err)

	require.Equal(t, k.Resolve(0x0000000000000000), "unknown")
	require.Equal(t, k.Resolve(0x0000000000000001), "f2")
	require.Equal(t, k.Resolve(0xffffffffffffffdf), "f2")
	require.Equal(t, k.Resolve(0xfffffffffffffff0), "f4")
	require.Equal(t, k.Resolve(0xfffffffffffffffa), "f4")
	require.Equal(t, k.Resolve(0xfffffffffffffffb), "f3")
}

func TestEmptyKallsyms(t *testing.T) {
	k, err := NewKallsymsResolver(bytes.NewReader([]byte(``)))
	require.NoError(t, err)

	require.Equal(t, k.Resolve(0x0000000000000000), "unknown")
	require.Equal(t, k.Resolve(0x0000000000015000), "unknown")
	require.Equal(t, k.Resolve(0x1111111111111111), "unknown")
	require.Equal(t, k.Resolve(0xffffffff8100456f), "unknown")
	require.Equal(t, k.Resolve(0xffffffffffffffff), "unknown")
}

func TestMalformedKallsyms(t *testing.T) {
	for i, test := range []struct {
		kallsyms string
		error    string
	}{
		{
			kallsyms: `
0000000000000000 A fixed_percpu_data kek kek
ffffffff81004810 t x86_pmu_event_init
ffffffff81004a20 T x86_release_hardware
ffffffff81004ac0 t hw_perf_event_destroy
ffffffff81004ae0 T x86_add_exclusive
ffffffff81004ba0 T x86_del_exclusive
ffffffff81004bd0 T hw_perf_lbr_event_destroy
		`,
			error: `malformed`,
		},
		{
			kallsyms: ``,
			error:    ``,
		},
		{
			kallsyms: `0xffffffff81004810 t x86_pmu_event_init`,
			error:    `failed to parse`,
		},
	} {
		t.Run(fmt.Sprintf("malformed_%d", i), func(t *testing.T) {
			_, err := NewKallsymsResolver(bytes.NewBufferString(test.kallsyms))
			if test.error == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.error)
			}
		})
	}
}

func TestFailingKallsyms(t *testing.T) {
	_, err := NewKallsymsResolver(iotest.ErrReader(fmt.Errorf("kek error")))
	require.ErrorContains(t, err, "kek error")
}
