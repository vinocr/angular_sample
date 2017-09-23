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
package service

import (
	"encoding/json"
	apt "github.com/ServiceComb/service-center/server/core"
	pb "github.com/ServiceComb/service-center/server/core/proto"
	"github.com/ServiceComb/service-center/server/core/registry"
	"github.com/ServiceComb/service-center/server/infra/quota"
	serviceUtil "github.com/ServiceComb/service-center/server/service/util"
	"github.com/ServiceComb/service-center/util"
	"golang.org/x/net/context"
)

func (s *ServiceController) AddTags(ctx context.Context, in *pb.AddServiceTagsRequest) (*pb.AddServiceTagsResponse, error) {
	if in == nil || len(in.ServiceId) == 0 || len(in.GetTags()) == 0 {
		util.Logger().Errorf(nil, "add service tags failed: invalid parameters.")
		return &pb.AddServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Request format invalid."),
		}, nil
	}

	err := apt.Validate(in)
	if err != nil {
		util.Logger().Errorf(err, "add service tags failed, serviceId %s, tags %v: invalid parameters.", in.ServiceId, in.Tags)
		return &pb.AddServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
		}, nil
	}

	tenant := util.ParseTenantProject(ctx)
	// service id存在性校验
	if !serviceUtil.ServiceExist(ctx, tenant, in.ServiceId) {
		util.Logger().Errorf(nil, "add service tags failed, serviceId %s, tags %v: service not exist.", in.ServiceId, in.Tags)
		return &pb.AddServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service does not exist."),
		}, nil
	}

	addTags := in.GetTags()
	ok, err := quota.QuotaPlugins[quota.QuataType]().Apply4Quotas(ctx, quota.TAGQuotaType, tenant, in.ServiceId, int16(len(addTags)))
	if err != nil {
		util.Logger().Errorf(err, "add tag info failed, check resource num failed, %s", in.ServiceId)
		return &pb.AddServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Modify schema info failed, check resource num failed."),
		}, err
	}
	if !ok {
		util.Logger().Errorf(err, "add tag info failed, reach the max size of tag, %s", in.ServiceId)
		return &pb.AddServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "reach the max size of tag."),
		}, nil
	}

	dataTags, err := serviceUtil.GetTagsUtils(ctx, tenant, in.ServiceId)
	if err != nil {
		util.Logger().Errorf(err, "add service tags failed, serviceId %s, tags %v: get existed tag failed.", in.ServiceId, in.Tags)
		return &pb.AddServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Get tags failed."),
		}, err
	}
	if len(dataTags) > 0 {
		for key, value := range addTags {
			dataTags[key] = value
		}
	} else {
		dataTags = addTags
	}

	err = serviceUtil.AddTagIntoETCD(ctx, tenant, in.ServiceId, dataTags)
	if err != nil {
		util.Logger().Errorf(err, "add service tags failed, serviceId %s, tags %v: commit tag data into etcd failed.", in.ServiceId, in.Tags)
		return &pb.AddServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Commit operations failed."),
		}, err
	}

	util.Logger().Infof("add service tags successful, serviceId %s, tags %v.", in.ServiceId, in.Tags)
	return &pb.AddServiceTagsResponse{
		Response: pb.CreateResponse(pb.Response_SUCCESS, "Add service tags successfully."),
	}, nil
}

func (s *ServiceController) UpdateTag(ctx context.Context, in *pb.UpdateServiceTagRequest) (*pb.UpdateServiceTagResponse, error) {
	if in == nil || len(in.ServiceId) == 0 || len(in.Key) == 0 || len(in.Value) == 0 {
		util.Logger().Errorf(nil, "update service tag failed: invalid parameters.")
		return &pb.UpdateServiceTagResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Request format invalid."),
		}, nil
	}
	tagFlag := util.StringJoin([]string{in.Key, in.Value}, "/")
	err := apt.Validate(in)
	if err != nil {
		util.Logger().Errorf(err, "update service tag failed, serviceId %s, tag %s: invalid params.", in.ServiceId, tagFlag)
		return &pb.UpdateServiceTagResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
		}, nil
	}

	tenant := util.ParseTenantProject(ctx)

	if !serviceUtil.ServiceExist(ctx, tenant, in.ServiceId) {
		util.Logger().Errorf(err, "update service tag failed, serviceId %s, tag %s: service not exist.", in.ServiceId, tagFlag)
		return &pb.UpdateServiceTagResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service does not exist."),
		}, nil
	}

	tags, err := serviceUtil.GetTagsUtils(ctx, tenant, in.ServiceId)
	if err != nil {
		util.Logger().Errorf(err, "update service tag failed, serviceId %s, tag %s: get tag failed.", in.ServiceId, tagFlag)
		return &pb.UpdateServiceTagResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Get tags for service failed."),
		}, err
	}
	//check tag 是否存在
	if _, ok := tags[in.Key]; !ok {
		util.Logger().Errorf(nil, "update service tag failed, serviceId %s, tag %s: tag not exist,please add first.", in.ServiceId, tagFlag)
		return &pb.UpdateServiceTagResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Update tag for service failed for update tags not exist, please add first."),
		}, err
	}
	tags[in.Key] = in.Value

	err = serviceUtil.AddTagIntoETCD(ctx, tenant, in.ServiceId, tags)

	if err != nil {
		util.Logger().Errorf(err, "update service tag failed, serviceId %s, tag %s: adding service tags failed.", in.ServiceId, tagFlag)
		return &pb.UpdateServiceTagResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Commit into etcd failed."),
		}, err
	}

	util.Logger().Infof("update tag successful, serviceId %s, tag %s.", in.ServiceId, tagFlag)
	return &pb.UpdateServiceTagResponse{
		Response: pb.CreateResponse(pb.Response_SUCCESS, "Update service tag success."),
	}, nil
}

