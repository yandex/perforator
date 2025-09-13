// Copyright (c) 2018, Maxime Soulé
// All rights reserved.
//
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree.

package td

import (
	"reflect"
	"strings"

	"github.com/maxatome/go-testdeep/internal/flat"
	"github.com/maxatome/go-testdeep/internal/util"
)

type tdList struct {
	baseOKNil
	items []reflect.Value
}

func newList(items ...any) tdList {
	return tdList{
		baseOKNil: newBaseOKNil(4),
		items:     flat.Values(items),
	}
}

func (l *tdList) String() string {
	var b strings.Builder
	b.WriteString(l.GetLocation().Func)
	return util.SliceToString(&b, l.items).
		String()
}
