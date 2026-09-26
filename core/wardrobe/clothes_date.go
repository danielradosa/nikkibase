package wardrobe

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
)

const (
	tsLen      = 10
	recPadding = 14
	lookahead  = 4
)

func Decode(src []byte, ks *Keystream) (*Wardrobe, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(src)))
	if err != nil {
		return nil, &base64Error{err}
	}
	if len(raw) < 2 {
		return nil, errors.New("wardrobe: file is " + strconv.Itoa(len(raw)) + " bytes, too short to hold a table")
	}
	p := plaintext{raw, ks}
	if p.contradicts(0, '{') {
		return nil, errors.New("wardrobe: does not start with a Lua table")
	}

	w := &Wardrobe{}
	for pos := 1; pos < len(raw)-1; {
		idLen, err := p.idLenAt(pos)
		if err != nil {
			return nil, err
		}
		if id, ok := p.readInt(pos+1, idLen); ok {
			w.Items = append(w.Items, id)
		} else {
			w.Unresolved++
		}
		pos += idLen + recPadding
	}
	return w, nil
}

type base64Error struct{ err error }

func (e *base64Error) Error() string { return "wardrobe: not valid base64: " + e.err.Error() }
func (e *base64Error) Unwrap() error { return e.err }

func (p plaintext) idLenAt(pos int) (int, error) {
	five, six := p.chainFits(pos, 5, lookahead), p.chainFits(pos, 6, lookahead)
	switch {
	case five && !six:
		return 5, nil
	case six && !five:
		return 6, nil
	case !five && !six:
		return 0, errors.New("wardrobe: no valid record at byte " + strconv.Itoa(pos))
	default:
		return 0, errors.New("wardrobe: record at byte " + strconv.Itoa(pos) + " is ambiguous; keystream gap too wide")
	}
}

type plaintext struct {
	cipher []byte
	ks     *Keystream
}

func (p plaintext) at(i int) (byte, bool) {
	if i < 0 || i >= len(p.cipher) {
		return 0, false
	}
	k, ok := p.ks.At(i)
	if !ok {
		return 0, false
	}
	return p.cipher[i] ^ k, true
}

func (p plaintext) contradicts(i int, c byte) bool {
	b, ok := p.at(i)
	return ok && b != c
}

func (p plaintext) notDigit(i int) bool {
	b, ok := p.at(i)
	return ok && (b < '0' || b > '9')
}

func (p plaintext) recordFits(pos, idLen int) bool {
	end := pos + idLen + recPadding - 1
	if end >= len(p.cipher) {
		return false
	}
	if p.contradicts(pos, '[') || p.contradicts(pos+idLen+1, ']') || p.contradicts(pos+idLen+2, '=') {
		return false
	}
	for i := pos + 1; i <= pos+idLen; i++ {
		if p.notDigit(i) {
			return false
		}
	}
	for i := pos + idLen + 3; i < pos+idLen+3+tsLen; i++ {
		if p.notDigit(i) {
			return false
		}
	}
	if b, ok := p.at(end); ok && b != ',' && b != '}' {
		return false
	}
	return end != len(p.cipher)-1 || !p.contradicts(end, '}')
}

func (p plaintext) chainFits(pos, idLen, depth int) bool {
	if !p.recordFits(pos, idLen) {
		return false
	}
	next := pos + idLen + recPadding
	if depth == 0 || next >= len(p.cipher)-1 {
		return true
	}
	return p.chainFits(next, 5, depth-1) || p.chainFits(next, 6, depth-1)
}

func (p plaintext) readInt(pos, n int) (int, bool) {
	v := 0
	for i := pos; i < pos+n; i++ {
		b, ok := p.at(i)
		if !ok || b < '0' || b > '9' {
			return 0, false
		}
		v = v*10 + int(b-'0')
	}
	return v, true
}
