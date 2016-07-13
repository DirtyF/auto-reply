package lgtm

import (
	"log"
	"sync"

	"github.com/google/go-github/github"
	"github.com/parkr/auto-reply/ctx"
)

var statusCache = statusMap{data: make(map[string]*statusInfo)}

type statusMap struct {
	sync.Mutex // protects 'data'
	data       map[string]*statusInfo
}

func lgtmContext(owner string) string {
	return owner + "/lgtm"
}

func setStatus(context *ctx.Context, ref prRef, sha string, status *statusInfo) error {
	_, _, err := context.GitHub.Repositories.CreateStatus(
		ref.Repo.Owner, ref.Repo.Name, sha, status.NewStatus(ref.Repo.Owner, ref.Repo.Quorum))
	if err != nil {
		return err
	}

	statusCache.Lock()
	statusCache.data[ref.String()] = status
	statusCache.Unlock()

	return nil
}

func getStatus(context *ctx.Context, ref prRef) (*statusInfo, error) {
	statusCache.Lock()
	cachedStatus, ok := statusCache.data[ref.String()]
	statusCache.Unlock()
	if ok && cachedStatus != nil {
		return cachedStatus, nil
	}

	pr, _, err := context.GitHub.PullRequests.Get(ref.Repo.Owner, ref.Repo.Name, ref.Number)
	if err != nil {
		return nil, err
	}

	statuses, _, err := context.GitHub.Repositories.ListStatuses(ref.Repo.Owner, ref.Repo.Name, *pr.Head.SHA, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var preExistingStatus *github.RepoStatus
	var info *statusInfo
	// Find the status matching context.
	neededContext := lgtmContext(ref.Repo.Owner)
	for _, status := range statuses {
		if *status.Context == neededContext {
			preExistingStatus = status
			info = parseStatus(*pr.Head.SHA, status)
			break
		}
	}

	// None of the contexts matched.
	if preExistingStatus == nil {
		preExistingStatus = newEmptyStatus(ref.Repo.Owner)
		info = parseStatus(*pr.Head.SHA, preExistingStatus)
		setStatus(context, ref, *pr.Head.SHA, info)
	}

	statusCache.Lock()
	statusCache.data[ref.String()] = info
	statusCache.Unlock()

	return info, nil
}

func newEmptyStatus(owner string) *github.RepoStatus {
	return &github.RepoStatus{
		Context:     github.String(lgtmContext(owner)),
		State:       github.String("failure"),
		Description: github.String(descriptionNoLGTMers),
	}
}