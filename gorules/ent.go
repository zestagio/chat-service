//go:build ruleguard

package gorules

import (
	"github.com/quasilyte/go-ruleguard/dsl"
)

func entNoDistinctQuery(m dsl.Matcher) {
	m.Match(`func (r *Repo) $_($*_) ($*_) { $*body }`).
		Where(
			m.File().PkgPath.Matches(`internal/repositories`) &&
				m["body"].Text.Matches(`Query`) &&
				!m["body"].Text.Matches(`QueryContext`) && // Ignore raw SQL.
				!m["body"].Text.Matches(`Unique\(false\)`),
		).
		Report("ent query: lost `Uniq(false)` call")
}
