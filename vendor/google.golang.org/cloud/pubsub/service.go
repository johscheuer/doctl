// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pubsub

import (
	"net/http"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
	raw "google.golang.org/api/pubsub/v1"
)

// service provides an internal abstraction to isolate the generated
// PubSub API; most of this package uses this interface instead.
// The single implementation, *apiService, contains all the knowledge
// of the generated PubSub API (except for that present in legacy code).
type service interface {
	createSubscription(ctx context.Context, topicName, subName string, ackDeadline time.Duration, pushConfig *PushConfig) error
	getSubscriptionConfig(ctx context.Context, subName string) (*SubscriptionConfig, string, error)
	listProjectSubscriptions(ctx context.Context, projName string) ([]string, error)
	deleteSubscription(ctx context.Context, name string) error
	subscriptionExists(ctx context.Context, name string) (bool, error)

	createTopic(ctx context.Context, name string) error
	deleteTopic(ctx context.Context, name string) error
	topicExists(ctx context.Context, name string) (bool, error)
	listProjectTopics(ctx context.Context, projName string) ([]string, error)
	listTopicSubscriptions(ctx context.Context, topicName string) ([]string, error)

	modifyAckDeadline(ctx context.Context, subName string, deadline time.Duration, ackIDs []string) error
}

type apiService struct {
	s *raw.Service
}

func newPubSubService(client *http.Client, endpoint string) (*apiService, error) {
	s, err := raw.New(client)
	if err != nil {
		return nil, err
	}
	s.BasePath = endpoint

	return &apiService{s: s}, nil
}

func (s *apiService) createSubscription(ctx context.Context, topicName, subName string, ackDeadline time.Duration, pushConfig *PushConfig) error {
	var rawPushConfig *raw.PushConfig
	if pushConfig != nil {
		rawPushConfig = &raw.PushConfig{
			Attributes:   pushConfig.Attributes,
			PushEndpoint: pushConfig.Endpoint,
		}
	}
	rawSub := &raw.Subscription{
		AckDeadlineSeconds: int64(ackDeadline.Seconds()),
		PushConfig:         rawPushConfig,
		Topic:              topicName,
	}
	_, err := s.s.Projects.Subscriptions.Create(subName, rawSub).Context(ctx).Do()
	return err
}

func (s *apiService) getSubscriptionConfig(ctx context.Context, subName string) (*SubscriptionConfig, string, error) {
	rawSub, err := s.s.Projects.Subscriptions.Get(subName).Context(ctx).Do()
	if err != nil {
		return nil, "", err
	}
	sub := &SubscriptionConfig{
		AckDeadline: time.Second * time.Duration(rawSub.AckDeadlineSeconds),
		PushConfig: PushConfig{
			Endpoint:   rawSub.PushConfig.PushEndpoint,
			Attributes: rawSub.PushConfig.Attributes,
		},
	}
	return sub, rawSub.Topic, err
}

func (s *apiService) listProjectSubscriptions(ctx context.Context, projName string) ([]string, error) {
	subs := []string{}
	err := s.s.Projects.Subscriptions.List(projName).
		Pages(ctx, func(res *raw.ListSubscriptionsResponse) error {
			for _, s := range res.Subscriptions {
				subs = append(subs, s.Name)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func (s *apiService) deleteSubscription(ctx context.Context, name string) error {
	_, err := s.s.Projects.Subscriptions.Delete(name).Context(ctx).Do()
	return err
}

func (s *apiService) subscriptionExists(ctx context.Context, name string) (bool, error) {
	_, err := s.s.Projects.Subscriptions.Get(name).Context(ctx).Do()
	if err == nil {
		return true, nil
	}
	if e, ok := err.(*googleapi.Error); ok && e.Code == http.StatusNotFound {
		return false, nil
	}
	return false, err
}

func (s *apiService) createTopic(ctx context.Context, name string) error {
	// Note: The raw API expects a Topic body, but ignores it.
	_, err := s.s.Projects.Topics.Create(name, &raw.Topic{}).
		Context(ctx).
		Do()
	return err
}

func (s *apiService) listProjectTopics(ctx context.Context, projName string) ([]string, error) {
	topics := []string{}
	err := s.s.Projects.Topics.List(projName).
		Pages(ctx, func(res *raw.ListTopicsResponse) error {
			for _, topic := range res.Topics {
				topics = append(topics, topic.Name)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return topics, nil
}

func (s *apiService) deleteTopic(ctx context.Context, name string) error {
	_, err := s.s.Projects.Topics.Delete(name).Context(ctx).Do()
	return err
}

func (s *apiService) topicExists(ctx context.Context, name string) (bool, error) {
	_, err := s.s.Projects.Topics.Get(name).Context(ctx).Do()
	if err == nil {
		return true, nil
	}
	if e, ok := err.(*googleapi.Error); ok && e.Code == http.StatusNotFound {
		return false, nil
	}
	return false, err
}

func (s *apiService) listTopicSubscriptions(ctx context.Context, topicName string) ([]string, error) {
	subs := []string{}
	err := s.s.Projects.Topics.Subscriptions.List(topicName).
		Pages(ctx, func(res *raw.ListTopicSubscriptionsResponse) error {
			for _, s := range res.Subscriptions {
				subs = append(subs, s)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func (s *apiService) modifyAckDeadline(ctx context.Context, subName string, deadline time.Duration, ackIDs []string) error {
	req := &raw.ModifyAckDeadlineRequest{
		AckDeadlineSeconds: int64(deadline.Seconds()),
		AckIds:             ackIDs,
	}
	_, err := s.s.Projects.Subscriptions.ModifyAckDeadline(subName, req).
		Context(ctx).
		Do()
	return err
}
