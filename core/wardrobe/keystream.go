package wardrobe

import (
	"encoding/binary"
	"errors"
	"strconv"
)

type Keystream struct {
	vals []byte
	mask []byte
}

func ParseKeystream(b []byte) (*Keystream, error) {
	if len(b) < 4 {
		return nil, errors.New("wardrobe: keystream too short")
	}
	n := int(binary.LittleEndian.Uint32(b))
	want := 4 + n + (n+7)/8
	if len(b) != want {
		return nil, errors.New("wardrobe: keystream is " + strconv.Itoa(len(b)) + " bytes, want " + strconv.Itoa(want))
	}
	return &Keystream{vals: b[4 : 4+n], mask: b[4+n:]}, nil
}

func (k *Keystream) Len() int { return len(k.vals) }

func (k *Keystream) At(i int) (byte, bool) {
	if i < 0 || i >= len(k.vals) {
		return 0, false
	}
	return k.vals[i], k.mask[i>>3]&(1<<(i&7)) != 0
}
