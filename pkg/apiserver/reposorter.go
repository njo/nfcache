package apiserver

import (
	"encoding/json"
	"sort"
	"strings"
)

type GithubSortField int

const (
	StarsField GithubSortField = iota
	ForksField
	IssuesField
	UpdatedField
)

func BottomNRepos(reposJSON []byte, field GithubSortField, numResults int) ([]byte, error) {
	if numResults < 0 {
		numResults = 0
	}

	var repos []GithubRepo
	err := json.Unmarshal(reposJSON, &repos)
	if err != nil {
		return nil, err
	}

	SortRepos(repos, field)
	if numResults > len(repos) {
		numResults = len(repos)
	}

	return sortedSliceToJSON(repos[len(repos)-numResults:], field)
}

// Struct with custom marshaller to encode the return value [["repo",123],...]
type repoPair struct {
	Name  string
	Value any
}

func (r *repoPair) MarshalJSON() ([]byte, error) {
	arr := []interface{}{r.Name, r.Value}
	return json.Marshal(arr)
}

func sortedSliceToJSON(repos []GithubRepo, field GithubSortField) ([]byte, error) {
	ret := make([]repoPair, len(repos))
	for i, r := range repos {
		var rp repoPair
		rp.Name = r.Name
		// This could be cleaner with reflection but I'd rather avoid it
		if field == StarsField {
			rp.Value = r.Stars
		} else if field == ForksField {
			rp.Value = r.Forks
		} else if field == IssuesField {
			rp.Value = r.Issues
		} else {
			rp.Value = r.Updated
		}
		ret[i] = rp
	}
	return json.Marshal(ret)
}

// To avoid keeping a fully typed repo struct up to date we only unpack the fields we care about
type GithubRepo struct {
	Name    string `json:"full_name"`
	Updated string `json:"updated_at"` // parsable to a date obj but no need to for sorting purposes
	Forks   int    `json:"forks_count"`
	Stars   int    `json:"stargazers_count"`
	Issues  int    `json:"open_issues_count"`
}

// Sorter skeleton below adapted from the package docs:
// https://pkg.go.dev/sort#example-package-SortMultiKeys
type compareFunction func(p1, p2 *GithubRepo) bool

// Implements sort.Sort interface
type repoSorter struct {
	repos   []GithubRepo
	compare compareFunction
}

// In-Place sort of the provided repo slice on the specified field.
func SortRepos(repos []GithubRepo, field GithubSortField) {
	rs := repoSorter{repos, sortFieldToFunc(field)}
	sort.Sort(&rs)
}

// Len is part of sort.Interface.
func (rs *repoSorter) Len() int {
	return len(rs.repos)
}

// Swap is part of sort.Interface.
func (rs *repoSorter) Swap(i, j int) {
	rs.repos[i], rs.repos[j] = rs.repos[j], rs.repos[i]
}

// This is named Less for the interface but we inverse when sorting values to get descending order.
func (rs *repoSorter) Less(i, j int) bool {
	r1, r2 := &rs.repos[i], &rs.repos[j]
	if rs.compare(r1, r2) {
		return false
	}
	if rs.compare(r2, r1) {
		return true
	}
	// Fields are equal, sort on the name instead
	return strings.ToLower(r1.Name) < strings.ToLower(r2.Name) // We lower because a < B in go
}

// Map sort fields to a function that compares on that field.
func sortFieldToFunc(field GithubSortField) compareFunction {
	if field == ForksField {
		return func(r1, r2 *GithubRepo) bool {
			return r1.Forks < r2.Forks
		}
	} else if field == IssuesField {
		return func(r1, r2 *GithubRepo) bool {
			return r1.Issues < r2.Issues
		}

	} else if field == UpdatedField {
		return func(r1, r2 *GithubRepo) bool {
			return r1.Updated < r2.Updated
		}
	}
	return func(r1, r2 *GithubRepo) bool {
		return r1.Stars < r2.Stars
	}
}
