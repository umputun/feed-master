// Copyright (c) 2020 KHS Films
//
// This file is a part of mtproto package.
// See https://github.com/xelaj/mtproto/blob/master/LICENSE for details

package mtproto

import (
	"github.com/xelaj/mtproto/internal/encoding/tl"
	"github.com/xelaj/mtproto/internal/mtproto/objects"
)

// это неофициальная информация, но есть подозрение, что список датацентров АБСОЛЮТНО идентичный для всех
// приложений. Несмотря на это, любой клиент ОБЯЗАН явно указывать список датацентров, ради надежности.
// данный список лишь эксперементальный и не является частью протокола.
var defaultDCList = map[int]string{
	1: "149.154.175.58:443",
	2: "149.154.167.50:443",
	3: "149.154.175.100:443",
	4: "149.154.167.91:443",
	5: "91.108.56.151:443",
}

// https://core.telegram.org/mtproto/mtproto-transports
var (
	transportModeAbridged           = [...]byte{0xef}                   // meta:immutable
	transportModeIntermediate       = [...]byte{0xee, 0xee, 0xee, 0xee} // meta:immutable
	transportModePaddedIntermediate = [...]byte{0xdd, 0xdd, 0xdd, 0xdd} // meta:immutable
	transportModeFull               = [...]byte{}                       // meta:immutable
)

func MessageRequireToAck(msg tl.Object) bool {
	switch msg.(type) {
	case /**objects.Ping,*/ *objects.MsgsAck:
		return false
	default:
		return true
	}
}
