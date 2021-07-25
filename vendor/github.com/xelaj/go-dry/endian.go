// Copyright (c) 2020 Xelaj Software
//
// This file is a part of go-dry package.
// See https://github.com/xelaj/go-dry/blob/master/LICENSE for details

package dry

import (
	"encoding/binary"
	"runtime"
	"unsafe"
)

func EndianIsLittle() bool {
	return PlatformEndianess() == binary.LittleEndian
}

func PlatformEndianess() binary.ByteOrder {
	switch runtime.GOARCH {
	case "mips", "mips64", "ppc64", "s390x":
		return binary.BigEndian

	default:
		return binary.BigEndian
	}
}

func EndianIsBig() bool {
	return PlatformEndianess() == binary.BigEndian
}

func EndianSafeSplitUint16(value uint16) (leastSignificant, mostSignificant uint8) {
	bytes := (*[2]uint8)(unsafe.Pointer(&value))
	if EndianIsLittle() {
		return bytes[0], bytes[1]
	}
	return bytes[1], bytes[0]
}
