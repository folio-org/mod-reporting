package reporting

import "strings"
import "testing"
import "github.com/stretchr/testify/assert"

func Test_getReportRepos(t *testing.T) {
	ts := MakeMockHTTPServer()
	defer ts.Close()

	cfg, err := readConfig("../etc/silent.json")
	assert.Nil(t, err)
	server := MakeModReportingServer(cfg, nil, "")
	session, err := NewModReportingSession(server, ts.URL, "dummyTenant", "dummyToken")
	assert.Nil(t, err)

	repos, err := getReportRepos(session.folioSession, "")
	assert.Nil(t, err)
	assert.Equal(t, 4, len(repos))
	assert.Equal(t, "seneyl", repos[0].User)
	assert.Equal(t, "circulation2", repos[0].Dir)
	assert.Equal(t, []string{"11111111-1111-1111-1111-111111111111"}, repos[0].AuthorizedUsers)
	assert.Equal(t, "gitlab", repos[1].Type)
	assert.Equal(t, "https://gitlab.example.com", repos[1].BaseUrl)
	assert.Nil(t, repos[1].AuthorizedUsers)
	assert.Equal(t, ts.URL, repos[2].BaseUrl)
	assert.Nil(t, repos[2].AuthorizedUsers)
	assert.Equal(t, []string{"22222222-2222-2222-2222-222222222222"}, repos[3].AuthorizedUsers)

	matches, err := session.findReportRepos("https://raw.githubusercontent.com/seneyl/test_reports/main/circulation2/circ.sql", "")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(matches))
	assert.Equal(t, "circulation2/circ.sql", matches[0].Path)
	assert.Equal(t, "github:seneyl/test_reports@main:circulation2", matches[0].Repo.String())
}

