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
	"github.com/ServiceComb/service-center/server/core"
	pb "github.com/ServiceComb/service-center/server/core/proto"
	"github.com/ServiceComb/service-center/util"
	"github.com/ServiceComb/service-center/util/rest"
	"io/ioutil"
	"net/http"
	"strings"
)

type MicroServiceService struct {
	//
}

func (this *MicroServiceService) URLPatterns() []rest.Route {
	return []rest.Route{
		{rest.HTTP_METHOD_GET, "/registry/v3/existence", this.GetExistence},
		{rest.HTTP_METHOD_GET, "/registry/v3/microservices", this.GetServices},
		{rest.HTTP_METHOD_GET, "/registry/v3/microservices/:serviceId", this.GetServiceOne},
		{rest.HTTP_METHOD_POST, "/registry/v3/microservices", this.Register},
		{rest.HTTP_METHOD_PUT, "/registry/v3/microservices/:serviceId/properties", this.Update},
		{rest.HTTP_METHOD_DELETE, "/registry/v3/microservices/:serviceId", this.Unregister},
		{rest.HTTP_METHOD_GET, "/registry/v3/microservices/:serviceId/schemas/:schemaId", this.GetSchemas},
		{rest.HTTP_METHOD_PUT, "/registry/v3/microservices/:serviceId/schemas/:schemaId", this.ModifySchemas},
		{rest.HTTP_METHOD_DELETE, "/registry/v3/microservices/:serviceId/schemas/:schemaId", this.DeleteSchemas},

		{rest.HTTP_METHOD_PUT, "/registry/v3/dependencies", this.CreateDependenciesForMicroServices},
		{rest.HTTP_METHOD_GET, "/registry/v3/microservices/:consumerId/providers", this.GetConProDependencies},
		{rest.HTTP_METHOD_GET, "/registry/v3/microservices/:providerId/consumers", this.GetProConDependencies},
		{rest.HTTP_METHOD_DELETE, "/registry/v3/microservices", this.UnregisterServices},
	}
}

func (this *MicroServiceService) GetSchemas(w http.ResponseWriter, r *http.Request) {
	noCache := r.URL.Query().Get("noCache")
	if noCache != "0" && noCache != "1" && strings.TrimSpace(noCache) != "" {
		WriteText(http.StatusBadRequest, "parameter noCache must be 1 or 0", w)
		return
	}
	request := &pb.GetSchemaRequest{
		ServiceId: r.URL.Query().Get(":serviceId"),
		SchemaId:  r.URL.Query().Get(":schemaId"),
		NoCache:   noCache == "1",
	}
	resp, err := core.ServiceAPI.GetSchemaInfo(r.Context(), request)
	if len(resp.Schema) != 0 {
		resp.Response = nil
		WriteJsonObject(http.StatusOK, resp, w)
		return
	}
	WriteTextResponse(resp.GetResponse(), err, "", w)
}

func (this *MicroServiceService) ModifySchemas(w http.ResponseWriter, r *http.Request) {
	message, err := ioutil.ReadAll(r.Body)
	if err != nil {
		util.Logger().Error("body err", err)
		WriteText(http.StatusBadRequest, err.Error(), w)
		return
	}
	request := &pb.ModifySchemaRequest{
		ServiceId: r.URL.Query().Get(":serviceId"),
		SchemaId:  r.URL.Query().Get(":schemaId"),
	}
	err = json.Unmarshal(message, request)
	if err != nil {
		util.Logger().Error("Unmarshal error", err)
		WriteText(http.StatusBadRequest, err.Error(), w)
		return
	}
	resp, err := core.ServiceAPI.ModifySchema(r.Context(), request)
	WriteTextResponse(resp.GetResponse(), err, "", w)
}

func (this *MicroServiceService) DeleteSchemas(w http.ResponseWriter, r *http.Request) {
	request := &pb.DeleteSchemaRequest{
		ServiceId: r.URL.Query().Get(":serviceId"),
		SchemaId:  r.URL.Query().Get(":schemaId"),
	}
	resp, err := core.ServiceAPI.DeleteSchema(r.Context(), request)
	WriteTextResponse(resp.GetResponse(), err, "", w)
}

func (this *MicroServiceService) Register(w http.ResponseWriter, r *http.Request) {
	message, err := ioutil.ReadAll(r.Body)
	if err != nil {
		util.Logger().Error("body err", err)
		WriteText(http.StatusBadRequest, err.Error(), w)
		return
	}
	var request pb.CreateServiceRequest
	err = json.Unmarshal(message, &request)
	if err != nil {
		util.Logger().Error("Unmarshal error", err)
		WriteText(http.StatusBadRequest, err.Error(), w)
		return
	}
	resp, err := core.ServiceAPI.Create(r.Context(), &request)
	if err != nil {
		WriteText(http.StatusInternalServerError, err.Error(), w)
		return
	}
	if resp.GetResponse().Code != pb.Response_SUCCESS {
		WriteText(http.StatusBadRequest, resp.GetResponse().Message, w)
		return
	}
	resp.Response = nil
	WriteJsonObject(http.StatusOK, resp, w)
}

