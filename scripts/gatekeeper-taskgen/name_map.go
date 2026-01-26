package main

import "fmt"

type nameKey struct {
	kind      string
	namespace string
	name      string
}

type nameRegistry struct {
	used map[nameKey]bool
}

func newNameRegistry() *nameRegistry {
	return &nameRegistry{used: map[nameKey]bool{}}
}

func (nr *nameRegistry) allocate(kind, namespace, base string) (string, bool) {
	if base == "" {
		return "", false
	}
	key := nameKey{kind: kind, namespace: namespace, name: base}
	if !nr.used[key] {
		nr.used[key] = true
		return base, false
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		key = nameKey{kind: kind, namespace: namespace, name: candidate}
		if !nr.used[key] {
			nr.used[key] = true
			return candidate, true
		}
	}
}
