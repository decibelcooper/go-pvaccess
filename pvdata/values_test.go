package pvdata

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPVSize(t *testing.T) {
	s := PVSize(0)
	var buf bytes.Buffer
	pvf := PVField(&s)
	if err := pvf.PVEncode(&EncoderState{
		Buf:       &buf,
		ByteOrder: binary.BigEndian,
	}); err != nil {
		t.Error(err)
	}
	bytes := buf.Bytes()
	if len(bytes) != 1 || bytes[0] != 0 {
		t.Errorf("bytes = %v, want [0]", bytes)
	}
}

func TestPVEncode(t *testing.T) {
	tests := []struct {
		in             interface{}
		wantBE, wantLE []byte
	}{
		{PVSize(-1), []byte{255}, nil},
		{PVSize(0), []byte{0}, nil},
		{PVSize(254), []byte{254, 0, 0, 0, 254}, []byte{254, 254, 0, 0, 0}},
		{PVSize(0x7fffffff), []byte{254, 0x7f, 0xff, 0xff, 0xff, 0, 0, 0, 0, 0x7f, 0xff, 0xff, 0xff}, []byte{254, 0xff, 0xff, 0xff, 0x7f, 0xff, 0xff, 0xff, 0x7f, 0, 0, 0, 0}},
		{PVBoolean(true), []byte{1}, nil},
		{PVBoolean(false), []byte{0}, nil},
		{true, []byte{1}, nil},
		{false, []byte{0}, nil},
		{PVByte(0), []byte{0}, nil},
		{PVByte(-1), []byte{0xFF}, nil},
		{int8(127), []byte{0x7F}, nil},
		{PVUByte(0), []byte{0}, nil},
		{PVUByte(129), []byte{129}, nil},
		{byte(13), []byte{13}, nil},
		{PVShort(256), []byte{1, 0}, []byte{}},
		{PVShort(-1), []byte{0xff, 0xff}, []byte{}},
		{int16(32), []byte{0, 32}, []byte{}},
		{PVUShort(32768), []byte{0x80, 0x00}, []byte{}},
		{uint16(32768), []byte{0x80, 0x00}, []byte{}},
		{PVInt(65536), []byte{0, 1, 0, 0}, []byte{}},
		{PVInt(-1), []byte{0xff, 0xff, 0xff, 0xff}, []byte{}},
		{int32(32), []byte{0, 0, 0, 32}, []byte{}},
		{PVUInt(0x80000000), []byte{0x80, 0, 0, 0}, []byte{}},
		{uint32(1), []byte{0, 0, 0, 1}, []byte{}},
		{PVLong(-1), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, nil},
		{int64(1), []byte{0, 0, 0, 0, 0, 0, 0, 1}, []byte{}},
		{PVULong(0x8000000000000000), []byte{0x80, 0, 0, 0, 0, 0, 0, 0}, []byte{}},
		{uint64(13), []byte{0, 0, 0, 0, 0, 0, 0, 13}, []byte{}},
		{float32(85.125), []byte{0x42, 0xAA, 0x40, 0x00}, []byte{}},
		{float64(85.125), []byte{0x40, 0x55, 0x48, 0, 0, 0, 0, 0}, []byte{}},
		{[]PVBoolean{true, false, false}, []byte{3, 1, 0, 0}, []byte{3, 1, 0, 0}},
	}
	for _, test := range tests {
		name := fmt.Sprintf("%T: %#v", test.in, test.in)
		t.Run(name, func(t *testing.T) {
			// Make a copy on the stack
			in := reflect.New(reflect.TypeOf(test.in))
			in.Elem().Set(reflect.ValueOf(test.in))
			pvf := valueToPVField(in)
			if pvf == nil && test.wantBE != nil {
				t.Fatal("failed to convert to PVField; expected PVField implementation")
			}
			if pvf != nil && test.wantBE == nil {
				t.Fatalf("got instance of %T, expected conversion failure", pvf)
			}
			if test.wantLE == nil {
				test.wantLE = test.wantBE
			} else if len(test.wantLE) == 0 {
				test.wantLE = make([]byte, len(test.wantBE))
				for i := 0; i < len(test.wantBE); i++ {
					test.wantLE[i] = test.wantBE[len(test.wantBE)-1-i]
				}
			}
			for _, byteOrder := range []struct {
				byteOrder binary.ByteOrder
				want      []byte
			}{
				{binary.BigEndian, test.wantBE},
				{binary.LittleEndian, test.wantLE},
			} {
				t.Run(byteOrder.byteOrder.String(), func(t *testing.T) {
					var buf bytes.Buffer
					s := &EncoderState{
						Buf:       &buf,
						ByteOrder: byteOrder.byteOrder,
					}
					if err := pvf.PVEncode(s); err != nil {
						t.Errorf("unexpected encode error: %v", err)
					}
					if diff := cmp.Diff(buf.Bytes(), byteOrder.want); diff != "" {
						t.Errorf("got(-)/want(+)\n%s", diff)
					}
				})
			}
		})
	}
}