func Test_findReportRepos(t *testing.T) {
	sha := "0123456789abcdef0123456789abcdef01234567"
	repos := []reportRepo{
		{Type: "github", User: "seneyl", Repo: "test_reports", Branch: "main", Dir: "circulation2"},
		{Type: "github", User: "seneyl", Repo: "test_reports", Branch: "main", Dir: "circulation2/additional_directory/"},
		{User: "seneyl", Repo: "test_reports", Branch: "feature/x"}, // no type means GitHub; no dir means whole repo
		{Type: "gitlab", User: "mike", Repo: "reports", Branch: "dev", Dir: "/"},
		{Type: "gitlab", BaseUrl: "https://gitlab.example.com/", User: "group", Repo: "sql", Branch: "main", Dir: "x"},
		{Type: "bitbucket", User: "seneyl", Repo: "test_reports", Branch: "main"},
	}

	tests := []struct {
		name   string
		url    string
		paths  []string // expected path of each match, in order
		dirs   []string // expected directory of each matching repo, in order
		commit string   // expected commit of every match
	}{
		{
			name:  "github, simple",
			url:   "https://raw.githubusercontent.com/seneyl/test_reports/main/circulation2/circ.sql",
			paths: []string{"circulation2/circ.sql"},
			dirs:  []string{"circulation2"},
		},
		{
			name:  "github, nested repos: most specific first",
			url:   "https://raw.githubusercontent.com/seneyl/test_reports/main/circulation2/additional_directory/circ_by_group.sql",
			paths: []string{"circulation2/additional_directory/circ_by_group.sql", "circulation2/additional_directory/circ_by_group.sql"},
			dirs:  []string{"circulation2/additional_directory/", "circulation2"},
		},
		{
			name:  "github, case-insensitive user and repo",
			url:   "https://raw.githubusercontent.com/Seneyl/Test_Reports/main/circulation2/circ.sql",
			paths: []string{"circulation2/circ.sql"},
			dirs:  []string{"circulation2"},
		},
		{
			name:   "github, commit SHA matches all branches, pending verification",
			url:    "https://raw.githubusercontent.com/seneyl/test_reports/" + sha + "/circulation2/circ.sql",
			paths:  []string{"circulation2/circ.sql", "circulation2/circ.sql"},
			dirs:   []string{"circulation2", ""},
			commit: sha,
		},
		{
			name: "github, SHA that is too short",
			url:  "https://raw.githubusercontent.com/seneyl/test_reports/" + sha[:39] + "/circulation2/circ.sql",
		},
		{
			name:  "github, branch containing slash",
			url:   "https://raw.githubusercontent.com/seneyl/test_reports/feature/x/circulation2/circ.sql",
			paths: []string{"circulation2/circ.sql"},
			dirs:  []string{""},
		},
		{
			name:  "github, fully qualified branch",
			url:   "https://raw.githubusercontent.com/seneyl/test_reports/refs/heads/main/circulation2/additional_directory/circ_by_group.sql",
			paths: []string{"circulation2/additional_directory/circ_by_group.sql", "circulation2/additional_directory/circ_by_group.sql"},
			dirs:  []string{"circulation2/additional_directory/", "circulation2"},
		},
		{
			name:  "github, fully qualified branch containing slash",
			url:   "https://raw.githubusercontent.com/seneyl/test_reports/refs/heads/feature/x/circulation2/circ.sql",
			paths: []string{"circulation2/circ.sql"},
			dirs:  []string{""},
		},
		{
			name: "github, fully qualified unknown branch",
			url:  "https://raw.githubusercontent.com/seneyl/test_reports/refs/heads/other/circulation2/circ.sql",
		},
		{
			name: "github, tag is not a branch",
			url:  "https://raw.githubusercontent.com/seneyl/test_reports/refs/tags/main/circulation2/circ.sql",
		},
		{
			name: "github, file outside directory",
			url:  "https://raw.githubusercontent.com/seneyl/test_reports/main/circulation/circ.sql",
		},
		{
			name: "github, directory-name prefix is not enough",
			url:  "https://raw.githubusercontent.com/seneyl/test_reports/main/circulation2x/circ.sql",
		},
		{
			name: "github, dot-dot escape from directory",
			url:  "https://raw.githubusercontent.com/seneyl/test_reports/main/circulation2/../secret/circ.sql",
		},
		{
			name: "github, encoded dot-dot escape from directory",
			url:  "https://raw.githubusercontent.com/seneyl/test_reports/main/circulation2/%2e%2e/secret/circ.sql",
		},
		{
			name: "github, unknown branch",
			url:  "https://raw.githubusercontent.com/seneyl/test_reports/other/circulation2/circ.sql",
		},
		{
			name: "github, wrong host",
			url:  "https://raw.example.com/seneyl/test_reports/main/circulation2/circ.sql",
		},
		{
			name: "github, http not https",
			url:  "http://raw.githubusercontent.com/seneyl/test_reports/main/circulation2/circ.sql",
		},
		{
			name:  "gitlab, default base URL",
			url:   "https://gitlab.com/api/v4/projects/mike%2Freports/repository/files/a%2Fb%2Fc.sql/raw?ref=dev",
			paths: []string{"a/b/c.sql"},
			dirs:  []string{"/"},
		},
		{
			name: "gitlab, wrong branch",
			url:  "https://gitlab.com/api/v4/projects/mike%2Freports/repository/files/c.sql/raw?ref=main",
		},
		{
			name: "gitlab, duplicate ref",
			url:  "https://gitlab.com/api/v4/projects/mike%2Freports/repository/files/c.sql/raw?ref=dev&ref=other",
		},
		{
			name: "gitlab, extra query parameter",
			url:  "https://gitlab.com/api/v4/projects/mike%2Freports/repository/files/c.sql/raw?ref=dev&x=1",
		},
		{
			name: "gitlab, no ref",
			url:  "https://gitlab.com/api/v4/projects/mike%2Freports/repository/files/c.sql/raw",
		},
		{
			name: "gitlab, malformed query",
			url:  "https://gitlab.com/api/v4/projects/mike%2Freports/repository/files/c.sql/raw?ref=dev;x=1",
		},
		{
			name:  "gitlab, encoded ref",
			url:   "https://gitlab.com/api/v4/projects/mike%2Freports/repository/files/c.sql/raw?ref=d%65v",
			paths: []string{"c.sql"},
			dirs:  []string{"/"},
		},
		{
			name: "gitlab, missing raw suffix",
			url:  "https://gitlab.com/api/v4/projects/mike%2Freports/repository/files/c.sql?ref=dev",
		},
		{
			name:  "gitlab, explicit base URL with trailing slash",
			url:   "https://gitlab.example.com/api/v4/projects/group%2Fsql/repository/files/x%2Fy.sql/raw?ref=main",
			paths: []string{"x/y.sql"},
			dirs:  []string{"x"},
		},
		{
			name: "gitlab, file outside directory",
			url:  "https://gitlab.example.com/api/v4/projects/group%2Fsql/repository/files/y.sql/raw?ref=main",
		},
		{
			name: "not a repository URL",
			url:  "http://localhost:12369/reports/loans.sql",
		},
		{
			name: "unparseable URL",
			url:  "://nope",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matches := findReportRepos(repos, test.url)
			paths := make([]string, 0)
			dirs := make([]string, 0)
			for _, m := range matches {
				paths = append(paths, m.Path)
				dirs = append(dirs, m.Repo.Dir)
				assert.Equal(t, test.commit, m.Commit)
			}
			if test.paths == nil {
				test.paths = []string{}
				test.dirs = []string{}
			}
			assert.Equal(t, test.paths, paths)
			assert.Equal(t, test.dirs, dirs)
		})
	}
}

