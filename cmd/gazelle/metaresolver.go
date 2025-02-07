/* Copyright 2019 The Bazel Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// metaResolver provides a rule.Resolver for any rule.Rule.
type metaResolver struct {
	// builtins provides a map of the language kinds to their resolver.
	builtins map[string]resolve.Resolver

	// mappedKinds provides a list of replacements used by File.Pkg.
	mappedKinds map[string][]config.MappedKind
}

func newMetaResolver() *metaResolver {
	return &metaResolver{
		builtins:    make(map[string]resolve.Resolver),
		mappedKinds: make(map[string][]config.MappedKind),
	}
}

// AddBuiltin registers a builtin kind with its info.
func (mr *metaResolver) AddBuiltin(kindName string, resolver resolve.Resolver) {
	mr.builtins[kindName] = resolver
}

// MappedKind records the fact that the given mapping was applied while
// processing the given package.
func (mr *metaResolver) MappedKind(pkgRel string, kind config.MappedKind) {
	mr.mappedKinds[pkgRel] = append(mr.mappedKinds[pkgRel], kind)
}

// Resolver returns a resolver for the given rule and package, and a bool
// indicating whether one was found. Empty string may be passed for pkgRel,
// which results in consulting the builtin kinds only.
func (mr *metaResolver) Resolver(ruleKind string, pkgRel string) resolve.Resolver {
	for _, mappedKind := range mr.mappedKinds[pkgRel] {
		if mappedKind.KindName == ruleKind {
			fromKindResolver := mr.builtins[mappedKind.FromKind]
			if fromKindResolver == nil {
				return nil
			}
			return inverseMapKindResolver{
				fromKind: mappedKind.FromKind,
				delegate: fromKindResolver,
			}
		}
	}
	return mr.builtins[ruleKind]
}

// inverseMapKindResolver applies an inverse of the map_kind
// operations to provided rules. This enables language
// modules to remain ignorant of mapped kinds.
type inverseMapKindResolver struct {
	fromKind string
	delegate resolve.Resolver
}

var _ resolve.Resolver = (*inverseMapKindResolver)(nil)

func (imkr inverseMapKindResolver) Name() string {
	return imkr.delegate.Name()
}

func (imkr inverseMapKindResolver) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	r = imkr.inverseMapKind(r)
	return imkr.delegate.Imports(c, r, f)
}

func (imkr inverseMapKindResolver) Embeds(r *rule.Rule, from label.Label) []label.Label {
	r = imkr.inverseMapKind(r)
	return imkr.delegate.Embeds(r, from)
}

func (imkr inverseMapKindResolver) Resolve(c *config.Config, ix *resolve.RuleIndex, rc *repo.RemoteCache, r *rule.Rule, imports interface{}, from label.Label) {
	r = imkr.inverseMapKind(r)
	imkr.delegate.Resolve(c, ix, rc, r, imports, from)
}

func (imkr inverseMapKindResolver) inverseMapKind(r *rule.Rule) *rule.Rule {
	rCopy := *r
	rCopy.SetKind(imkr.fromKind)
	return &rCopy
}
