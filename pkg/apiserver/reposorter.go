package apiserver

import (
	"encoding/json"
	"sort"
)

type GithubSortField int

const (
	StarsField GithubSortField = iota
	ForksField
	IssuesField
	UpdatedField
)

func BottomNRepos(rawJson []byte, sort GithubSortField, numResults int) ([]byte, error) {
	var repos []GithubRepo
	err := json.Unmarshal(rawJson, &repos)
	if err != nil {
		return nil, err
	}

	SortRepos(repos, sort)
	if numResults > len(repos) {
		numResults = len(repos)
	}

	return sortedSliceToBytes(repos[len(repos)-numResults:], sort)
}

// Struct with custom marshaller to encode the return value [["repo",123],...]
type Pair struct {
	Name  string
	Value any
}

func (r *Pair) MarshalJSON() ([]byte, error) {
	arr := []interface{}{r.Name, r.Value}
	return json.Marshal(arr)
}

func sortedSliceToBytes(l []GithubRepo, f GithubSortField) ([]byte, error) {
	ret := make([]Pair, len(l))
	for i, r := range l {
		var p Pair
		p.Name = r.Name
		// This could be cleaner with reflection but I'd rather avoid it
		if f == StarsField {
			p.Value = r.Stars
		} else if f == ForksField {
			p.Value = r.Forks
		} else if f == IssuesField {
			p.Value = r.Issues
		} else {
			p.Value = r.Updated
		}
		ret[i] = p
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

// Sort sorts the argument slice according to the less functions passed to OrderedBy.
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

// Although this is named Less for the interface we inverse
// when sorting values to get descending order.
func (rs *repoSorter) Less(i, j int) bool {
	r1, r2 := &rs.repos[i], &rs.repos[j]
	if rs.compare(r1, r2) {
		return false
	}
	if rs.compare(r2, r1) {
		return true
	}
	// Fields are equal, sort on the name instead
	if r1.Name < r2.Name {
		return true
	}
	return false
}

func sortFieldToFunc(s GithubSortField) compareFunction {
	// Switch statement was longer here :)
	if s == ForksField {
		return compareForks
	} else if s == IssuesField {
		return compareIssues
	} else if s == UpdatedField {
		return compareUpdated
	}
	return compareStars
}

func compareUpdated(r1, r2 *GithubRepo) bool {
	return r1.Updated < r2.Updated
}

func compareForks(r1, r2 *GithubRepo) bool {
	return r1.Forks < r2.Forks
}

func compareStars(r1, r2 *GithubRepo) bool {
	return r1.Stars < r2.Stars
}

func compareIssues(r1, r2 *GithubRepo) bool {
	return r1.Issues < r2.Issues
}
