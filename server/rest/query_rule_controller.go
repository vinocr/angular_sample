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
package rest

import (
	"encoding/json"
	"fmt"
	"github.com/ServiceComb/service-center/server/core"
	pb "github.com/ServiceComb/service-center/server/core/proto"
	"github.com/ServiceComb/service-center/util"
	"github.com/ServiceComb/service-center/util/rest"
	"io/ioutil"
	"net/http"
	"strings"
)

type RuleService struct {
	//
}

func (this *RuleService) URLPatterns() []rest.Route {
	return []rest.Route{
		{rest.HTTP_METHOD_POST, "/registry/v3/microservices/:serviceId/rules", this.AddRule},
		{rest.HTTP_METHOD_GET, "/registry/v3/microservices/:serviceId/rules", this.GetRules},
		{rest.HTTP_METHOD_PUT, "/registry/v3/microservices/:serviceId/rules/:rule_id", this.UpdateRule},
		{rest.HTTP_METHOD_DELETE, "/registry/v3/microservices/:serviceId/rules/:rule_id", this.DeleteRule},
	}
}
func (this *RuleService) AddRule(w http.ResponseWriter, r *http.Request) {
	message, err := ioutil.ReadAll(r.Body)
	if err != nil {
		util.Logger().Error("bory err", err)
		WriteText(http.StatusInternalServerError, fmt.Sprintf("body error %s", err.Error()), w)
		return
	}
	rule := map[string][]*pb.AddOrUpdateServiceRule{}
	err = json.Unmarshal(message, &rule)
	if err != nil {
		util.Logger().Error("Unmarshal error", err)
		WriteText(http.StatusBadRequest, "Unmarshal error", w)
		return
	}

	resp, err := core.ServiceAPI.AddRule(r.Context(), &pb.AddServiceRulesRequest{
		ServiceId: r.URL.Query().Get(":serviceId"),
		Rules:     rule["rules"],
	})
	respInter := resp.Response
	resp.Response = nil
	WriteJsonResponse(respInter, resp, err, w)
}

func (this *RuleService) DeleteRule(w http.ResponseWriter, r *http.Request) {
	rule_id := r.URL.Query().Get(":rule_id")
	ids := strings.Split(rule_id, ",")

	resp, err := core.ServiceAPI.DeleteRule(r.Context(), &pb.DeleteServiceRulesRequest{
		ServiceId: r.URL.Query().Get(":serviceId"),
		RuleIds:   ids,
	})
	WriteTextResponse(resp.GetResponse(), err, "", w)
}

func (this *RuleService) UpdateRule(w http.ResponseWriter, r *http.Request) {
	message, err := ioutil.ReadAll(r.Body)
	if err != nil {
		util.Logger().Error("body err", err)
		WriteText(http.StatusBadRequest, "body error", w)
		return
	}

	rule := pb.AddOrUpdateServiceRule{}
	err = json.Unmarshal(message, &rule)
	if err != nil {
		util.Logger().Error("Unmarshal error", err)
		WriteText(http.StatusBadRequest, "Unmarshal error", w)
		return
	}
	resp, err := core.ServiceAPI.UpdateRule(r.Context(), &pb.UpdateServiceRuleRequest{
		ServiceId: r.URL.Query().Get(":serviceId"),
		RuleId:    r.URL.Query().Get(":rule_id"),
		Rule:      &rule,
	})
	WriteTextResponse(resp.GetResponse(), err, "", w)
}

func (this *RuleService) GetRules(w http.ResponseWriter, r *http.Request) {
	noCache := r.URL.Query().Get("noCache")
	if noCache != "0" && noCache != "1" && strings.TrimSpace(noCache) != "" {
		WriteText(http.StatusBadRequest, "parameter noCache must be 1 or 0", w)
		return
	}
	// TODO 根据attribute查询
	// attribute := r.URL.Query().Get("attribute")

	resp, err := core.ServiceAPI.GetRule(r.Context(), &pb.GetServiceRulesRequest{
		ServiceId: r.URL.Query().Get(":serviceId"),
		NoCache:   noCache == "1",
	})
	respInternal := resp.Response
	resp.Response = nil
	WriteJsonResponse(respInternal, resp, err, w)
}
