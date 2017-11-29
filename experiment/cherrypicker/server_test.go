/*
Copyright 2017 The Kubernetes Authors.

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
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/git/localgit"
	"k8s.io/test-infra/prow/github"
)

type fghc struct {
	sync.Mutex
	pr       *github.PullRequest
	isMember bool

	comments   []string
	prs        []string
	prComments []github.IssueComment
	createdNum int
}

func (f *fghc) GetPullRequest(org, repo string, number int) (*github.PullRequest, error) {
	f.Lock()
	defer f.Unlock()
	return f.pr, nil
}

func (f *fghc) CreateComment(org, repo string, number int, comment string) error {
	f.Lock()
	defer f.Unlock()
	f.comments = append(f.comments, fmt.Sprintf("%s/%s#%d %s", org, repo, number, comment))
	return nil
}

func (f *fghc) IsMember(org, user string) (bool, error) {
	f.Lock()
	defer f.Unlock()
	return f.isMember, nil
}

var expectedFmt = `repo=%s title=%q body=%q head=%s base=%s maintainer_can_modify=%t`

func (f *fghc) CreatePullRequest(org, repo, title, body, head, base string, canModify bool) (int, error) {
	f.Lock()
	defer f.Unlock()
	f.prs = append(f.prs, fmt.Sprintf(expectedFmt, org+"/"+repo, title, body, head, base, canModify))
	return f.createdNum, nil
}

func (f *fghc) ListIssueComments(org, repo string, number int) ([]github.IssueComment, error) {
	return f.prComments, nil
}

func (f *fghc) CreateFork(org, repo string) error {
	return nil
}

var initialFiles = map[string][]byte{
	"bar.go": []byte(`// Package bar does an interesting thing.
package bar

// Foo does a thing.
func Foo(wow int) int {
	return 42 + wow
}
`),
}

var patch = []byte(`From af468c9e69dfdf39db591f1e3e8de5b64b0e62a2 Mon Sep 17 00:00:00 2001
From: Wise Guy <wise@guy.com>
Date: Thu, 19 Oct 2017 15:14:36 +0200
Subject: [PATCH] Update magic number

---
 bar.go | 3 ++-
 1 file changed, 2 insertions(+), 1 deletion(-)

diff --git a/bar.go b/bar.go
index 1ea52dc..5bd70a9 100644
--- a/bar.go
+++ b/bar.go
@@ -3,5 +3,6 @@ package bar

 // Foo does a thing.
 func Foo(wow int) int {
-	return 42 + wow
+	// Needs to be 49 because of a reason.
+	return 49 + wow
 }
`)

func TestCherryPickIC(t *testing.T) {
	lg, c, err := localgit.New()
	if err != nil {
		t.Fatalf("Making localgit: %v", err)
	}
	defer func() {
		if err := lg.Clean(); err != nil {
			t.Errorf("Cleaning up localgit: %v", err)
		}
		if err := c.Clean(); err != nil {
			t.Errorf("Cleaning up client: %v", err)
		}
	}()
	if err := lg.MakeFakeRepo("foo", "bar"); err != nil {
		t.Fatalf("Making fake repo: %v", err)
	}
	if err := lg.AddCommit("foo", "bar", initialFiles); err != nil {
		t.Fatalf("Adding initial commit: %v", err)
	}
	if err := lg.CheckoutNewBranch("foo", "bar", "stage"); err != nil {
		t.Fatalf("Checking out pull branch: %v", err)
	}

	ghc := &fghc{
		pr: &github.PullRequest{
			Base: github.PullRequestBranch{
				Ref: "master",
			},
			Merged: true,
		},
		isMember:   true,
		createdNum: 3,
	}
	ic := github.IssueCommentEvent{
		Action: github.IssueCommentActionCreated,
		Repo: github.Repo{
			Owner: github.User{
				Login: "foo",
			},
			Name:     "bar",
			FullName: "foo/bar",
		},
		Issue: github.Issue{
			Number:      2,
			State:       "closed",
			PullRequest: &struct{}{},
		},
		Comment: github.IssueComment{
			User: github.User{
				Login: "wiseguy",
			},
			Body: "/cherrypick stage",
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("test request %+v", r)
		expected := "/raw/foo/bar/pull/2.patch"
		if r.URL.Path != expected {
			t.Fatalf("wrong link was queried: %s, expected: %s", r.URL.Path, expected)
		}
		w.Write(patch)
	}))
	defer ts.Close()

	botName := "ci-robot"
	expectedRepo := "foo/bar"
	expectedTitle := "Automated cherry-pick of #2 on stage"
	expectedBody := "This is an automated cherry-pick of #2\n\n/assign wiseguy"
	expectedBase := "stage"
	expectedHead := fmt.Sprintf(botName+":"+cherryPickBranchFmt, 2, expectedBase)
	expected := fmt.Sprintf(expectedFmt, expectedRepo, expectedTitle, expectedBody, expectedHead, expectedBase, true)

	s := &Server{
		credentials: "012345",
		botName:     botName,
		gc:          c,
		push:        func(botName, credentials, repo, newBranch string) error { return nil },
		ghc:         ghc,
		hmacSecret:  []byte("sha=abcdefg"),
		bare:        ts.Client(),
		patchURL:    ts.URL,
		log:         logrus.StandardLogger().WithField("client", "cherrypicker"),
		repos:       []github.Repo{{Fork: true, FullName: "ci-robot/bar"}},
	}

	if err := s.handleIssueComment(ic); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if ghc.prs[0] != expected {
		t.Errorf("Expected (%d):\n%s\nGot (%d):\n%+v\n", len(expected), expected, len(ghc.prs[0]), ghc.prs[0])
	}
}

func TestCherryPickPR(t *testing.T) {
	lg, c, err := localgit.New()
	if err != nil {
		t.Fatalf("Making localgit: %v", err)
	}
	defer func() {
		if err := lg.Clean(); err != nil {
			t.Errorf("Cleaning up localgit: %v", err)
		}
		if err := c.Clean(); err != nil {
			t.Errorf("Cleaning up client: %v", err)
		}
	}()
	if err := lg.MakeFakeRepo("foo", "bar"); err != nil {
		t.Fatalf("Making fake repo: %v", err)
	}
	if err := lg.AddCommit("foo", "bar", initialFiles); err != nil {
		t.Fatalf("Adding initial commit: %v", err)
	}
	if err := lg.CheckoutNewBranch("foo", "bar", "release-1.5"); err != nil {
		t.Fatalf("Checking out pull branch: %v", err)
	}

	ghc := &fghc{
		prComments: []github.IssueComment{
			{
				User: github.User{
					Login: "developer",
				},
				Body: "a review comment",
			},
			{
				User: github.User{
					Login: "approver",
				},
				Body: "/cherrypick release-1.5",
			},
			{
				User: github.User{
					Login: "approver",
				},
				Body: "/approve",
			},
		},
		isMember:   true,
		createdNum: 3,
	}
	pr := github.PullRequestEvent{
		Action: github.PullRequestActionClosed,
		PullRequest: github.PullRequest{
			Base: github.PullRequestBranch{
				Ref: "master",
				Repo: github.Repo{
					Owner: github.User{
						Login: "foo",
					},
					Name: "bar",
				},
			},
			Number:   2,
			Merged:   true,
			MergeSHA: new(string),
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("test request %+v", r)
		expected := "/raw/foo/bar/pull/2.patch"
		if r.URL.Path != expected {
			t.Fatalf("wrong link was queried: %s, expected: %s", r.URL.Path, expected)
		}
		w.Write(patch)
	}))
	defer ts.Close()

	botName := "ci-robot"
	expectedRepo := "foo/bar"
	expectedTitle := "Automated cherry-pick of #2 on release-1.5"
	expectedBody := "This is an automated cherry-pick of #2\n\n/assign approver"
	expectedBase := "release-1.5"
	expectedHead := fmt.Sprintf(botName+":"+cherryPickBranchFmt, 2, expectedBase)
	expected := fmt.Sprintf(expectedFmt, expectedRepo, expectedTitle, expectedBody, expectedHead, expectedBase, true)

	s := &Server{
		credentials: "012345",
		botName:     botName,
		gc:          c,
		push:        func(botName, credentials, repo, newBranch string) error { return nil },
		ghc:         ghc,
		hmacSecret:  []byte("sha=abcdefg"),
		bare:        ts.Client(),
		patchURL:    ts.URL,
		log:         logrus.StandardLogger().WithField("client", "cherrypicker"),
		repos:       []github.Repo{{Fork: true, FullName: "ci-robot/bar"}},
	}

	if err := s.handlePullRequest(pr); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if ghc.prs[0] != expected {
		t.Errorf("Expected (%d):\n%s\nGot (%d):\n%+v\n", len(expected), expected, len(ghc.prs[0]), ghc.prs[0])
	}
}