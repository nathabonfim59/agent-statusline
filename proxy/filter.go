package proxy

import "strings"

type DomainFilter struct {
	domains []string
}

func NewDomainFilter(domains []string) *DomainFilter {
	return &DomainFilter{domains: domains}
}

func (f *DomainFilter) ShouldIntercept(host string) bool {
	host = strings.TrimSuffix(host, ".")
	for _, d := range f.domains {
		if host == d {
			return true
		}
		if strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}
