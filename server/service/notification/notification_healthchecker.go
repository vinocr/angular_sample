//Copyright 2017 Huawei Technologies Co., Ltd
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
package notification

import "github.com/ServiceComb/service-center/util"

const (
	NOTIFY_SERVER_CHECKER_NAME  = "__HealthChecker__"
	NOTIFY_SERVER_CHECK_SUBJECT = "__NotifyServerHealthCheck__"
)

//Notifier 健康检查
type NotifyServiceHealthChecker struct {
	BaseSubscriber
}

type NotifyServiceHealthCheckJob struct {
	BaseNotifyJob
	ErrorSubscriber Subscriber
}

func (s *NotifyServiceHealthChecker) OnMessage(job NotifyJob) {
	j := job.(*NotifyServiceHealthCheckJob)
	err := j.ErrorSubscriber.Err()
	util.Logger().Warnf(err, "notify server remove watcher %s %s",
		j.ErrorSubscriber.Subject(), j.ErrorSubscriber.Id())
	s.Service().RemoveSubscriber(j.ErrorSubscriber)
}

func NewNotifyServiceHealthChecker() *NotifyServiceHealthChecker {
	return &NotifyServiceHealthChecker{
		BaseSubscriber: BaseSubscriber{
			id:      NOTIFY_SERVER_CHECKER_NAME,
			subject: NOTIFY_SERVER_CHECK_SUBJECT,
			nType:   NOTIFTY,
		},
	}
}

func NewNotifyServiceHealthCheckJob(s Subscriber) *NotifyServiceHealthCheckJob {
	return &NotifyServiceHealthCheckJob{
		BaseNotifyJob: BaseNotifyJob{
			subscriberId: NOTIFY_SERVER_CHECKER_NAME,
			subject:      NOTIFY_SERVER_CHECK_SUBJECT,
			nType:        NOTIFTY,
		},
		ErrorSubscriber: s,
	}
}