func Test_verifyCommits(t *testing.T) {
	ts := MakeMockHTTPServer()
	defer ts.Close()
	savedApiUrl := githubApiUrl
	githubApiUrl = ts.URL
	t.Cleanup(func() { githubApiUrl = savedApiUrl })

	server, err := MakeConfiguredServer("../etc/silent.json", ".")
	assert.Nil(t, err)
	session, err := NewModReportingSession(server, ts.URL, "dummyTenant", "dummyToken")
	assert.Nil(t, err)

	main := reportRepo{User: "seneyl", Repo: "test_reports", Branch: "main", Dir: "circulation2"}
	mainWhole := reportRepo{User: "Seneyl", Repo: "test_reports", Branch: "main"}
	feature := reportRepo{User: "seneyl", Repo: "test_reports", Branch: "feature/x"}
	byBranch := reportRepoMatch{Repo: main, Path: "circulation2/circ.sql"}

	// The mock server's answer depends on the first character of the SHA
	sha := func(c string) string { return strings.Repeat(c, 40) }
	bySha := func(repo reportRepo, commit string) reportRepoMatch {
		return reportRepoMatch{Repo: repo, Path: "circulation2/circ.sql", Commit: commit}
	}

	tests := []struct {
		name     string
		matches  []reportRepoMatch
		expected []reportRepoMatch
		errorstr string
	}{
		{name: "no matches", matches: []reportRepoMatch{}, expected: []reportRepoMatch{}},
		{name: "match by branch is not checked", matches: []reportRepoMatch{byBranch}, expected: []reportRepoMatch{byBranch}},
		{name: "ancestor of branch", matches: []reportRepoMatch{bySha(main, sha("a"))}, expected: []reportRepoMatch{bySha(main, sha("a"))}},
		{name: "head of branch", matches: []reportRepoMatch{bySha(main, sha("b"))}, expected: []reportRepoMatch{bySha(main, sha("b"))}},
		{name: "ahead of branch", matches: []reportRepoMatch{bySha(main, sha("c"))}, expected: []reportRepoMatch{}},
		{name: "diverged from branch", matches: []reportRepoMatch{bySha(main, sha("d"))}, expected: []reportRepoMatch{}},
		{name: "unknown commit", matches: []reportRepoMatch{bySha(main, sha("e"))}, expected: []reportRepoMatch{}},
		{
			name:     "on one branch but not another",
			matches:  []reportRepoMatch{bySha(main, sha("a")), bySha(mainWhole, sha("a")), bySha(feature, sha("a"))},
			expected: []reportRepoMatch{bySha(main, sha("a")), bySha(mainWhole, sha("a"))},
		},
		{
			name:     "branch containing slash",
			matches:  []reportRepoMatch{bySha(feature, sha("b"))},
			expected: []reportRepoMatch{bySha(feature, sha("b"))},
		},
		{
			name:     "GitHub refuses",
			matches:  []reportRepoMatch{bySha(main, sha("f"))},
			errorstr: "could not ask GitHub whether commit " + sha("f") + " is on branch main of seneyl/test_reports: 403 Forbidden",
		},
		{
			name:     "GitHub sends bad JSON",
			matches:  []reportRepoMatch{bySha(main, sha("9"))},
			errorstr: "could not decode GitHub's comparison of commit " + sha("9") + " with branch main of seneyl/test_reports: ",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verified, err := session.verifyCommits(test.matches)
			if test.errorstr != "" {
				assert.ErrorContains(t, err, test.errorstr)
				return
			}
			assert.Nil(t, err)
			assert.Equal(t, test.expected, verified)
		})
	}
}

