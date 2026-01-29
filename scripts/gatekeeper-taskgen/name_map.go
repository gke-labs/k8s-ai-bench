// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import "fmt"

type nameKey struct {
	kind      string
	namespace string
	name      string
}

type nameRegistry struct {
	used map[nameKey]struct{}
}

func newNameRegistry() *nameRegistry {
	return &nameRegistry{used: map[nameKey]struct{}{}}
}

func (nr *nameRegistry) allocate(kind, namespace, base string) (string, bool) {
	if base == "" {
		return "", false
	}

	key := nameKey{kind: kind, namespace: namespace, name: base}
	if _, ok := nr.used[key]; !ok {
		nr.used[key] = struct{}{}
		return base, false
	}

	// try base-2, base-3, ...
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		key = nameKey{kind: kind, namespace: namespace, name: candidate}
		if _, ok := nr.used[key]; !ok {
			nr.used[key] = struct{}{}
			return candidate, true
		}
	}
}
