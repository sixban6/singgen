package singgen

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"

	"github.com/sixban6/singgen/internal/transformer"
	"github.com/sixban6/singgen/pkg/model"
)

// ErrInvalidNodeRule is returned when an include/exclude/rename pattern is not a valid regular expression
var ErrInvalidNodeRule = errors.New("invalid node rule")

// RenameRule renames nodes whose name matches Match (a Go regular expression).
// Replace may reference capture groups, e.g. "$1".
type RenameRule struct {
	Match   string `json:"match" yaml:"match"`
	Replace string `json:"replace" yaml:"replace"`
}

type compiledRename struct {
	re      *regexp.Regexp
	replace string
}

// nodeRules is the compiled form of include/exclude/rename for one subscription
type nodeRules struct {
	include *regexp.Regexp
	exclude *regexp.Regexp
	rename  []compiledRename
}

func (r nodeRules) empty() bool {
	return r.include == nil && r.exclude == nil && len(r.rename) == 0
}

// nodeRulesFor resolves the rules for a subscription: subscription-level values
// override the global ones (an explicitly empty value disables the global rule).
func (mc *MultiConfig) nodeRulesFor(sub SubscriptionConfig) (nodeRules, error) {
	include, exclude, rename := mc.Global.Include, mc.Global.Exclude, mc.Global.Rename
	if sub.Include != nil {
		include = *sub.Include
	}
	if sub.Exclude != nil {
		exclude = *sub.Exclude
	}
	if sub.Rename != nil {
		rename = sub.Rename
	}

	var rules nodeRules
	var err error
	if rules.include, err = compileRule("include", sub.Name, include); err != nil {
		return rules, err
	}
	if rules.exclude, err = compileRule("exclude", sub.Name, exclude); err != nil {
		return rules, err
	}
	for i, rr := range rename {
		re, err := compileRule(fmt.Sprintf("rename[%d]", i), sub.Name, rr.Match)
		if err != nil {
			return rules, err
		}
		if re != nil {
			rules.rename = append(rules.rename, compiledRename{re: re, replace: rr.Replace})
		}
	}
	return rules, nil
}

func compileRule(field, subName, pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("%w: subscription %q %s %q: %v", ErrInvalidNodeRule, subName, field, pattern, err)
	}
	return re, nil
}

// apply filters and renames nodes by their name as provided by the subscription
// (before emoji removal and the subscription prefix). Nodes without a name use
// their server address. Returns the kept nodes and the number of dropped ones.
func (r nodeRules) apply(nodes []model.Node) ([]model.Node, int) {
	if r.empty() {
		return nodes, 0
	}
	kept := make([]model.Node, 0, len(nodes))
	for _, node := range nodes {
		name := node.Tag
		if name == "" {
			name = node.Addr
		}
		if r.exclude != nil && r.exclude.MatchString(name) {
			continue
		}
		if r.include != nil && !r.include.MatchString(name) {
			continue
		}
		for _, rr := range r.rename {
			name = rr.re.ReplaceAllString(name, rr.replace)
		}
		node.Tag = name
		kept = append(kept, node)
	}
	return kept, len(nodes) - len(kept)
}

// dedupeOutboundsByServer keeps only the first outbound for each protocol+server+port.
func dedupeOutboundsByServer(outbounds []transformer.Outbound) ([]transformer.Outbound, int) {
	seen := make(map[string]bool, len(outbounds))
	kept := make([]transformer.Outbound, 0, len(outbounds))
	for _, ob := range outbounds {
		key := ob.Type + "|" + net.JoinHostPort(ob.Server, strconv.Itoa(int(ob.ServerPort)))
		if seen[key] {
			continue
		}
		seen[key] = true
		kept = append(kept, ob)
	}
	return kept, len(outbounds) - len(kept)
}
