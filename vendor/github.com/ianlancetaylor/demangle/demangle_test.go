// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package demangle

import (
	"strconv"
	"strings"
	"testing"
)

// Check test cases discovered after the code passed the tests in
// demangle-expected (which are tested by TestExpected).  Some of this
// are cases where we differ from the standard demangler, some we are
// the same but we weren't initially.
func TestDemangler(t *testing.T) {
	var tests = []struct {
		input                 string
		want                  string
		wantNoParams          string
		wantNoTemplateParams  string
		wantNoEnclosingParams string
		wantMinimal           string
	}{
		{
			"_ZNSaIcEC1ERKS_",
			"std::allocator<char>::allocator(std::allocator<char> const&)",
			"std::allocator<char>::allocator",
			"std::allocator::allocator(std::allocator const&)",
			"std::allocator<char>::allocator(std::allocator<char> const&)",
			"std::allocator::allocator",
		},
		{
			"_ZN9__gnu_cxx13stdio_filebufIcSt11char_traitsIcEEC1EP8_IO_FILESt13_Ios_Openmodem",
			"__gnu_cxx::stdio_filebuf<char, std::char_traits<char> >::stdio_filebuf(_IO_FILE*, std::_Ios_Openmode, unsigned long)",
			"__gnu_cxx::stdio_filebuf<char, std::char_traits<char> >::stdio_filebuf",
			"__gnu_cxx::stdio_filebuf::stdio_filebuf(_IO_FILE*, std::_Ios_Openmode, unsigned long)",
			"__gnu_cxx::stdio_filebuf<char, std::char_traits<char> >::stdio_filebuf(_IO_FILE*, std::_Ios_Openmode, unsigned long)",
			"__gnu_cxx::stdio_filebuf::stdio_filebuf",
		},
		{
			"_ZN1n1CcvNS_1DIT_EEI1EEEv",
			"n::C::operator n::D<E><E>()",
			"n::C::operator n::D<E><E>",
			"n::C::operator n::D()",
			"n::C::operator n::D<E><E>()",
			"n::C::operator n::D",
		},
		{
			"_Z1CIvPN1D1E1FIdJEEEdEPN1GILb0ET_T0_T1_E1HEPFS6_S7_S8_EN1H1I1JIS7_E1KENSG_IS8_E1KE",
			"G<false, void, D::E::F<double>*, double>::H* C<void, D::E::F<double>*, double>(void (*)(D::E::F<double>*, double), H::I::J<D::E::F<double>*>::K, H::I::J<double>::K)",
			"C<void, D::E::F<double>*, double>",
			"G::H* C(void (*)(D::E::F*, double), H::I::J::K, H::I::J::K)",
			"G<false, void, D::E::F<double>*, double>::H* C<void, D::E::F<double>*, double>(void (*)(D::E::F<double>*, double), H::I::J<D::E::F<double>*>::K, H::I::J<double>::K)",
			"C",
		},
		{
			"_ZZNK1CI1DIcSt1EIcESaIcEEJEE1FEvE1F",
			"C<D<char, std::E<char>, std::allocator<char> > >::F() const::F",
			"C<D<char, std::E<char>, std::allocator<char> > >::F() const::F",
			"C::F() const::F",
			"C<D<char, std::E<char>, std::allocator<char> > >::F() const::F",
			"C::F() const::F",
		},
		{
			"_ZN1CI1DSt1EIK1FN1G1HEEE1I1JIJRKS6_EEEvDpOT_",
			"void C<D, std::E<F const, G::H> >::I::J<std::E<F const, G::H> const&>(std::E<F const, G::H> const&)",
			"C<D, std::E<F const, G::H> >::I::J<std::E<F const, G::H> const&>",
			"void C::I::J(std::E const&)",
			"void C<D, std::E<F const, G::H> >::I::J<std::E<F const, G::H> const&>(std::E<F const, G::H> const&)",
			"C::I::J",
		},
		{
			"_ZN1C1D1E1FIJEEEvi1GDpT_",
			"void C::D::E::F<>(int, G)",
			"C::D::E::F<>",
			"void C::D::E::F(int, G)",
			"void C::D::E::F<>(int, G)",
			"C::D::E::F",
		},
		{
			"_ZN1CILj50ELb1EE1DEv",
			"C<50u, true>::D()",
			"C<50u, true>::D",
			"C::D()",
			"C<50u, true>::D()",
			"C::D",
		},
		{
			"_ZN1CUt_C2Ev",
			"C::{unnamed type#1}::{unnamed type#1}()",
			"C::{unnamed type#1}::{unnamed type#1}",
			"C::{unnamed type#1}::{unnamed type#1}()",
			"C::{unnamed type#1}::{unnamed type#1}()",
			"C::{unnamed type#1}::{unnamed type#1}",
		},
		{
			"_ZN1C12_GLOBAL__N_11DINS_1EEEEN1F1GIDTadcldtcvT__E1HEEEERKS5_NS_1I1JE",
			"F::G<decltype (&((((C::E)()).H)()))> C::(anonymous namespace)::D<C::E>(C::E const&, C::I::J)",
			"C::(anonymous namespace)::D<C::E>",
			"F::G C::(anonymous namespace)::D(C::E const&, C::I::J)",
			"F::G<decltype (&((((C::E)()).H)()))> C::(anonymous namespace)::D<C::E>(C::E const&, C::I::J)",
			"C::(anonymous namespace)::D",
		},
		{
			"_ZN1CI1DE1EIJiRiRPKcRA1_S4_S8_bS6_S3_RjRPKN1F1GERPKN1H1IEEEEvDpOT_",
			"void C<D>::E<int, int&, char const*&, char const (&) [1], char const (&) [1], bool, char const*&, int&, unsigned int&, F::G const*&, H::I const*&>(int&&, int&, char const*&, char const (&) [1], char const (&) [1], bool&&, char const*&, int&, unsigned int&, F::G const*&, H::I const*&)",
			"C<D>::E<int, int&, char const*&, char const (&) [1], char const (&) [1], bool, char const*&, int&, unsigned int&, F::G const*&, H::I const*&>",
			"void C::E(int&&, int&, char const*&, char const (&) [1], char const (&) [1], bool&&, char const*&, int&, unsigned int&, F::G const*&, H::I const*&)",
			"void C<D>::E<int, int&, char const*&, char const (&) [1], char const (&) [1], bool, char const*&, int&, unsigned int&, F::G const*&, H::I const*&>(int&&, int&, char const*&, char const (&) [1], char const (&) [1], bool&&, char const*&, int&, unsigned int&, F::G const*&, H::I const*&)",
			"C::E",
		},
		{
			"_ZN1C12_GLOBAL__N_11DIFbPKNS_1EEEEEvPNS_1FERKT_",
			"void C::(anonymous namespace)::D<bool (C::E const*)>(C::F*, bool (&)(C::E const*) const)",
			"C::(anonymous namespace)::D<bool (C::E const*)>",
			"void C::(anonymous namespace)::D(C::F*, bool (&)(C::E const*) const)",
			"void C::(anonymous namespace)::D<bool (C::E const*)>(C::F*, bool (&)(C::E const*) const)",
			"C::(anonymous namespace)::D",
		},
		{
			"_ZN1C1D1EIJRFviSt1FIFvRKN1G1H1IEEERKSt6vectorINS_1JESaISB_EEERiS9_EvEENS0_1K1LIJDpNSt1MIT_E1NEEEEDpOSM_",
			"C::D::K::L<std::M<void (&)(int, std::F<void (G::H::I const&)>, std::vector<C::J, std::allocator<C::J> > const&)>::N, std::M<int&>::N, std::M<std::F<void (G::H::I const&)> >::N> C::D::E<void (&)(int, std::F<void (G::H::I const&)>, std::vector<C::J, std::allocator<C::J> > const&), int&, std::F<void (G::H::I const&)>, void>(void (&)(int, std::F<void (G::H::I const&)>, std::vector<C::J, std::allocator<C::J> > const&), int&, std::F<void (G::H::I const&)>&&)",
			"C::D::E<void (&)(int, std::F<void (G::H::I const&)>, std::vector<C::J, std::allocator<C::J> > const&), int&, std::F<void (G::H::I const&)>, void>",
			"C::D::K::L C::D::E(void (&)(int, std::F, std::vector const&), int&, std::F&&)",
			"C::D::K::L<std::M<void (&)(int, std::F<void (G::H::I const&)>, std::vector<C::J, std::allocator<C::J> > const&)>::N, std::M<int&>::N, std::M<std::F<void (G::H::I const&)> >::N> C::D::E<void (&)(int, std::F<void (G::H::I const&)>, std::vector<C::J, std::allocator<C::J> > const&), int&, std::F<void (G::H::I const&)>, void>(void (&)(int, std::F<void (G::H::I const&)>, std::vector<C::J, std::allocator<C::J> > const&), int&, std::F<void (G::H::I const&)>&&)",
			"C::D::E",
		},
		{
			"_ZN1C1D1E1FcvNS_1GIT_EEI1HEEv",
			"C::D::E::F::operator C::G<H><H>()",
			"C::D::E::F::operator C::G<H><H>",
			"C::D::E::F::operator C::G()",
			"C::D::E::F::operator C::G<H><H>()",
			"C::D::E::F::operator C::G",
		},
		{
			"_ZN9__gnu_cxx17__normal_iteratorIPK1EIN1F1G1HEESt6vectorIS5_SaIS5_EEEC2IPS5_EERKNS0_IT_NS_11__enable_ifIXsr3std10__are_sameISE_SD_EE7__valueESA_E1IEEE",
			"__gnu_cxx::__normal_iterator<E<F::G::H> const*, std::vector<E<F::G::H>, std::allocator<E<F::G::H> > > >::__normal_iterator<E<F::G::H>*>(__gnu_cxx::__normal_iterator<E<F::G::H>*, __gnu_cxx::__enable_if<std::__are_same<E<F::G::H>*, E<F::G::H>*>::__value, std::vector<E<F::G::H>, std::allocator<E<F::G::H> > > >::I> const&)",
			"__gnu_cxx::__normal_iterator<E<F::G::H> const*, std::vector<E<F::G::H>, std::allocator<E<F::G::H> > > >::__normal_iterator<E<F::G::H>*>",
			"__gnu_cxx::__normal_iterator::__normal_iterator(__gnu_cxx::__normal_iterator const&)",
			"__gnu_cxx::__normal_iterator<E<F::G::H> const*, std::vector<E<F::G::H>, std::allocator<E<F::G::H> > > >::__normal_iterator<E<F::G::H>*>(__gnu_cxx::__normal_iterator<E<F::G::H>*, __gnu_cxx::__enable_if<std::__are_same<E<F::G::H>*, E<F::G::H>*>::__value, std::vector<E<F::G::H>, std::allocator<E<F::G::H> > > >::I> const&)",
			"__gnu_cxx::__normal_iterator::__normal_iterator",
		},
		{
			"_ZNKSt1CIM1DKFjvEEclIJEvEEjPKS0_DpOT_",
			"unsigned int std::C<unsigned int (D::*)() const>::operator()<void>(D const*) const",
			"std::C<unsigned int (D::*)() const>::operator()<void>",
			"unsigned int std::C::operator()(D const*) const",
			"unsigned int std::C<unsigned int (D::*)() const>::operator()<void>(D const*) const",
			"std::C::operator()",
		},
		{
			"_ZNSt10_HashtableI12basic_stringIcSt11char_traitsIcESaIcEESt4pairIKS4_N1C1D1EEESaISA_ENSt8__detail10_Select1stESt8equal_toIS4_ESt4hashIS4_ENSC_18_Mod_range_hashingENSC_20_Default_ranged_hashENSC_20_Prime_rehash_policyENSC_17_Hashtable_traitsILb1ELb0ELb1EEEE9_M_assignIZNSN_C1ERKSN_EUlPKNSC_10_Hash_nodeISA_Lb1EEEE_EEvSQ_RKT_",
			"void std::_Hashtable<basic_string<char, std::char_traits<char>, std::allocator<char> >, std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, std::allocator<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E> >, std::__detail::_Select1st, std::equal_to<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::hash<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::__detail::_Mod_range_hashing, std::__detail::_Default_ranged_hash, std::__detail::_Prime_rehash_policy, std::__detail::_Hashtable_traits<true, false, true> >::_M_assign<std::_Hashtable<basic_string<char, std::char_traits<char>, std::allocator<char> >, std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, std::allocator<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E> >, std::__detail::_Select1st, std::equal_to<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::hash<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::__detail::_Mod_range_hashing, std::__detail::_Default_ranged_hash, std::__detail::_Prime_rehash_policy, std::__detail::_Hashtable_traits<true, false, true> >::_Hashtable(std::_Hashtable<basic_string<char, std::char_traits<char>, std::allocator<char> >, std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, std::allocator<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E> >, std::__detail::_Select1st, std::equal_to<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::hash<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::__detail::_Mod_range_hashing, std::__detail::_Default_ranged_hash, std::__detail::_Prime_rehash_policy, std::__detail::_Hashtable_traits<true, false, true> > const&)::{lambda(std::__detail::_Hash_node<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, true> const*)#1}>(std::_Hashtable<basic_string<char, std::char_traits<char>, std::allocator<char> >, std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, std::allocator<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E> >, std::__detail::_Select1st, std::equal_to<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::hash<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::__detail::_Mod_range_hashing, std::__detail::_Default_ranged_hash, std::__detail::_Prime_rehash_policy, std::__detail::_Hashtable_traits<true, false, true> > const&, std::_Hashtable<basic_string<char, std::char_traits<char>, std::allocator<char> >, std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, std::allocator<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E> >, std::__detail::_Select1st, std::equal_to<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::hash<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::__detail::_Mod_range_hashing, std::__detail::_Default_ranged_hash, std::__detail::_Prime_rehash_policy, std::__detail::_Hashtable_traits<true, false, true> >::_Hashtable(std::_Hashtable<basic_string<char, std::char_traits<char>, std::allocator<char> >, std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, std::allocator<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E> >, std::__detail::_Select1st, std::equal_to<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::hash<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::__detail::_Mod_range_hashing, std::__detail::_Default_ranged_hash, std::__detail::_Prime_rehash_policy, std::__detail::_Hashtable_traits<true, false, true> > const&)::{lambda(std::__detail::_Hash_node<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, true> const*)#1} const&)",
			"std::_Hashtable<basic_string<char, std::char_traits<char>, std::allocator<char> >, std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, std::allocator<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E> >, std::__detail::_Select1st, std::equal_to<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::hash<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::__detail::_Mod_range_hashing, std::__detail::_Default_ranged_hash, std::__detail::_Prime_rehash_policy, std::__detail::_Hashtable_traits<true, false, true> >::_M_assign<std::_Hashtable<basic_string<char, std::char_traits<char>, std::allocator<char> >, std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, std::allocator<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E> >, std::__detail::_Select1st, std::equal_to<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::hash<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::__detail::_Mod_range_hashing, std::__detail::_Default_ranged_hash, std::__detail::_Prime_rehash_policy, std::__detail::_Hashtable_traits<true, false, true> >::_Hashtable(std::_Hashtable<basic_string<char, std::char_traits<char>, std::allocator<char> >, std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, std::allocator<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E> >, std::__detail::_Select1st, std::equal_to<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::hash<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::__detail::_Mod_range_hashing, std::__detail::_Default_ranged_hash, std::__detail::_Prime_rehash_policy, std::__detail::_Hashtable_traits<true, false, true> > const&)::{lambda(std::__detail::_Hash_node<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, true> const*)#1}>",
			"void std::_Hashtable::_M_assign(std::_Hashtable const&, std::_Hashtable::_Hashtable(std::_Hashtable const&)::{lambda(std::__detail::_Hash_node const*)#1} const&)",
			"void std::_Hashtable<basic_string<char, std::char_traits<char>, std::allocator<char> >, std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, std::allocator<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E> >, std::__detail::_Select1st, std::equal_to<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::hash<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::__detail::_Mod_range_hashing, std::__detail::_Default_ranged_hash, std::__detail::_Prime_rehash_policy, std::__detail::_Hashtable_traits<true, false, true> >::_M_assign<std::_Hashtable<basic_string<char, std::char_traits<char>, std::allocator<char> >, std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, std::allocator<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E> >, std::__detail::_Select1st, std::equal_to<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::hash<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::__detail::_Mod_range_hashing, std::__detail::_Default_ranged_hash, std::__detail::_Prime_rehash_policy, std::__detail::_Hashtable_traits<true, false, true> >::_Hashtable()::{lambda(std::__detail::_Hash_node<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, true> const*)#1}>(std::_Hashtable<basic_string<char, std::char_traits<char>, std::allocator<char> >, std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, std::allocator<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E> >, std::__detail::_Select1st, std::equal_to<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::hash<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::__detail::_Mod_range_hashing, std::__detail::_Default_ranged_hash, std::__detail::_Prime_rehash_policy, std::__detail::_Hashtable_traits<true, false, true> > const&, std::_Hashtable<basic_string<char, std::char_traits<char>, std::allocator<char> >, std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, std::allocator<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E> >, std::__detail::_Select1st, std::equal_to<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::hash<basic_string<char, std::char_traits<char>, std::allocator<char> > >, std::__detail::_Mod_range_hashing, std::__detail::_Default_ranged_hash, std::__detail::_Prime_rehash_policy, std::__detail::_Hashtable_traits<true, false, true> >::_Hashtable()::{lambda(std::__detail::_Hash_node<std::pair<basic_string<char, std::char_traits<char>, std::allocator<char> > const, C::D::E>, true> const*)#1} const&)",
			"std::_Hashtable::_M_assign",
		},
		{
			"_ZSt3maxIVdERKT_S3_S3_",
			"double const volatile& std::max<double volatile>(double const volatile&, double const volatile&)",
			"std::max<double volatile>",
			"double const volatile& std::max(double const volatile&, double const volatile&)",
			"double const volatile& std::max<double volatile>(double const volatile&, double const volatile&)",
			"std::max",
		},
		{
			"_ZZN1C1D1E1F1G1HEvENUlvE_C2EOS4_",
			"C::D::E::F::G::H()::{lambda()#1}::{lambda()#1}({lambda()#1}&&)",
			"C::D::E::F::G::H()::{lambda()#1}::{lambda()#1}",
			"C::D::E::F::G::H()::{lambda()#1}::{lambda()#1}({lambda()#1}&&)",
			"C::D::E::F::G::H()::{lambda()#1}::{lambda()#1}({lambda()#1}&&)",
			"C::D::E::F::G::H()::{lambda()#1}::{lambda()#1}",
		},
		{
			"_ZThn8_NK1C1D1EEv",
			"non-virtual thunk to C::D::E() const",
			"non-virtual thunk to C::D::E() const",
			"non-virtual thunk to C::D::E() const",
			"non-virtual thunk to C::D::E() const",
			"non-virtual thunk to C::D::E() const",
		},
		{
			"_ZTv0_n96_NK1C1D1E1FEv",
			"virtual thunk to C::D::E::F() const",
			"virtual thunk to C::D::E::F() const",
			"virtual thunk to C::D::E::F() const",
			"virtual thunk to C::D::E::F() const",
			"virtual thunk to C::D::E::F() const",
		},
		{
			"_ZTCSt9strstream16_So",
			"construction vtable for std::ostream-in-std::strstream",
			"construction vtable for std::ostream-in-std::strstream",
			"construction vtable for std::ostream-in-std::strstream",
			"construction vtable for std::ostream-in-std::strstream",
			"construction vtable for std::ostream-in-std::strstream",
		},
		{
			"_ZGVZZN1C1D1EEvENK3$_0clEvE1F",
			"guard variable for C::D::E()::$_0::operator()() const::F",
			"guard variable for C::D::E()::$_0::operator()() const::F",
			"guard variable for C::D::E()::$_0::operator()() const::F",
			"guard variable for C::D::E()::$_0::operator()() const::F",
			"guard variable for C::D::E()::$_0::operator()() const::F",
		},
		{
			"_Z1fICiEvT_",
			"void f<int _Complex>(int _Complex)",
			"f<int _Complex>",
			"void f(int _Complex)",
			"void f<int _Complex>(int _Complex)",
			"f",
		},
		{
			"_GLOBAL__D__Z2fnv",
			"global destructors keyed to fn()",
			"global destructors keyed to fn()",
			"global destructors keyed to fn()",
			"global destructors keyed to fn()",
			"global destructors keyed to fn()",
		},
		{
			"_Z1fIXadL_Z1hvEEEvv",
			"void f<&h>()",
			"f<&h>",
			"void f()",
			"void f<&h>()",
			"f",
		},
		{
			"_Z1CIP1DEiRK1EPT_N1F1GIS5_Xaasr1HIS5_E1IntsrSA_1JEE1KE",
			"int C<D*>(E const&, D**, F::G<D*, H<D*>::I&&(!H::J)>::K)",
			"C<D*>",
			"int C(E const&, D**, F::G::K)",
			"int C<D*>(E const&, D**, F::G<D*, H<D*>::I&&(!H::J)>::K)",
			"C",
		},
		{
			"_ZNO1A1B1C1DIZN1E1F1GINS3_1HE1IEEvMNS3_1JEFvP1LPKT_PT0_P1KESD_SA_SF_SH_EUlvE_Lb0EEcvPSB_ISG_vvEEv",
			"A::B::C::D<E::F::G<E::H, I>(void (E::J::*)(L*, E::H const*, I*, K*), E::H const*, L*, I*, K*)::{lambda()#1}, false>::operator K*<K, void, void>() &&",
			"A::B::C::D<E::F::G<E::H, I>(void (E::J::*)(L*, E::H const*, I*, K*), E::H const*, L*, I*, K*)::{lambda()#1}, false>::operator K*<K, void, void>",
			"A::B::C::D::operator K*() &&",
			"A::B::C::D<E::F::G<E::H, I>()::{lambda()#1}, false>::operator K*<K, void, void>() &&",
			"A::B::C::D::operator K*",
		},
		{
			"_ZNSt1AIFSt1BImjEjEZN1C1DI1EEENSt1FIXeqsr1G1H1IIDTadsrT_onclEEE1JLi2EEvE1KEPKcSC_OS7_EUljE_E1KERKSt1Lj",
			"std::A<std::B<unsigned long, unsigned int> (unsigned int), C::D<E>(char const*, char const, I&&)::{lambda(unsigned int)#1}>::K(std::L const&, unsigned int)",
			"std::A<std::B<unsigned long, unsigned int> (unsigned int), C::D<E>(char const*, char const, I&&)::{lambda(unsigned int)#1}>::K",
			"std::A::K(std::L const&, unsigned int)",
			"std::A<std::B<unsigned long, unsigned int> (unsigned int), C::D<E>()::{lambda(unsigned int)#1}>::K(std::L const&, unsigned int)",
			"std::A::K",
		},
		{
			"_ZNSt1A1BIiNS_1CIiEEE1DIPiEENS_1EIXaasr1FIT_EE1Gsr1HIiNS_1IIS7_E1JEEE1KEvE1LES7_S7_",
			"std::A::E<F<int*>::G&&H<int, std::A::I<F>::J>::K, void>::L std::A::B<int, std::A::C<int> >::D<int*>(F, F)",
			"std::A::B<int, std::A::C<int> >::D<int*>",
			"std::A::E::L std::A::B::D(F, F)",
			"std::A::E<F<int*>::G&&H<int, std::A::I<F>::J>::K, void>::L std::A::B<int, std::A::C<int> >::D<int*>(F, F)",
			"std::A::B::D",
		},
		{
			"_ZNO1A1B1C1DIJOZZN1E1F1GINS4_1HINS4_1IINS4_1JEEEEEJNS4_1KEEEEN1L1MINS4_1OINT_1PEEEEERKSt6vectorIN1Q1RESaISL_EERKN3gtl1S1TIN1U1VEEERKNS4_1W1XERKNS4_1YERKNSQ_1ZINS4_1aEEEPSt13unordered_mapISL_NSK_9UniquePtrINS4_1bINS0_1cIJS9_NS7_INST_1dEEEEEENS4_1fEEEEENSC_1g1hIvEESt8equal_toISL_ESaISt4pairIKSL_S1J_EEEDpRKT0_ENKUlSL_mmS1G_E_clESL_mmS1G_EUlS9_E_OZZNS5_ISA_JSB_EEESI_SP_SX_S11_S14_S19_S1U_S1Y_ENKS1Z_clESL_mmS1G_EUlS1F_E0_EEclIJRS9_EEEDTclcl1iIXsrNS1_1jISt5tupleIJNS1_1kIS21_EENS29_IS23_EEEEJDpT_EEE1lEEcl1mIS2C_EEEspcl1mIS2D_EEEEDpOS2D_",
			"decltype (((i<A::B::C::j<std::tuple<A::B::C::k<E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>(std::vector<Q::R, std::allocator<Q::R> > const&, gtl::S::T<U::V> const&, E::F::W::X const&, E::F::Y const&, gtl::Z<E::F::a> const&, std::unordered_map<Q::R, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> >, L::g::h<void>, std::equal_to<Q::R>, std::allocator<std::pair<Q::R const, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> > > > >*, E::F::K const&)::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >) const::{lambda(E::F::I<E::F::J>)#1}&&>, A::B::C::k<E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>(std::vector<Q::R, std::allocator<Q::R> > const&, gtl::S::T<U::V> const&, E::F::W::X const&, E::F::Y const&, gtl::Z<E::F::a> const&, std::unordered_map<Q::R, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> >, L::g::h<void>, std::equal_to<Q::R>, std::allocator<std::pair<Q::R const, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> > > > >*, E::F::K const&)::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >) const::{lambda(E::F::I<U::d>)#2}&&> >, E::F::I<E::F::J>&>::l>)((m<std::tuple<A::B::C::k<E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>(std::vector<Q::R, std::allocator<Q::R> > const&, gtl::S::T<U::V> const&, E::F::W::X const&, E::F::Y const&, gtl::Z<E::F::a> const&, std::unordered_map<Q::R, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> >, L::g::h<void>, std::equal_to<Q::R>, std::allocator<std::pair<Q::R const, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> > > > >*, E::F::K const&)::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >) const::{lambda(E::F::I<E::F::J>)#1}&&>, A::B::C::k<E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>(std::vector<Q::R, std::allocator<Q::R> > const&, gtl::S::T<U::V> const&, E::F::W::X const&, E::F::Y const&, gtl::Z<E::F::a> const&, std::unordered_map<Q::R, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> >, L::g::h<void>, std::equal_to<Q::R>, std::allocator<std::pair<Q::R const, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> > > > >*, E::F::K const&)::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >) const::{lambda(E::F::I<U::d>)#2}&&> > >)()))((m<E::F::I<E::F::J>&>)())) A::B::C::D<E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>(std::vector<Q::R, std::allocator<Q::R> > const&, gtl::S::T<U::V> const&, E::F::W::X const&, E::F::Y const&, gtl::Z<E::F::a> const&, std::unordered_map<Q::R, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> >, L::g::h<void>, std::equal_to<Q::R>, std::allocator<std::pair<Q::R const, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> > > > >*, E::F::K const&)::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >) const::{lambda(E::F::I<E::F::J>)#1}&&, E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>(std::vector<Q::R, std::allocator<Q::R> > const&, gtl::S::T<U::V> const&, E::F::W::X const&, E::F::Y const&, gtl::Z<E::F::a> const&, std::unordered_map<Q::R, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> >, L::g::h<void>, std::equal_to<Q::R>, std::allocator<std::pair<Q::R const, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> > > > >*, E::F::K const&)::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >) const::{lambda(E::F::I<U::d>)#2}&&>::operator()<E::F::I<E::F::J>&>(E::F::I<E::F::J>&) &&",
			"A::B::C::D<E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>(std::vector<Q::R, std::allocator<Q::R> > const&, gtl::S::T<U::V> const&, E::F::W::X const&, E::F::Y const&, gtl::Z<E::F::a> const&, std::unordered_map<Q::R, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> >, L::g::h<void>, std::equal_to<Q::R>, std::allocator<std::pair<Q::R const, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> > > > >*, E::F::K const&)::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >) const::{lambda(E::F::I<E::F::J>)#1}&&, E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>(std::vector<Q::R, std::allocator<Q::R> > const&, gtl::S::T<U::V> const&, E::F::W::X const&, E::F::Y const&, gtl::Z<E::F::a> const&, std::unordered_map<Q::R, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> >, L::g::h<void>, std::equal_to<Q::R>, std::allocator<std::pair<Q::R const, Q::UniquePtr<E::F::b<A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >, E::F::f> > > > >*, E::F::K const&)::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >) const::{lambda(E::F::I<U::d>)#2}&&>::operator()<E::F::I<E::F::J>&>",
			"decltype (((i)((m)()))((m)())) A::B::C::D::operator()(E::F::I&) &&",
			"decltype (((i<A::B::C::j<std::tuple<A::B::C::k<E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>()::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()() const::{lambda(E::F::I<E::F::J>)#1}&&>, A::B::C::k<E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>()::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()() const::{lambda(E::F::I<U::d>)#2}&&> >, E::F::I<E::F::J>&>::l>)((m<std::tuple<A::B::C::k<E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>()::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()() const::{lambda(E::F::I<E::F::J>)#1}&&>, A::B::C::k<E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>()::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()() const::{lambda(E::F::I<U::d>)#2}&&> > >)()))((m<E::F::I<E::F::J>&>)())) A::B::C::D<E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>()::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()() const::{lambda(E::F::I<E::F::J>)#1}&&, E::F::G<E::F::H<E::F::I<E::F::J> >, E::F::K>()::{lambda(Q::R, unsigned long, unsigned long, A::B::c<E::F::I<E::F::J>, E::F::I<U::d> >)#1}::operator()() const::{lambda(E::F::I<U::d>)#2}&&>::operator()<E::F::I<E::F::J>&>(E::F::I<E::F::J>&) &&",
			"A::B::C::D::operator()",
		},
		{
			"_ZcvAna_eE_e",
			"operator long double [new long double]",
			"operator long double [new long double]",
			"operator long double [new long double]",
			"operator long double [new long double]",
			"operator long double [new long double]",
		},
		{
			"_ZZ1irFeeEES_S_",
			"i(() restrict)::long double (long double)(() restrict) restrict",
			"i(long double (long double) restrict)::long double (long double)",
			"i(() restrict)::long double (long double)(() restrict) restrict",
			"i()::long double (long double)(() restrict) restrict",
			"i()::long double (long double)",
		},
		{
			"_Z1_VFaeEZS_S_ES_",
			"_((() volatile) volatile, signed char (long double)(() volatile) volatile::(() volatile) volatile)",
			"_",
			"_((() volatile) volatile, signed char (long double)(() volatile) volatile::(() volatile) volatile)",
			"_(() volatile, signed char (long double)() volatile::() volatile)",
			"_",
		},
		{
			"_ZdsrFliEZS_GS_EcvS_",
			"operator.*(( ( _Imaginary)( _Imaginary) restrict) restrict, long (int)( ( _Imaginary)( _Imaginary) restrict) restrict::operator ( ( _Imaginary)( _Imaginary) restrict) restrict)",
			"operator.*",
			"operator.*(( ( _Imaginary)( _Imaginary) restrict) restrict, long (int)( ( _Imaginary)( _Imaginary) restrict) restrict::operator ( ( _Imaginary)( _Imaginary) restrict) restrict)",
			"operator.*(() restrict, long (int)() restrict::operator () restrict)",
			"operator.*",
		},
		{
			"_ZZN1A1B1CIfEEvPNS_1DERKNS_1EEiS6_T_S7_S7_RKNSt3__u1FIFS7_iiEEEbbPiENKUlZNS1_IfEEvS3_S6_iS6_S7_S7_S7_SD_bbSE_E1GSF_E_clESF_SF_",
			"A::B::C<float>(A::D*, A::E const&, int, A::E const&, float, float, float, std::__u::F<float (int, int)> const&, bool, bool, int*)::{lambda(A::B::C<float>(A::D*, A::E const&, int, A::E const&, float, float, float, std::__u::F<float (int, int)> const&, bool, bool, int*)::G, A::B::C<float>(A::D*, A::E const&, int, A::E const&, float, float, float, std::__u::F<float (int, int)> const&, bool, bool, int*)::G)#1}::operator()(A::B::C<float>(A::D*, A::E const&, int, A::E const&, float, float, float, std::__u::F<float (int, int)> const&, bool, bool, int*)::G, A::B::C<float>(A::D*, A::E const&, int, A::E const&, float, float, float, std::__u::F<float (int, int)> const&, bool, bool, int*)::G) const",
			"A::B::C<float>(A::D*, A::E const&, int, A::E const&, float, float, float, std::__u::F<float (int, int)> const&, bool, bool, int*)::{lambda(A::B::C<float>(A::D*, A::E const&, int, A::E const&, float, float, float, std::__u::F<float (int, int)> const&, bool, bool, int*)::G, A::B::C<float>(A::D*, A::E const&, int, A::E const&, float, float, float, std::__u::F<float (int, int)> const&, bool, bool, int*)::G)#1}::operator()",
			"A::B::C(A::D*, A::E const&, int, A::E const&, float, float, float, std::__u::F const&, bool, bool, int*)::{lambda(A::B::C(A::D*, A::E const&, int, A::E const&, float, float, float, std::__u::F const&, bool, bool, int*)::G, A::B::C(A::D*, A::E const&, int, A::E const&, float, float, float, std::__u::F const&, bool, bool, int*)::G)#1}::operator()(A::B::C(A::D*, A::E const&, int, A::E const&, float, float, float, std::__u::F const&, bool, bool, int*)::G, A::B::C(A::D*, A::E const&, int, A::E const&, float, float, float, std::__u::F const&, bool, bool, int*)::G) const",
			"A::B::C<float>()::{lambda(A::B::C<float>()::G, A::B::C<float>()::G)#1}::operator()(A::B::C<float>()::G, A::B::C<float>()::G) const",
			"A::B::C()::{lambda(A::B::C()::G, A::B::C()::G)#1}::operator()",
		},
		{
			"_ZN1A1B1CIJLNS_1DE131067ELS2_4EEEC2EUa9enable_ifIXclL_ZNS0_1EIJLS2_131067ELS2_4EEEEbNSt1F1GIcNS5_1HIcEEEEEfL0p_EEEPKc",
			"A::B::C<(A::D)131067, (A::D)4>::C(char const*) [enable_if:bool A::B::E<(A::D)131067, (A::D)4>(std::F::G<char, std::F::H<char> >)({parm#1})]",
			"A::B::C<(A::D)131067, (A::D)4>::C",
			"A::B::C::C(char const*) [enable_if:bool A::B::E(std::F::G)({parm#1})]",
			"A::B::C<(A::D)131067, (A::D)4>::C(char const*) [enable_if:bool A::B::E<(A::D)131067, (A::D)4>(std::F::G<char, std::F::H<char> >)({parm#1})]",
			"A::B::C::C",
		},
		{
			"_ZNK1A1B1C1DINS0_1EIKZNK1F1G1HIJNS4_1IINSt1J1KIcNS8_1LIcEENS8_1MIcEEEEEENS4_1NIixEEEE1OEvEUlRT_E_EERKNS8_1PIJNS4_1QISF_EENSP_ISH_EEEEEEclILm0EEEDTcldtclL_ZNS8_1RIRKSN_EEDTclsr3std1SE1TISJ_ELi0EEEvEEonclIXT_EEcl1UIXT_EEclL_ZNSX_ISU_EES10_vEEEEEv",
			"decltype (((decltype (std::S::T<A::B::E<F::G::H<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > >, F::N<int, long long> >::O() const::{lambda(auto:1&)#1} const> const&>(0)) std::J::R<A::B::E<F::G::H<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > >, F::N<int, long long> >::O() const::{lambda(auto:1&)#1} const> const&>()()).(operator()<0ul>))((U<0ul>)(decltype (std::S::T<std::J::P<F::Q<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > > >, F::Q<F::N<int, long long> > > const&>(0)) std::J::R<std::J::P<F::Q<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > > >, F::Q<F::N<int, long long> > > const&>()()))) A::B::C::D<A::B::E<F::G::H<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > >, F::N<int, long long> >::O() const::{lambda(auto:1&)#1} const>, std::J::P<F::Q<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > > >, F::Q<F::N<int, long long> > > const&>::operator()<0ul>() const",
			"A::B::C::D<A::B::E<F::G::H<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > >, F::N<int, long long> >::O() const::{lambda(auto:1&)#1} const>, std::J::P<F::Q<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > > >, F::Q<F::N<int, long long> > > const&>::operator()<0ul>",
			"decltype (((decltype (std::S::T(0)) std::J::R()()).(operator()))((U)(decltype (std::S::T(0)) std::J::R()()))) A::B::C::D::operator()() const",
			"decltype (((decltype (std::S::T<A::B::E<F::G::H<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > >, F::N<int, long long> >::O() const::{lambda(auto:1&)#1} const> const&>(0)) std::J::R<A::B::E<F::G::H<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > >, F::N<int, long long> >::O() const::{lambda(auto:1&)#1} const> const&>()()).(operator()<0ul>))((U<0ul>)(decltype (std::S::T<std::J::P<F::Q<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > > >, F::Q<F::N<int, long long> > > const&>(0)) std::J::R<std::J::P<F::Q<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > > >, F::Q<F::N<int, long long> > > const&>()()))) A::B::C::D<A::B::E<F::G::H<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > >, F::N<int, long long> >::O() const::{lambda(auto:1&)#1} const>, std::J::P<F::Q<F::I<std::J::K<char, std::J::L<char>, std::J::M<char> > > >, F::Q<F::N<int, long long> > > const&>::operator()<0ul>() const",
			"A::B::C::D::operator()",
		},
		{
			"_ZN1A1B1CIKZN1D1E1FIJNS2_1GINSt1H1IIcNS6_1JIcEENS6_1KIcEEEEiEENS5_IixEEEE1LEvEUlRT_E_EC2EPSJ_",
			"A::B::C<D::E::F<D::G<std::H::I<char, std::H::J<char>, std::H::K<char> >, int>, D::G<int, long long> >::L()::{lambda(auto:1&)#1} const>::C(D::E::F<D::G<std::H::I<char, std::H::J<char>, std::H::K<char> >, int>, D::G<int, long long> >::L()::{lambda(auto:1&)#1} const*)",
			"A::B::C<D::E::F<D::G<std::H::I<char, std::H::J<char>, std::H::K<char> >, int>, D::G<int, long long> >::L()::{lambda(auto:1&)#1} const>::C",
			"A::B::C::C(D::E::F::L()::{lambda(auto:1&)#1} const*)",
			"A::B::C<D::E::F<D::G<std::H::I<char, std::H::J<char>, std::H::K<char> >, int>, D::G<int, long long> >::L()::{lambda(auto:1&)#1} const>::C(D::E::F<D::G<std::H::I<char, std::H::J<char>, std::H::K<char> >, int>, D::G<int, long long> >::L()::{lambda(auto:1&)#1} const*)",
			"A::B::C::C",
		},
		{
			"_ZN1A1B1CILb0EvOZN1D1E1FINS3_1GENS3_1HENS3_1IEE1JEvEUlvE0_JEEET0_PNS0_1KEDpDTcp1KIT2_EcvNSt1L1MIbXsr1NE1OISE_EEEE_EEE",
			"void A::B::C<false, void, D::E::F<D::E::G, D::E::H, D::E::I>::J()::{lambda()#2}&&>(A::B::K*)",
			"A::B::C<false, void, D::E::F<D::E::G, D::E::H, D::E::I>::J()::{lambda()#2}&&>",
			"void A::B::C(A::B::K*)",
			"void A::B::C<false, void, D::E::F<D::E::G, D::E::H, D::E::I>::J()::{lambda()#2}&&>(A::B::K*)",
			"A::B::C",
		},
		{
			"_ZN1A1BIJNS_1C1DIJNS_1EENS_1FEEEEENSt1G1HIcNS6_1IIcEENS6_1JIcEEEEEEDTcl1KINS1_1KIsr1LIT0_EE1NJNS1_1MIJDpT_EE1NEEE1N1NEEcldtfp_1OEfp0_EERKNS_1PISE_EEPKc",
			"decltype ((K<A::C::K<short, L<std::G::H<char, std::G::I<char>, std::G::J<char> > > restrict>::N<A::C::M<A::C::D<A::E, A::F> >::N>::N::N>)(({parm#1}.O)(), {parm#2})) A::B<A::C::D<A::E, A::F>, std::G::H<char, std::G::I<char>, std::G::J<char> > >(A::P<L> const&, char const*)",
			"A::B<A::C::D<A::E, A::F>, std::G::H<char, std::G::I<char>, std::G::J<char> > >",
			"decltype ((K)(({parm#1}.O)(), {parm#2})) A::B(A::P const&, char const*)",
			"decltype ((K<A::C::K<short, L<std::G::H<char, std::G::I<char>, std::G::J<char> > > restrict>::N<A::C::M<A::C::D<A::E, A::F> >::N>::N::N>)(({parm#1}.O)(), {parm#2})) A::B<A::C::D<A::E, A::F>, std::G::H<char, std::G::I<char>, std::G::J<char> > >(A::P<L> const&, char const*)",
			"A::B",
		},
		{
			"_ZNSt1A1B1CIZN1D1E1F1GIZNS3_1H1IEixxE1JEENS_1KIFDTclclsr3stdE7declvalIT_EEEEvEEEPN1L1MENS_1OIcNS_1PIcEEEES9_EUlvE_FN1Q1RINS_1SIJNSD_1T1U1VENS6_1WEEEEEEvEEclEv",
			"std::A::B::C<D::E::F::G<D::E::H::I(int, long long, long long)::J>(L::M*, std::A::O<char, std::A::P<char> >, D::E::H::I(int, long long, long long)::J)::{lambda()#1}, Q::R<std::A::S<L::T::U::V, D::E::H::W> > ()>::operator()()",
			"std::A::B::C<D::E::F::G<D::E::H::I(int, long long, long long)::J>(L::M*, std::A::O<char, std::A::P<char> >, D::E::H::I(int, long long, long long)::J)::{lambda()#1}, Q::R<std::A::S<L::T::U::V, D::E::H::W> > ()>::operator()",
			"std::A::B::C::operator()()",
			"std::A::B::C<D::E::F::G<D::E::H::I()::J>()::{lambda()#1}, Q::R<std::A::S<L::T::U::V, D::E::H::W> > ()>::operator()()",
			"std::A::B::C::operator()",
		},
		{
			"_ZNSt1A1B1CIFN1D1EIN1F1G1HEEERKNS5_1IEEE1JINS0_1KIZN1L1MINS2_1NIS8_S6_EENS5_12_GLOBAL__N_11OES8_S6_EEN1P1QINS_1RINSH_IT1_T2_EENS_1SISQ_EEEEEENS_1TIKT_EENSF_1UET0_EUlSA_E_SB_EEEES7_PKNS0_1VESA_",
			"D::E<F::G::H> std::A::B::C<D::E<F::G::H> (F::G::I const&)>::J<std::A::B::K<L::M<D::N<F::G::I, F::G::H>, F::G::(anonymous namespace)::O, F::G::I, F::G::H>(std::A::T<D::N<F::G::I, F::G::H> const>, L::U, F::G::(anonymous namespace)::O)::{lambda(F::G::I const&)#1}, D::E<F::G::H> (F::G::I const&)> >(std::A::B::V const*, F::G::I const&)",
			"std::A::B::C<D::E<F::G::H> (F::G::I const&)>::J<std::A::B::K<L::M<D::N<F::G::I, F::G::H>, F::G::(anonymous namespace)::O, F::G::I, F::G::H>(std::A::T<D::N<F::G::I, F::G::H> const>, L::U, F::G::(anonymous namespace)::O)::{lambda(F::G::I const&)#1}, D::E<F::G::H> (F::G::I const&)> >",
			"D::E std::A::B::C::J(std::A::B::V const*, F::G::I const&)",
			"D::E<F::G::H> std::A::B::C<D::E<F::G::H> (F::G::I const&)>::J<std::A::B::K<L::M<D::N<F::G::I, F::G::H>, F::G::(anonymous namespace)::O, F::G::I, F::G::H>()::{lambda(F::G::I const&)#1}, D::E<F::G::H> (F::G::I const&)> >(std::A::B::V const*, F::G::I const&)",
			"std::A::B::C::J",
		},
		{
			"_ZGVZNK1A1B1CIZNKS0_1DIXadL_ZNS_1EIvE1FEvEES4_EcvNS_1GIFT_DpT0_EEEIvJEvEEvEUlPS4_E_S4_EcvSB_IvJEEEvE1H",
			"guard variable for A::B::C<A::B::D<&A::E<void>::F, A::E<void> >::operator A::G<void (()...)><void, void>() const::{lambda(A::E<void>*)#1}, A::E<void> >::operator A::G<void (()...)><void>() const::H",
			"guard variable for A::B::C<A::B::D<&A::E<void>::F, A::E<void> >::operator A::G<void (()...)><void, void>() const::{lambda(A::E<void>*)#1}, A::E<void> >::operator A::G<void (()...)><void>() const::H",
			"guard variable for A::B::C::operator A::G() const::H",
			"guard variable for A::B::C<A::B::D<&A::E<void>::F, A::E<void> >::operator A::G<void (()...)><void, void>() const::{lambda(A::E<void>*)#1}, A::E<void> >::operator A::G<void (()...)><void>() const::H",
			"guard variable for A::B::C::operator A::G() const::H",
		},
		{
			"_ZNKSt2Cr1AIN1B1CENS_1DIS2_EEEcvbB7v170000Ev",
			"std::Cr::A<B::C, std::Cr::D<B::C> >::operator bool[abi:v170000]() const",
			"std::Cr::A<B::C, std::Cr::D<B::C> >::operator bool[abi:v170000]",
			"std::Cr::A::operator bool[abi:v170000]() const",
			"std::Cr::A<B::C, std::Cr::D<B::C> >::operator bool[abi:v170000]() const",
			"std::Cr::A::operator bool[abi:v170000]",
		},
		{
			"_ZN1A1B1CIZN1D1E1FINS2_2KVINSt3__u1GIcNS6_1HIcEENS6_1IIcEEEESC_EEEENS_1JERKN1K1L1M1NENS_1OIFvvEEENSL_IFvRKT_EEEPNS6_1PIXsr1Q1R1SISO_EE1TENSH_1TISO_EENSH_1UISO_EEE1VEEUlRKNSH_1WES13_E_vJS13_S13_EEET0_NS0_1XEDpNS0_1YIT1_E1ZE",
			"void A::B::C<D::E::F<D::KV<std::__u::G<char, std::__u::H<char>, std::__u::I<char> >, std::__u::G<char, std::__u::H<char>, std::__u::I<char> > > >(K::L::M::N const&, A::O<void ()>, A::O<void (D::KV<std::__u::G<char, std::__u::H<char>, std::__u::I<char> >, std::__u::G<char, std::__u::H<char>, std::__u::I<char> > > const&)>, std::__u::P<Q::R::S<D::KV<std::__u::G<char, std::__u::H<char>, std::__u::I<char> >, std::__u::G<char, std::__u::H<char>, std::__u::I<char> > > >::T, K::L::M::T<D::KV<std::__u::G<char, std::__u::H<char>, std::__u::I<char> >, std::__u::G<char, std::__u::H<char>, std::__u::I<char> > > >, K::L::M::U<D::KV<std::__u::G<char, std::__u::H<char>, std::__u::I<char> >, std::__u::G<char, std::__u::H<char>, std::__u::I<char> > > > >::V*)::{lambda(K::L::M::W const&, K::L::M::W const)#1}, void, K::L::M::W const, K::L::M::W const>(A::B::X, A::B::Y<K::L::M::W const>::Z, A::B::Y<K::L::M::W const>::Z)",
			"A::B::C<D::E::F<D::KV<std::__u::G<char, std::__u::H<char>, std::__u::I<char> >, std::__u::G<char, std::__u::H<char>, std::__u::I<char> > > >(K::L::M::N const&, A::O<void ()>, A::O<void (D::KV<std::__u::G<char, std::__u::H<char>, std::__u::I<char> >, std::__u::G<char, std::__u::H<char>, std::__u::I<char> > > const&)>, std::__u::P<Q::R::S<D::KV<std::__u::G<char, std::__u::H<char>, std::__u::I<char> >, std::__u::G<char, std::__u::H<char>, std::__u::I<char> > > >::T, K::L::M::T<D::KV<std::__u::G<char, std::__u::H<char>, std::__u::I<char> >, std::__u::G<char, std::__u::H<char>, std::__u::I<char> > > >, K::L::M::U<D::KV<std::__u::G<char, std::__u::H<char>, std::__u::I<char> >, std::__u::G<char, std::__u::H<char>, std::__u::I<char> > > > >::V*)::{lambda(K::L::M::W const&, K::L::M::W const)#1}, void, K::L::M::W const, K::L::M::W const>",
			"void A::B::C(A::B::X, A::B::Y::Z, A::B::Y::Z)",
			"void A::B::C<D::E::F<D::KV<std::__u::G<char, std::__u::H<char>, std::__u::I<char> >, std::__u::G<char, std::__u::H<char>, std::__u::I<char> > > >()::{lambda(K::L::M::W const&, K::L::M::W const)#1}, void, K::L::M::W const, K::L::M::W const>(A::B::X, A::B::Y<K::L::M::W const>::Z, A::B::Y<K::L::M::W const>::Z)",
			"A::B::C",
		},
		{
			"_ZZNK1P1A1B1CIZNS0_1DIJETpTnRiJENS0_1E1F1GEvEENS_1HIFvDpT_EEEN1J1KINS0_1LIT1_EEEEEUlPNSF_IS7_EEE_SJ_EcvNS8_IFT_DpT0_EEEIvJEQsr1ME1NITL0__SN_PT0_DpTL0_0_EEEvENKUllE_clEl",
			"P::A::B::C<P::A::D<, P::A::E::F::G, void>(J::K<P::A::L<P::A::E::F::G> >)::{lambda(P::A::L<P::A::E::F::G>*)#1}, P::A::L<P::A::E::F::G> >::operator P::H<void (()...)><void>() const::{lambda(long)#1}::operator()(long) const",
			"P::A::B::C<P::A::D<, P::A::E::F::G, void>(J::K<P::A::L<P::A::E::F::G> >)::{lambda(P::A::L<P::A::E::F::G>*)#1}, P::A::L<P::A::E::F::G> >::operator P::H<void (()...)><void>() const::{lambda(long)#1}::operator()",
			"P::A::B::C::operator P::H() const::{lambda(long)#1}::operator()(long) const",
			"P::A::B::C<P::A::D<, P::A::E::F::G, void>()::{lambda(P::A::L<P::A::E::F::G>*)#1}, P::A::L<P::A::E::F::G> >::operator P::H<void (()...)><void>() const::{lambda(long)#1}::operator()(long) const",
			"P::A::B::C::operator P::H() const::{lambda(long)#1}::operator()",
		},
	}

	for _, test := range tests {
		if got, err := ToString(test.input); err != nil {
			t.Errorf("demangling %s: unexpected error %v", test.input, err)
		} else if got != test.want {
			t.Errorf("demangling %s: got %s, want %s", test.input, got, test.want)
		}

		if got, err := ToString(test.input, NoParams); err != nil {
			t.Errorf("demangling NoParams  %s: unexpected error %v", test.input, err)
		} else if got != test.wantNoParams {
			t.Errorf("demangling NoParams %s: got %s, want %s", test.input, got, test.wantNoParams)
		}

		if got, err := ToString(test.input, NoTemplateParams); err != nil {
			t.Errorf("demangling NoTemplateParams %s: unexpected error %v", test.input, err)
		} else if got != test.wantNoTemplateParams {
			t.Errorf("demangling NoTemplateParams %s: got %s, want %s", test.input, got, test.wantNoTemplateParams)
		}

		if got, err := ToString(test.input, NoEnclosingParams); err != nil {
			t.Errorf("demangling NoEnclosingParams %s: unexpected error %v", test.input, err)
		} else if got != test.wantNoEnclosingParams {
			t.Errorf("demangling NoEnclosingParams %s: got %s, want %s", test.input, got, test.wantNoEnclosingParams)
		}

		if got, err := ToString(test.input, NoParams, NoTemplateParams, NoEnclosingParams); err != nil {
			t.Errorf("demangling NoTemplateParams %s: unexpected error %v", test.input, err)
		} else if got != test.wantMinimal {
			t.Errorf("demangling Minimal %s: got %s, want %s", test.input, got, test.wantMinimal)
		}

		// Test Filter also.
		if got := Filter(test.input); got != test.want {
			t.Errorf("Filter(%s) == %s, want %s", test.input, got, test.want)
		}
	}
}

