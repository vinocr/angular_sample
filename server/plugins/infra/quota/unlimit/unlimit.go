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
package unlimit

import (
	"github.com/ServiceComb/service-center/server/infra/quota"
	"golang.org/x/net/context"
	"github.com/astaxie/beego"
	apt "github.com/ServiceComb/service-center/server/core"
)

type Unlimit struct {
}

func New() quota.QuotaManager {
	return &Unlimit{}
}
func init() {
	quataType := beego.AppConfig.DefaultString("quota_plugin", "")
	if quataType != "unlimit" {
		return
	}
	apt.MicroServiceValidator.GetRule("Schemas").Length = 0
	quota.QuotaPlugins["unlimit"] = New
}

func (q *Unlimit) Apply4Quotas(ctx context.Context, quotaType quota.ResourceType, tenant string, serviceId string, quotaSize int16) (bool, error) {
	return true, nil
}

func (q *Unlimit) ReportCurrentQuotasUsage(ctx context.Context, quotaType int, usedQuotaSize int16) bool {
	return false
}
