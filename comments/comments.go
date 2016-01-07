package comments

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/parkr/auto-reply/Godeps/_workspace/src/github.com/google/go-github/github"
)

type CommentHandler func(client *github.Client, comment github.IssueCommentEvent) error

type CommentsHandler struct {
	client               *github.Client
	issueCommentHandlers []CommentHandler
	pullCommentHandlers  []CommentHandler
}

// NewHandler returns an HTTP handler which deprecates repositories
// by closing new issues with a comment directing attention elsewhere.
func NewHandler(client *github.Client, issuesHandlers []CommentHandler, pullRequestsHandlers []CommentHandler) *CommentsHandler {
	return &CommentsHandler{
		client:               client,
		issueCommentHandlers: issuesHandlers,
		pullCommentHandlers:  pullRequestsHandlers,
	}
}

func (h *CommentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-GitHub-Event") != "issue_comment" {
		log.Println("received non-issues event for comments. ignoring.")
		http.Error(w, "not an issue_comment event.", 200)
		return
	}

	var event github.IssueCommentEvent
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		log.Println("error unmarshalling issue comment stuffs:", err)
		http.Error(w, "bad json", 400)
		return
	}

	var handlers []CommentHandler
	if isPullRequest(event) {
		handlers = h.pullCommentHandlers
	} else {
		handlers = h.issueCommentHandlers
	}

	for _, handler := range handlers {
		go handler(h.client, event)
	}

	fmt.Fprintf(w, "fired %d handlers", len(handlers))
}

func isPullRequest(event github.IssueCommentEvent) bool {
	return event.Issue.PullRequestLinks != nil
}