func (this *MicroServiceService) Update(w http.ResponseWriter, r *http.Request) {
	message, err := ioutil.ReadAll(r.Body)
	if err != nil {
		util.Logger().Error("body err", err)
		WriteText(http.StatusBadRequest, err.Error(), w)
		return
	}
	request := &pb.UpdateServicePropsRequest{
		ServiceId: r.URL.Query().Get(":serviceId"),
	}
	err = json.Unmarshal(message, request)
	if err != nil {
		util.Logger().Error("Unmarshal error", err)
		WriteText(http.StatusBadRequest, err.Error(), w)
		return
	}
	resp, err := core.ServiceAPI.UpdateProperties(r.Context(), request)
	WriteTextResponse(resp.GetResponse(), err, "", w)
}

func (this *MicroServiceService) Unregister(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("force")
	serviceId := r.URL.Query().Get(":serviceId")
	util.Logger().Warnf(nil, "Service %s unregists, force is %s.", serviceId, force)
	if force != "0" && force != "1" && strings.TrimSpace(force) != "" {
		WriteText(http.StatusBadRequest, "parameter force must be 1 or 0", w)
		return
	}
	request := &pb.DeleteServiceRequest{
		ServiceId: serviceId,
		Force:     force == "1",
	}
	resp, err := core.ServiceAPI.Delete(r.Context(), request)
	WriteTextResponse(resp.GetResponse(), err, "", w)
}

func (this *MicroServiceService) GetServices(w http.ResponseWriter, r *http.Request) {
	noCache := r.URL.Query().Get("noCache")
	if noCache != "0" && noCache != "1" && strings.TrimSpace(noCache) != "" {
		WriteText(http.StatusBadRequest, "parameter noCache must be 1 or 0", w)
		return
	}
	request := &pb.GetServicesRequest{
		NoCache: noCache == "1",
	}
	util.Logger().Debugf("tenant is %s", util.ParseTenant(r.Context()))
	resp, err := core.ServiceAPI.GetServices(r.Context(), request)
	if err != nil {
		WriteText(http.StatusInternalServerError, err.Error(), w)
		return
	}
	if resp.GetResponse().Code != pb.Response_SUCCESS {
		WriteText(http.StatusBadRequest, resp.GetResponse().Message, w)
		return
	}
	resp.Response = nil
	WriteJsonObject(http.StatusOK, resp, w)
}

func (this *MicroServiceService) GetExistence(w http.ResponseWriter, r *http.Request) {
	noCache := r.URL.Query().Get("noCache")
	if noCache != "0" && noCache != "1" && strings.TrimSpace(noCache) != "" {
		WriteText(http.StatusBadRequest, "parameter noCache must be 1 or 0", w)
		return
	}
	request := &pb.GetExistenceRequest{
		Type:        r.URL.Query().Get("type"),
		AppId:       r.URL.Query().Get("appId"),
		ServiceName: r.URL.Query().Get("serviceName"),
		Version:     r.URL.Query().Get("version"),
		ServiceId:   r.URL.Query().Get("serviceId"),
		SchemaId:    r.URL.Query().Get("schemaId"),
		NoCache:     noCache == "1",
	}
	resp, err := core.ServiceAPI.Exist(r.Context(), request)
	if err != nil {
		WriteText(http.StatusInternalServerError, err.Error(), w)
		return
	}
	if resp.GetResponse().Code != pb.Response_SUCCESS {
		WriteText(http.StatusBadRequest, resp.GetResponse().Message, w)
		return
	}
	resp.Response = nil
	WriteJsonObject(http.StatusOK, resp, w)
}

func (this *MicroServiceService) GetServiceOne(w http.ResponseWriter, r *http.Request) {
	noCache := r.URL.Query().Get("noCache")
	if noCache != "0" && noCache != "1" && strings.TrimSpace(noCache) != "" {
		WriteText(http.StatusBadRequest, "parameter noCache must be 1 or 0", w)
		return
	}
	request := &pb.GetServiceRequest{
		ServiceId: r.URL.Query().Get(":serviceId"),
		NoCache:   noCache == "1",
	}
	resp, err := core.ServiceAPI.GetOne(r.Context(), request)
	if err != nil {
		WriteText(http.StatusInternalServerError, err.Error(), w)
		return
	}
	if resp.GetResponse().Code != pb.Response_SUCCESS {
		WriteText(http.StatusBadRequest, resp.GetResponse().Message, w)
		return
	}
	resp.Response = nil
	WriteJsonObject(http.StatusOK, resp, w)
}