func (s *ServiceController) DeleteTags(ctx context.Context, in *pb.DeleteServiceTagsRequest) (*pb.DeleteServiceTagsResponse, error) {
	if in == nil || len(in.ServiceId) == 0 || len(in.Keys) == 0 {
		util.Logger().Errorf(nil, "delete service tags failed: invalid parameters.")
		return &pb.DeleteServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Request format invalid."),
		}, nil
	}
	err := apt.Validate(in)
	if err != nil {
		util.Logger().Errorf(err, "delete service tags failed, serviceId %s, tags %v: invalid params.", in.ServiceId, in.Keys)
		return &pb.DeleteServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
		}, nil
	}

	tenant := util.ParseTenantProject(ctx)

	if !serviceUtil.ServiceExist(ctx, tenant, in.ServiceId) {
		util.Logger().Errorf(nil, "delete service tags failed, serviceId %s, tags %v: service not exist.", in.ServiceId, in.Keys)
		return &pb.DeleteServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service does not exist."),
		}, nil
	}

	tags, err := serviceUtil.GetTagsUtils(ctx, tenant, in.ServiceId)
	if err != nil {
		util.Logger().Errorf(err, "delete service tags failed, serviceId %s, tags %v: query service failed.", in.ServiceId, in.Keys)
		return &pb.DeleteServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Get service tags file failed."),
		}, err
	}
	for _, key := range in.Keys {
		if _, ok := tags[key]; !ok {
			util.Logger().Errorf(nil, "delete service tags failed, serviceId %s, tags %v: tag %s not exist.", in.ServiceId, in.Keys, key)
			return &pb.DeleteServiceTagsResponse{
				Response: pb.CreateResponse(pb.Response_FAIL, "Delete tags failed for this key "+key+" does not exist."),
			}, nil
		}
		delete(tags, key)
	}

	// tags 可能size == 0
	data, err := json.Marshal(tags)
	if err != nil {
		util.Logger().Errorf(err, "delete service tags failed, serviceId %s, tags %v: marshall service tag failed.", in.ServiceId, in.Keys)
		return &pb.DeleteServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Marshal service tags file failed."),
		}, err
	}

	key := apt.GenerateServiceTagKey(tenant, in.ServiceId)

	util.Logger().Debugf("start delete service tags file: %s %v", key, in.Keys)
	_, err = registry.GetRegisterCenter().Do(ctx,
		registry.PUT,
		registry.WithStrKey(key),
		registry.WithValue(data))
	if err != nil {
		util.Logger().Errorf(err, "delete service tags failed, serviceId %s, tags %v: commit tag data into etcd failed.", in.ServiceId, in.Keys)
		return &pb.DeleteServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Commit operations failed."),
		}, err
	}

	util.Logger().Infof("delete service tags successful: serviceId %s, tag %v.", in.ServiceId, in.Keys)
	return &pb.DeleteServiceTagsResponse{
		Response: pb.CreateResponse(pb.Response_SUCCESS, "Delete service tags successfully."),
	}, nil
}

func (s *ServiceController) GetTags(ctx context.Context, in *pb.GetServiceTagsRequest) (*pb.GetServiceTagsResponse, error) {
	if in == nil || len(in.ServiceId) == 0 {
		util.Logger().Errorf(nil, "get service tags failed: invalid params.")
		return &pb.GetServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Request format invalid."),
		}, nil
	}
	err := apt.Validate(in)
	if err != nil {
		util.Logger().Errorf(err, "get service tags failed, serviceId %s: invalid parameters.", in.ServiceId)
		return &pb.GetServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
		}, nil
	}

	tenant := util.ParseTenantProject(ctx)

	opts := serviceUtil.QueryOptions(serviceUtil.WithNoCache(in.NoCache))

	if !serviceUtil.ServiceExist(ctx, tenant, in.ServiceId, opts...) {
		util.Logger().Errorf(err, "get service tags failed, serviceId %s: service not exist.", in.ServiceId)
		return &pb.GetServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service does not exist."),
		}, nil
	}

	tags, err := serviceUtil.GetTagsUtils(ctx, tenant, in.ServiceId, opts...)
	if err != nil {
		util.Logger().Errorf(err, "get service tags failed, serviceId %s: get tag failed.", in.ServiceId)
		return &pb.GetServiceTagsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Get tags for service failed."),
		}, err
	}

	return &pb.GetServiceTagsResponse{
		Response: pb.CreateResponse(pb.Response_SUCCESS, "Get service tags successfully."),
		Tags:     tags,
	}, nil
}
