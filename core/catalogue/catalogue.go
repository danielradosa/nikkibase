package catalogue

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const magic = "NBI3"

type Item struct {
	ID        int32
	Slot      uint8
	Position  uint16
	Group     uint8
	Attrs     [5]int8
	Stats     [5]int32
	Tags      []int32
	FlatBonus int32
}

func Write(items []Item) []byte {
	var buf bytes.Buffer
	buf.WriteString(magic)
	put(&buf, uint32(len(items)))

	for _, it := range items {
		put(&buf, it.ID)
	}
	for _, it := range items {
		buf.WriteByte(it.Slot)
	}
	for _, it := range items {
		put(&buf, it.Position)
	}
	for _, it := range items {
		buf.WriteByte(it.Group)
	}
	for _, it := range items {
		for _, a := range it.Attrs {
			buf.WriteByte(byte(a))
		}
	}
	for _, it := range items {
		for _, s := range it.Stats {
			put(&buf, s)
		}
	}

	for _, it := range items {
		put(&buf, it.FlatBonus)
	}

	offset := uint32(0)
	for _, it := range items {
		put(&buf, offset)
		offset += uint32(len(it.Tags))
	}
	put(&buf, offset)
	for _, it := range items {
		for _, t := range it.Tags {
			put(&buf, t)
		}
	}
	return buf.Bytes()
}

func put(buf *bytes.Buffer, v any) {
	if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
		panic(err)
	}
}

type Catalogue struct {
	IDs       []int32
	Slots     []uint8
	Positions []uint16
	Groups    []uint8
	Attrs     []int8
	Stats     []int32
	FlatBonus []int32
	TagOffset []uint32
	Tags      []int32
}

func Read(b []byte) (*Catalogue, error) {
	if len(b) < 8 || string(b[:4]) != magic {
		return nil, errors.New("catalogue: not a catalogue bundle")
	}
	n := int(binary.LittleEndian.Uint32(b[4:]))
	r := reader{b: b, at: 8}

	c := &Catalogue{
		IDs:       make([]int32, n),
		Slots:     make([]uint8, n),
		Positions: make([]uint16, n),
		Groups:    make([]uint8, n),
		Attrs:     make([]int8, n*5),
		Stats:     make([]int32, n*5),
		FlatBonus: make([]int32, n),
		TagOffset: make([]uint32, n+1),
	}
	for i := range c.IDs {
		c.IDs[i] = int32(r.uint32())
	}
	for i := range c.Slots {
		c.Slots[i] = r.byte()
	}
	for i := range c.Positions {
		c.Positions[i] = r.uint16()
	}
	for i := range c.Groups {
		c.Groups[i] = r.byte()
	}
	for i := range c.Attrs {
		c.Attrs[i] = int8(r.byte())
	}
	for i := range c.Stats {
		c.Stats[i] = int32(r.uint32())
	}
	for i := range c.FlatBonus {
		c.FlatBonus[i] = int32(r.uint32())
	}
	for i := range c.TagOffset {
		c.TagOffset[i] = r.uint32()
	}
	if r.err == nil {
		c.Tags = make([]int32, c.TagOffset[n])
		for i := range c.Tags {
			c.Tags[i] = int32(r.uint32())
		}
	}
	if r.err != nil {
		return nil, r.err
	}
	return c, nil
}

type reader struct {
	b   []byte
	at  int
	err error
}

func (r *reader) take(n int) []byte {
	if r.err != nil || r.at+n > len(r.b) {
		r.err = errors.New("catalogue: bundle is truncated")
		return make([]byte, n)
	}
	v := r.b[r.at : r.at+n]
	r.at += n
	return v
}

func (r *reader) byte() uint8    { return r.take(1)[0] }
func (r *reader) uint16() uint16 { return binary.LittleEndian.Uint16(r.take(2)) }
func (r *reader) uint32() uint32 { return binary.LittleEndian.Uint32(r.take(4)) }

const WholeGroup = 0x80

func SplitGroup(g uint8) (group uint8, whole bool) {
	return g &^ WholeGroup, g&WholeGroup != 0
}
