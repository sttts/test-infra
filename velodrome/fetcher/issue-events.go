/*
Copyright 2016 The Kubernetes Authors.

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
	"github.com/golang/glog"
	"github.com/google/go-github/github"
	"github.com/jinzhu/gorm"
	"k8s.io/test-infra/velodrome/sql"
)

func findLatestEvent(db *gorm.DB) (*int, error) {
	var latestEvent sql.IssueEvent

	query := db.Select("id, event_created_at").Order("event_created_at desc").First(&latestEvent)
	if query.RecordNotFound() {
		return nil, nil
	}
	if query.Error != nil {
		return nil, query.Error
	}

	return &latestEvent.ID, nil
}

// UpdateIssueEvents fetches all events until we find the most recent we
// have in db, and saves everything in database
func UpdateIssueEvents(db *gorm.DB, client ClientInterface) {
	latest, err := findLatestEvent(db)
	if err != nil {
		glog.Error("Failed to find last event: ", err)
		return
	}
	c := make(chan *github.IssueEvent, 500)

	go client.FetchIssueEvents(latest, c)
	for event := range c {
		eventOrm, err := NewIssueEvent(event)
		if err != nil {
			glog.Error("Failed to create issue-event", err)
		}
		db.Create(eventOrm)
	}
}
