package feature

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Extractor struct {
	ops []Operator
}

func NewFromText(text string) (*Extractor, error) {
	return New(strings.NewReader(text))
}

func New(r io.Reader) (*Extractor, error) {
	e := &Extractor{ops: make([]Operator, 0, 64)}

	seenName := make(map[string]bool)
	seenSlot := make(map[int]string)

	s := bufio.NewScanner(r)
	for lineNo := 1; s.Scan(); lineNo++ {
		conf, ok, err := parseConfLine(s.Text())
		if err != nil {
			return nil, fmt.Errorf("operator: feature conf line %d: %v", lineNo, err)
		}
		if !ok {
			continue
		}

		op, err := create(conf)
		if err != nil {
			return nil, fmt.Errorf("operator: feature conf line %d: %v", lineNo, err)
		}
		b := op.Conf()

		if seenName[b.name] {
			return nil, fmt.Errorf("operator: feature conf line %d: duplicate feature name %q",
				lineNo, b.name)
		}
		seenName[b.name] = true

		if b.slotID != 0 {
			if owner, dup := seenSlot[b.slotID]; dup {
				return nil, fmt.Errorf("operator: feature conf line %d: slot_id %d already used by %q",
					lineNo, b.slotID, owner)
			}
			seenSlot[b.slotID] = b.name
		}

		e.ops = append(e.ops, op)
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("operator: read feature conf: %v", err)
	}
	if len(e.ops) == 0 {
		return nil, fmt.Errorf("operator: feature conf has no valid feature")
	}
	return e, nil
}

func (e *Extractor) Operators() []Operator { return e.ops }

func (e *Extractor) Extract(in *Input, st *State, out []Fid) ([]Fid, error) {
	if st == nil {
		st = NewState()
	}
	st.ensure()

	var firstErr error
	var failed int

	for _, op := range e.ops {
		if err := op.Emit(in, st, &out); err != nil {
			failed++
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	if firstErr != nil {
		return out, fmt.Errorf("operator: %d/%d features failed, first: %v",
			failed, len(e.ops), firstErr)
	}
	return out, nil
}

func parseConfLine(raw string) (map[string]string, bool, error) {
	line := raw
	if i := strings.IndexByte(line, '#'); i >= 0 {
		line = line[:i]
	}
	line = strings.Trim(line, " \t\r;")
	if line == "" {
		return nil, false, nil
	}

	conf := make(map[string]string, 8)
	for _, field := range strings.Split(line, ";") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		kv := strings.SplitN(field, "=", 2)
		if len(kv) != 2 {
			return nil, false, fmt.Errorf("malformed field %q (want key=value)", field)
		}
		key := strings.TrimSpace(kv[0])
		if key == "" {
			return nil, false, fmt.Errorf("empty key in field %q", field)
		}
		if _, dup := conf[key]; dup {
			return nil, false, fmt.Errorf("duplicate key %q", key)
		}
		conf[key] = strings.TrimSpace(kv[1])
	}
	if len(conf) == 0 {
		return nil, false, nil
	}
	return conf, true, nil
}