func Test_checkAccess(t *testing.T) {
	alice := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	bob := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	open := reportRepoMatch{Repo: reportRepo{User: "u", Repo: "r", Branch: "main", Dir: "open"}}
	forAlice := reportRepoMatch{Repo: reportRepo{User: "u", Repo: "r", Branch: "main", Dir: "a", AuthorizedUsers: []string{alice}}}
	forBob := reportRepoMatch{Repo: reportRepo{User: "u", Repo: "r", Branch: "main", Dir: "b", AuthorizedUsers: []string{bob}}}

	tests := []struct {
		name    string
		matches []reportRepoMatch
		userId  string
		allowed bool
		message string // expected error message, if not allowed
	}{
		{name: "open repository, user", matches: []reportRepoMatch{open}, userId: alice, allowed: true},
		{name: "open repository, no user", matches: []reportRepoMatch{open}, allowed: true},
		{name: "restricted repository, authorized user", matches: []reportRepoMatch{forAlice}, userId: alice, allowed: true},
		{
			name:    "restricted repository, unauthorized user",
			matches: []reportRepoMatch{forAlice},
			userId:  bob,
			message: "user " + bob + " is not authorized to run reports from repository github:u/r@main:a",
		},
		{
			name:    "restricted repository, no user",
			matches: []reportRepoMatch{forAlice},
			message: "request has no X-Okapi-User-Id header, so cannot check whether the user is authorized to run reports from repository github:u/r@main:a",
		},
		{name: "restricted and open repositories, no user", matches: []reportRepoMatch{forAlice, open}, allowed: true},
		{name: "two restricted repositories, authorized for second", matches: []reportRepoMatch{forAlice, forBob}, userId: bob, allowed: true},
		{
			name:    "two restricted repositories, authorized for neither",
			matches: []reportRepoMatch{forAlice, forBob},
			userId:  "cccccccc-cccc-cccc-cccc-cccccccccccc",
			message: "user cccccccc-cccc-cccc-cccc-cccccccccccc is not authorized to run reports from repository github:u/r@main:a or github:u/r@main:b",
		},
		{
			name:    "two restricted repositories, no user",
			matches: []reportRepoMatch{forAlice, forBob},
			message: "request has no X-Okapi-User-Id header, so cannot check whether the user is authorized to run reports from repository github:u/r@main:a or github:u/r@main:b",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := checkAccess(test.matches, test.userId)
			// The status of a refusal is decided by handleReport, and
			// checked by Test_reportRepoChecks
			if test.allowed {
				assert.Nil(t, err)
			} else {
				assert.EqualError(t, err, test.message)
			}
		})
	}
}
