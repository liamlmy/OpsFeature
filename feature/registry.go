package feature

import "fmt"

var registry = map[string]func() Operator{}

// Register registers an operator implementation by its configuration class.
// Operator packages call Register from init, keeping the feature package free
// of import cycles while it owns the parser and extraction runtime.
func Register(class string, ctor func() Operator) {
	if class == "" || ctor == nil {
		panic("feature: invalid operator registration")
	}
	if _, exists := registry[class]; exists {
		panic(fmt.Sprintf("feature: operator class %q already registered", class))
	}
	registry[class] = ctor
}

func Classes() []string {
	out := make([]string, 0, len(registry))
	for c := range registry {
		out = append(out, c)
	}
	sortStrings(out)
	return out
}

func create(conf map[string]string) (Operator, error) {
	class := conf["class"]
	if class == "" {
		return nil, fmt.Errorf("operator: feature %q has no class", conf["name"])
	}
	ctor, ok := registry[class]
	if !ok {
		return nil, fmt.Errorf("operator: unknown class %q (feature %q); registered: %v",
			class, conf["name"], Classes())
	}

	op := ctor()
	b := op.Conf()
	if err := b.load(conf); err != nil {
		return nil, err
	}
	if err := op.Init(); err != nil {
		return nil, err
	}
	if err := b.validate(); err != nil {
		return nil, err
	}
	return op, nil
}
