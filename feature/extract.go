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
	var ranges []struct {
		start, end int
		name       string
	}

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

		start, end := b.slotID, b.slotID
		if b.isSeq {
			start, end = b.slots.start, b.slots.end
		}
		if b.slotID != 0 && !b.strOut {
			for _, r := range ranges {
				if start <= r.end && r.start <= end {
					return nil, fmt.Errorf("operator: feature conf line %d: output slots of %q overlap %q", lineNo, b.name, r.name)
				}
			}
			ranges = append(ranges, struct {
				start, end int
				name       string
			}{start, end, b.name})
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

// Extract is atomic for the current sample: on failure it preserves only the
// caller's existing output prefix and resets all intermediate sample state.
func (e *Extractor) Extract(in *Input, st *State, out []Fid) ([]Fid, error) {
	if st == nil {
		st = NewState()
		defer st.Reset()
	}
	st.ensure()
	st.err = nil
	start := len(out)
	for i, op := range e.ops {
		err := op.Emit(in, st, &out)
		if err == nil {
			err = st.err
		}
		if err != nil {
			clear(out[start:])
			out = out[:start]
			st.Reset()
			return out, fmt.Errorf("operator: feature %q (%d/%d) failed: %w", op.Conf().name, i+1, len(e.ops), err)
		}
	}
	return out, nil
}

// ValidateInputs checks the ordered graph against raw/global input names. New
// cannot know a caller's schema, so package consumers invoke this explicitly.
func (e *Extractor) ValidateInputs(names []string) error {
	available := make(map[string]bool, len(names)+len(e.ops))
	for _, name := range names {
		available[name] = true
	}
	var failures []string
	for _, op := range e.ops {
		b := op.Conf()
		for _, dep := range b.depends {
			if !available[dep] {
				failures = append(failures, fmt.Sprintf("feature %q: dependency %q is not an input or preceding addcol", b.name, dep))
			}
		}
		if b.addcol {
			available[b.name] = true
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("operator: invalid dependencies:\n%s", strings.Join(failures, "\n"))
	}
	return nil
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