// Test for some failure cases.
func TestFailure(t *testing.T) {
	var tests = []struct {
		input string
		error string
		off   int
	}{
		{
			"_Z1FE",
			"unparsed characters at end of mangled name",
			4,
		},
		{
			"_Z1FQ",
			"expected at least one type",
			4,
		},
		{
			"_ZZSaIL0D",
			"expected positive number",
			8,
		},
		{
			"_ZNKE",
			"expected prefix",
			4,
		},
		{
			"_ZcvT_",
			"not in scope of template",
			6,
		},
		{
			"_Z1AIXsZ1_EE",
			"missing argument pack",
			8,
		},
		{
			"_Z1gIEDTclspilE",
			"expected expression",
			15,
		},
		{
			"_ZNcvZN1ET_IEE",
			"after local name",
			14,
		},
		{
			"_Zv00",
			"expected positive number",
			5,
		},
		{
			"_ZcvT_B2T0",
			"template parameter not in scope",
			10,
		},
		{
			"_ZStcvT_",
			"template parameter not in scope",
			8,
		},
		{
			"_Z1aIeEU1RT_ZcvS1_",
			"expected E after local name",
			18,
		},
		{
			"_ZNcvT_oRIEE",
			"template index out of range",
			11,
		},
		{
			"_ZNcvT_D0IIEE",
			"expected prefix",
			13,
		},
		{
			"_ZcvT_IAoncvT__eE",
			"template parameter not in scope",
			17,
		},
	}

	for _, test := range tests {
		got, err := ToString(test.input)
		if err == nil {
			t.Errorf("unexpected success for %s: %s", test.input, got)
		} else if !strings.Contains(err.Error(), test.error) {
			t.Errorf("unexpected error for %s: %v", test.input, err)
		} else {
			s := err.Error()
			i := strings.LastIndex(s, " at ")
			if i < 0 {
				t.Errorf("missing offset in error for %s: %v", test.input, err)
			} else {
				off, oerr := strconv.Atoi(s[i+4:])
				if oerr != nil {
					t.Errorf("can't parse offset (%s) for %s: %v", s[i+4:], test.input, err)
				} else if off != test.off {
					t.Errorf("unexpected offset for %s: got %d, want %d", test.input, off, test.off)
				}
			}
		}

		if got := Filter(test.input); got != test.input {
			t.Errorf("Filter(%s) == %s, want %s", test.input, got, test.input)
		}
	}
}

func TestMaxLength(t *testing.T) {
	if isMaxLength(Option(0)) {
		t.Errorf("isMaxLength(0) returned true")
	}
	for pow := 1; pow <= 30; pow++ {
		opt := MaxLength(pow)
		if !isMaxLength(opt) {
			t.Errorf("isMaxLength(%x) returned false", opt)
		}
		if got := maxLength(opt); got != 1<<pow {
			t.Errorf("maxLength(%x) = %v, want %v", opt, got, 1<<pow)
		}
	}
}
