/*
 *  Copyright (c) 2024 Mikhail Knyazhev <markus621@yandex.ru>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package syncing

import (
	"testing"

	"go.osspkg.com/casecheck"
)

func TestUnit_NewSwitch(t *testing.T) {
	sync := NewSwitch()

	casecheck.False(t, sync.IsOn())
	casecheck.True(t, sync.IsOff())

	casecheck.True(t, sync.On())
	casecheck.False(t, sync.On())

	casecheck.False(t, sync.IsOff())
	casecheck.True(t, sync.IsOn())

}