func (this *MicroServiceService) CreateDependenciesForMicroServices(w http.ResponseWriter, r *http.Request) {
	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		util.Logger().Error("body err", err)
		WriteText(http.StatusBadRequest, err.Error(), w)
		return
	}
	request := &pb.CreateDependenciesRequest{}
	err = json.Unmarshal(requestBody, request)
	if err != nil {
		util.Logger().Error("Invalid json", err)
		WriteText(http.StatusInternalServerError, "Unmarshal error", w)
		return
	}
	//fmt.Println("request is ", request)
	rsp, err := core.ServiceAPI.CreateDependenciesForMircServices(r.Context(), request)
	//fmt.Println("rsp is ", rsp)
	//请求错误
	if err != nil {
		util.Logger().Errorf(err, "create dependency failed for service internal reason.")
		WriteText(http.StatusInternalServerError, err.Error(), w)
		return
	}
	//服务内部错误
	if rsp.Response.Code == pb.Response_FAIL {
		util.Logger().Errorf(nil, "create dependency failed for request invalid. %s", rsp.Response.Message)
		WriteText(http.StatusBadRequest, rsp.Response.Message, w)
		return
	}
	WriteText(http.StatusOK, "add dependency success.", w)
}

func (this *MicroServiceService) GetConProDependencies(w http.ResponseWriter, r *http.Request) {
	noCache := r.URL.Query().Get("noCache")
	if noCache != "0" && noCache != "1" && strings.TrimSpace(noCache) != "" {
		WriteText(http.StatusBadRequest, "parameter noCache must be 1 or 0", w)
		return
	}
	request := &pb.GetDependenciesRequest{
		ServiceId: r.URL.Query().Get(":consumerId"),
		NoCache:   noCache == "1",
	}
	resp, err := core.ServiceAPI.GetConsumerDependencies(r.Context(), request)
	if err != nil {
		util.Logger().Error("get Dependency failed.", err)
		WriteText(http.StatusInternalServerError, "get Dependency failed", w)
		return
	}
	//服务请求错误
	if resp.Response.Code == pb.Response_FAIL {
		util.Logger().Errorf(nil, resp.Response.Message)
		WriteText(http.StatusBadRequest, resp.Response.Message, w)
		return
	}
	resp.Response = nil
	WriteJsonObject(http.StatusOK, resp, w)
}

func (this *MicroServiceService) GetProConDependencies(w http.ResponseWriter, r *http.Request) {
	noCache := r.URL.Query().Get("noCache")
	if noCache != "0" && noCache != "1" && strings.TrimSpace(noCache) != "" {
		WriteText(http.StatusBadRequest, "parameter noCache must be 1 or 0", w)
		return
	}
	request := &pb.GetDependenciesRequest{
		ServiceId: r.URL.Query().Get(":providerId"),
		NoCache:   noCache == "1",
	}
	resp, err := core.ServiceAPI.GetProviderDependencies(r.Context(), request)
	if err != nil {
		util.Logger().Error("get Dependency failed.", err)
		WriteText(http.StatusInternalServerError, "get Dependency failed", w)
		return
	}
	//服务请求错误
	if resp.Response.Code == pb.Response_FAIL {
		util.Logger().Errorf(nil, resp.Response.Message)
		WriteText(http.StatusBadRequest, resp.Response.Message, w)
		return
	}
	resp.Response = nil
	WriteJsonObject(http.StatusOK, resp, w)
}

func (this *MicroServiceService) UnregisterServices(w http.ResponseWriter, r *http.Request) {
	request_body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		util.Logger().Error("body ,err", err)
		WriteText(http.StatusBadRequest, err.Error(), w)
		return
	}

	request := &pb.DelServicesRequest{}

	err = json.Unmarshal(request_body, request)
	if err != nil {
		util.Logger().Error("unmarshal ,err ", err)
		WriteText(http.StatusBadRequest, err.Error(), w)
		return
	}

	resp, err := core.ServiceAPI.DeleteServices(r.Context(), request)

	if resp.Response.Code == pb.Response_SUCCESS {
		WriteText(http.StatusOK, "", w)
		return
	}
	if resp.Services == nil || len(resp.Services) == 0 {
		WriteText(http.StatusBadRequest, resp.Response.Message, w)
		return
	}
	resp.Response = nil
	objJson, err := json.Marshal(resp)
	if err != nil {
		WriteText(http.StatusInternalServerError, err.Error(), w)
		return
	}
	WriteJson(http.StatusBadRequest, objJson, w)
	return
}
