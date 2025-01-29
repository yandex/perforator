/*
   Copyright 2020 YANDEX LLC

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package hasql

import (
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func Test_uncheckedSQLxNode(t *testing.T) {
	assert.Nil(t, uncheckedSQLxNode(nil))

	expected := NewNode("foo", &sqlx.DB{})
	assert.Equal(t, expected, uncheckedSQLxNode(expected))
}

func Test_checkedSQLxNode(t *testing.T) {
	node, err := checkedSQLxNode(nil, errors.New("err"))
	assert.Error(t, err)
	assert.Nil(t, node)

	node, err = checkedSQLxNode(NewNode("foo", &sqlx.DB{}), errors.New("err"))
	assert.Error(t, err)
	assert.Nil(t, node)

	node, err = checkedSQLxNode(nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, node)

	expected := NewNode("foo", &sqlx.DB{})
	node, err = checkedSQLxNode(expected, nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, node)
}
