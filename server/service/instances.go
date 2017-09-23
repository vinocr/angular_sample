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
	"errors"
	"fmt"
	apt "github.com/ServiceComb/service-center/server/core"
	pb "github.com/ServiceComb/service-center/server/core/proto"
	"github.com/ServiceComb/service-center/server/core/registry"
	"github.com/ServiceComb/service-center/server/core/registry/store"
	"github.com/ServiceComb/service-center/server/infra/quota"
	"github.com/ServiceComb/service-center/server/plugins/dynamic"
	nf "github.com/ServiceComb/service-center/server/service/notification"
	serviceUtil "github.com/ServiceComb/service-center/server/service/util"
	"github.com/ServiceComb/service-center/util"
	errorsEx "github.com/ServiceComb/service-center/util/errors"
	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
	"math"
	"strconv"
	"time"
)

type InstanceController struct {
}

func (s *InstanceController) Register(ctx context.Context, in *pb.RegisterInstanceRequest) (*pb.RegisterInstanceResponse, error) {
	if in == nil || in.Instance == nil {
		util.Logger().Errorf(nil, "register instance failed: invalid params.")
		return &pb.RegisterInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Request format invalid."),
		}, nil
	}
	instance := in.GetInstance()
	if len(instance.Environment) == 0 {
		instance.Environment = apt.REGISTRY_DEFAULT_INSTANCE_ENV
	}
	remoteIP := util.GetIPFromContext(ctx)
	instanceFlag := util.StringJoin([]string{instance.ServiceId, instance.HostName}, "/")
	err := apt.Validate(instance)
	if err != nil {
		util.Logger().Errorf(err, "register instance failed, service %s, operator %s: invalid instance parameters.",
			instanceFlag, remoteIP)
		return &pb.RegisterInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
		}, nil
	}
	//先以tenant/project的方式组装
	tenant := util.ParseTenantProject(ctx)

	// service id存在性校验
	if !serviceUtil.ServiceExist(ctx, tenant, instance.ServiceId) {
		util.Logger().Errorf(nil, "register instance failed, service %s, operator %s: service not exist.", instanceFlag, remoteIP)
		return &pb.RegisterInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service does not exist."),
		}, nil
	}

	instanceId := instance.InstanceId
	//允许自定义id
	//如果没填写 并且endpoints沒重復，則产生新的全局instance id
	oldInstanceId := ""

	if instanceId == "" {
		util.Logger().Infof("start register a new instance: service %s", instanceFlag)
		if len(instance.Endpoints) != 0 {
			oldInstanceId, err = serviceUtil.CheckEndPoints(ctx, in)
			if err != nil {
				util.Logger().Errorf(err, "register instance failed, service %s, operator %s: check endpoints failed.", instanceFlag, remoteIP)
				return &pb.RegisterInstanceResponse{
					Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
				}, nil
			}
			if oldInstanceId != "" {
				instanceId = oldInstanceId
				instance.InstanceId = instanceId
			}
		}
	}

	if len(oldInstanceId) == 0 {
		ok, err := quota.QuotaPlugins[quota.QuataType]().Apply4Quotas(ctx, quota.MicroServiceInstanceQuotaType, tenant, in.Instance.ServiceId, 1)
		if err != nil {
			util.Logger().Errorf(err, "register instance failed, service %s, operator %s: check apply quota failed.", instanceFlag, remoteIP)
			return &pb.RegisterInstanceResponse{
				Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
			}, err
		}
		if !ok {
			util.Logger().Errorf(nil, "register instance failed, service %s, operator %s: no quota apply.", instanceFlag, remoteIP)
			return &pb.RegisterInstanceResponse{
				Response: pb.CreateResponse(pb.Response_FAIL, "No quota to create instance."),
			}, nil
		}
	}

	if len(instanceId) == 0 {
		instanceId = dynamic.GetInstanceId()
		instance.InstanceId = instanceId
	}

	instance.Timestamp = strconv.FormatInt(time.Now().Unix(), 10)
	instance.ModTimestamp = instance.Timestamp
	util.Logger().Debug(fmt.Sprintf("instance ID [%s]", instanceId))

	// 这里应该根据租约计时
	renewalInterval := apt.REGISTRY_DEFAULT_LEASE_RENEWALINTERVAL
	retryTimes := apt.REGISTRY_DEFAULT_LEASE_RETRYTIMES
	if instance.GetHealthCheck() == nil {
		instance.HealthCheck = &pb.HealthCheck{
			Mode:     pb.CHECK_BY_HEARTBEAT,
			Interval: renewalInterval,
			Times:    retryTimes,
		}
	} else {
		// Health check对象仅用于呈现服务健康检查逻辑，如果CHECK_BY_PLATFORM类型，表明由sidecar代发心跳，实例120s超时
		switch instance.HealthCheck.Mode {
		case pb.CHECK_BY_HEARTBEAT:
			if instance.HealthCheck.Interval <= 0 || instance.HealthCheck.Interval >= math.MaxInt32 ||
				instance.HealthCheck.Times <= 0 || instance.HealthCheck.Times >= math.MaxInt32 ||
				instance.HealthCheck.Interval*(instance.HealthCheck.Times+1) >= math.MaxInt32 {
				util.Logger().Errorf(err, "register instance %s(%s) failed for invalid health check settings.", instance.ServiceId, instance.HostName)
				return &pb.RegisterInstanceResponse{
					Response: pb.CreateResponse(pb.Response_FAIL, "Invalid health check settings."),
				}, nil
			}
			renewalInterval = instance.HealthCheck.Interval
			retryTimes = instance.HealthCheck.Times
		case pb.CHECK_BY_PLATFORM:
			// 默认120s
		}
	}
	ttl := int64(renewalInterval * (retryTimes + 1))

	data, err := json.Marshal(instance)
	if err != nil {
		util.Logger().Errorf(err, "register instance failed, service %s, instanceId %s, operator %s: json marshal data failed.",
			instanceFlag, instanceId, remoteIP)
		return &pb.RegisterInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Instance file marshal error."),
		}, err
	}

	leaseID, err := grantOrRenewLease(ctx, tenant, instance.ServiceId, instanceId, ttl)
	if err != nil {
		return &pb.RegisterInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Lease grant or renew failed."),
		}, err
	}

	index := apt.GenerateInstanceIndexKey(tenant, instanceId)
	key := apt.GenerateInstanceKey(tenant, instance.ServiceId, instanceId)
	hbKey := apt.GenerateInstanceLeaseKey(tenant, instance.ServiceId, instanceId)

	util.Logger().Debugf("start register service instance: %s %v, lease: %s %ds", key, instance, hbKey, ttl)

	opts := []registry.PluginOp{
		registry.OpPut(registry.WithStrKey(key), registry.WithValue(data),
			registry.WithLease(leaseID), registry.WithIgnoreLease()),
		registry.OpPut(registry.WithStrKey(index), registry.WithStrValue(instance.ServiceId),
			registry.WithLease(leaseID), registry.WithIgnoreLease()),
	}
	if leaseID != 0 {
		opts = append(opts,
			registry.OpPut(registry.WithStrKey(hbKey), registry.WithStrValue(fmt.Sprintf("%d", leaseID)),
				registry.WithLease(leaseID), registry.WithIgnoreLease()))
	}

	// Set key file
	_, err = registry.GetRegisterCenter().Txn(ctx, opts)
	if err != nil {
		util.Logger().Errorf(err, "register instance failed, service %s, instanceId %s, operator %s: commit data into etcd failed.",
			instanceFlag, instanceId, remoteIP)
		return &pb.RegisterInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Commit operations failed."),
		}, err
	}

	//新租户，则进行监听
	newDomain := util.ParseTenant(ctx)
	ok, err := serviceUtil.DomainExist(ctx, newDomain)
	if err != nil {
		util.Logger().Errorf(err, "register instance failed, service %s, instanceId %s, operator %s: find domain failed.",
			instanceFlag, instanceId, remoteIP)
		return &pb.RegisterInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Find domain failed."),
		}, err
	}

	if !ok {
		err = serviceUtil.NewDomain(ctx, util.ParseTenant(ctx))
		if err != nil {
			util.Logger().Errorf(err, "register instance failed, service %s, instanceId %s, operator %s: new tenant failed.",
				instanceFlag, instanceId, remoteIP)
			return &pb.RegisterInstanceResponse{
				Response: pb.CreateResponse(pb.Response_FAIL, "Commit operations failed."),
			}, err
		}
	}

	util.Logger().Infof("register instance successful service %s, instanceId %s, operator %s.", instanceFlag, instanceId, remoteIP)
	return &pb.RegisterInstanceResponse{
		Response:   pb.CreateResponse(pb.Response_SUCCESS, "Register service instance successfully."),
		InstanceId: instanceId,
	}, nil
}

func (s *InstanceController) Unregister(ctx context.Context, in *pb.UnregisterInstanceRequest) (*pb.UnregisterInstanceResponse, error) {
	if in == nil || len(in.ServiceId) == 0 || len(in.InstanceId) == 0 {
		util.Logger().Errorf(nil, "unregister instance failed: invalid params.")
		return &pb.UnregisterInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Request format invalid."),
		}, nil
	}

	tenant := util.ParseTenantProject(ctx)
	serviceId := in.ServiceId
	instanceId := in.InstanceId

	instanceFlag := util.StringJoin([]string{serviceId, instanceId}, "/")
	remoteIP := util.GetIPFromContext(ctx)
	isExist, err := serviceUtil.InstanceExist(ctx, tenant, serviceId, instanceId)
	if err != nil {
		util.Logger().Errorf(err, "unregister instance failed, instance %s, operator %s: query instance failed.", instanceFlag, remoteIP)
		return &pb.UnregisterInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Query service instance failed."),
		}, err
	}
	if !isExist {
		util.Logger().Errorf(nil, "unregister instance failed, instance %s, operator %s: instance not exist.", instanceFlag, remoteIP)
		return &pb.UnregisterInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service instance does not exist."),
		}, nil
	}

	err, isInnerErr := revokeInstance(ctx, tenant, serviceId, instanceId)
	if err != nil {
		util.Logger().Errorf(nil, "unregister instance failed, instance %s, operator %s: revoke instance failed.", instanceFlag, remoteIP)
		if isInnerErr {
			return &pb.UnregisterInstanceResponse{
				Response: pb.CreateResponse(pb.Response_FAIL, "Revoke instance failed."),
			}, err
		}
		return &pb.UnregisterInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Revoke instance failed."),
		}, nil
	}

	util.Logger().Infof("unregister instance successful isntance %s, operator %s.", instanceFlag, remoteIP)
	return &pb.UnregisterInstanceResponse{
		Response: pb.CreateResponse(pb.Response_SUCCESS, "Unregister service instance successfully."),
	}, nil
}

func revokeInstance(ctx context.Context, tenant string, serviceId string, instanceId string) (error, bool) {
	leaseID, err := serviceUtil.GetLeaseId(ctx, tenant, serviceId, instanceId)
	if err != nil {
		return err, true
	}
	if leaseID == -1 {
		return errors.New("instance's leaseId not exist."), false
	}

	err = registry.GetRegisterCenter().LeaseRevoke(ctx, leaseID)
	if err != nil {
		return err, true
	}
	return nil, false
}

func (s *InstanceController) Heartbeat(ctx context.Context, in *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	if in == nil || len(in.ServiceId) == 0 || len(in.InstanceId) == 0 {
		return &pb.HeartbeatResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Request format invalid."),
		}, nil
	}
	remoteIP := util.GetIPFromContext(ctx)
	tenant := util.ParseTenantProject(ctx)
	instanceFlag := util.StringJoin([]string{in.ServiceId, in.InstanceId}, "/")

	_, ttl, err, isInnerErr := serviceUtil.HeartbeatUtil(ctx, tenant, in.ServiceId, in.InstanceId)
	if err != nil {
		util.Logger().Errorf(err, "heartbeat failed, instance %s, internal error '%v'. operator: %s",
			instanceFlag, isInnerErr, remoteIP)
		if isInnerErr {
			return &pb.HeartbeatResponse{
				Response: pb.CreateResponse(pb.Response_FAIL, "Service instance does not exist."),
			}, err
		}
		return &pb.HeartbeatResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service instance does not exist."),
		}, nil
	}
	util.Logger().Infof("heartbeat successful: %s renew ttl to %d. operator: %s", instanceFlag, ttl, remoteIP)
	return &pb.HeartbeatResponse{
		Response: pb.CreateResponse(pb.Response_SUCCESS, "Update service instance heartbeat successfully."),
	}, nil
}

func grantOrRenewLease(ctx context.Context, tenant string, serviceId string, instanceId string, ttl int64) (leaseID int64, err error) {
	remoteIP := util.GetIPFromContext(ctx)
	instanceFlag := util.StringJoin([]string{serviceId, instanceId}, "/")

	var (
		oldTTL int64
		inner  bool
	)

	leaseID, oldTTL, err, inner = serviceUtil.HeartbeatUtil(ctx, tenant, serviceId, instanceId)
	if inner {
		util.Logger().Errorf(err, "grant or renew lease failed, instance %s, operator: %s",
			instanceFlag, remoteIP)
		return
	}

	if leaseID < 0 || (oldTTL > 0 && oldTTL != ttl) {
		leaseID, err = registry.GetRegisterCenter().LeaseGrant(ctx, ttl)
		if err != nil {
			util.Logger().Errorf(err, "grant or renew lease failed, instance %s, operator: %s: lease grant failed.",
				instanceFlag, remoteIP)
			return
		}
		util.Logger().Infof("lease grant %d->%d successfully, instance %s, operator: %s.",
			oldTTL, ttl, instanceFlag, remoteIP)
		return
	}
	return
}

func (s *InstanceController) HeartbeatSet(ctx context.Context, in *pb.HeartbeatSetRequest) (*pb.HeartbeatSetResponse, error) {
	if in == nil || len(in.Instances) == 0 {
		util.Logger().Errorf(nil, "heartbeats failed, invalid request. Body not contain Instances or is empty.")
		return &pb.HeartbeatSetResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Request format invalid."),
		}, nil
	}
	tenant := util.ParseTenantProject(ctx)
	instanceHbRstArr := []*pb.InstanceHbRst{}
	existFlag := map[string]bool{}
	instancesHbRst := make(chan *pb.InstanceHbRst, len(in.Instances))
	noMultiCounter := 0
	for _, heartbeatElement := range in.Instances {
		if _, ok := existFlag[heartbeatElement.ServiceId+heartbeatElement.InstanceId]; ok {
			util.Logger().Warnf(nil, "heartbeatset %s/%s multiple", heartbeatElement.ServiceId, heartbeatElement.InstanceId)
			continue
		} else {
			existFlag[heartbeatElement.ServiceId+heartbeatElement.InstanceId] = true
			noMultiCounter++
		}
		go func(element *pb.HeartbeatSetElement) {
			hbRst := &pb.InstanceHbRst{
				ServiceId:  element.ServiceId,
				InstanceId: element.InstanceId,
				ErrMessage: "",
			}
			_, _, err, _ := serviceUtil.HeartbeatUtil(ctx, tenant, element.ServiceId, element.InstanceId)
			if err != nil {
				hbRst.ErrMessage = err.Error()
				util.Logger().Errorf(err, "heartbeatset failed, %s/%s", element.ServiceId, element.InstanceId)
			}
			instancesHbRst <- hbRst
		}(heartbeatElement)
	}
	count := 0
	successFlag := false
	failFlag := false
	for heartbeat := range instancesHbRst {
		count++
		if len(heartbeat.ErrMessage) != 0 {
			failFlag = true
		} else {
			successFlag = true
		}
		instanceHbRstArr = append(instanceHbRstArr, heartbeat)
		if count == noMultiCounter {
			close(instancesHbRst)
		}
	}
	if !failFlag && successFlag {
		util.Logger().Infof("heartbeatset success")
		return &pb.HeartbeatSetResponse{
			Response:  pb.CreateResponse(pb.Response_SUCCESS, "Heartbeatset successfully."),
			Instances: instanceHbRstArr,
		}, nil
	} else {
		util.Logger().Errorf(nil, "heartbeatset failed, %v", in.Instances)
		return &pb.HeartbeatSetResponse{
			Response:  pb.CreateResponse(pb.Response_FAIL, "Heartbeatset failed."),
			Instances: instanceHbRstArr,
		}, nil
	}
}

func (s *InstanceController) GetOneInstance(ctx context.Context, in *pb.GetOneInstanceRequest) (*pb.GetOneInstanceResponse, error) {
	err, isInternalErr := s.getInstancePreCheck(ctx, in)
	if err != nil {
		if isInternalErr {
			util.Logger().Errorf(err, "get instance failed: pre check failed.")
			return &pb.GetOneInstanceResponse{
				Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
			}, err
		}
		util.Logger().Errorf(err, "get instance failed: pre check failed.")
		return &pb.GetOneInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
		}, nil
	}
	conPro := util.StringJoin([]string{in.ConsumerServiceId, in.ProviderServiceId, in.ProviderInstanceId}, "/")

	tenant := util.ParseTenantProject(ctx)

	serviceId := in.ProviderServiceId
	instanceId := in.ProviderInstanceId
	instance, err := serviceUtil.GetInstance(ctx, tenant, serviceId, instanceId,
		serviceUtil.QueryOptions(serviceUtil.WithNoCache(in.NoCache))...)
	if err != nil {
		util.Logger().Errorf(err, "get instance failed, %s(consumer/provider): get instance failed.", conPro)
		return &pb.GetOneInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Update service instance properties failed."),
		}, err
	}
	if instance == nil {
		util.Logger().Errorf(nil, "get instance failed, %s(consumer/provider): instance not exist.", conPro)
		return &pb.GetOneInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service instance does not exist."),
		}, nil
	}

	if len(in.Env) != 0 && in.Env != instance.Environment {
		util.Logger().Errorf(nil, "get instance failed, %s(consumer/provider): environment not match, can't access.", conPro)
		return &pb.GetOneInstanceResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Environment mismatch, can't access this instance."),
		}, nil
	}

	return &pb.GetOneInstanceResponse{
		Response: pb.CreateResponse(pb.Response_SUCCESS, "Get instance successfully."),
		Instance: instance,
	}, nil
}

func (s *InstanceController) getInstancePreCheck(ctx context.Context, in interface{}) (err error, IsInnerErr bool) {
	err = apt.Validate(in)
	if err != nil {
		return err, false
	}
	var providerServiceId, consumerServiceId string
	var tags []string
	var opts []registry.PluginOpOption
	tenant := util.ParseTenantProject(ctx)

	switch in.(type) {
	case *pb.GetOneInstanceRequest:
		providerServiceId = in.(*pb.GetOneInstanceRequest).ProviderServiceId
		consumerServiceId = in.(*pb.GetOneInstanceRequest).ConsumerServiceId
		tags = in.(*pb.GetOneInstanceRequest).Tags
		opts = serviceUtil.QueryOptions(serviceUtil.WithNoCache(in.(*pb.GetOneInstanceRequest).NoCache))
	case *pb.GetInstancesRequest:
		providerServiceId = in.(*pb.GetInstancesRequest).ProviderServiceId
		consumerServiceId = in.(*pb.GetInstancesRequest).ConsumerServiceId
		tags = in.(*pb.GetInstancesRequest).Tags
		opts = serviceUtil.QueryOptions(serviceUtil.WithNoCache(in.(*pb.GetInstancesRequest).NoCache))
	}

	if !serviceUtil.ServiceExist(ctx, tenant, providerServiceId, opts...) {
		return fmt.Errorf("Service does not exist. Service id is %s", providerServiceId), false
	}

	// Tag过滤
	if len(tags) > 0 {
		tagsFromETCD, err := serviceUtil.GetTagsUtils(ctx, tenant, providerServiceId, opts...)
		if err != nil {
			return err, true
		}
		for _, tag := range tags {
			if _, ok := tagsFromETCD[tag]; !ok {
				return fmt.Errorf("Provider's tag not contain %s", tag), false
			}
		}
	}
	// 黑白名单
	// 跨应用调用
	err = Accessible(ctx, tenant, consumerServiceId, providerServiceId, opts...)
	switch err.(type) {
	case errorsEx.InternalError:
		return err, true
	default:
		return err, false
	}
}

func (s *InstanceController) GetInstances(ctx context.Context, in *pb.GetInstancesRequest) (*pb.GetInstancesResponse, error) {
	err, isInternalErr := s.getInstancePreCheck(ctx, in)
	if err != nil {
		if isInternalErr {
			util.Logger().Errorf(err, "get instances failed: pre check failed.")
			return &pb.GetInstancesResponse{
				Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
			}, err
		}
		util.Logger().Errorf(err, "get instances failed: pre check failed.")
		return &pb.GetInstancesResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
		}, nil
	}
	conPro := util.StringJoin([]string{in.ConsumerServiceId, in.ProviderServiceId}, "/")

	tenant := util.ParseTenantProject(ctx)

	instances := []*pb.MicroServiceInstance{}
	instances, err = serviceUtil.GetAllInstancesOfOneService(ctx, tenant, in.ProviderServiceId, in.Env,
		serviceUtil.QueryOptions(serviceUtil.WithNoCache(in.NoCache))...)
	if err != nil {
		util.Logger().Errorf(err, "get instances failed, %s(consumer/provider): get instances from etcd failed.", conPro)
		return &pb.GetInstancesResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
		}, err
	}
	return &pb.GetInstancesResponse{
		Response:  pb.CreateResponse(pb.Response_SUCCESS, "Query service instances successfully."),
		Instances: instances,
	}, nil
}

func (s *InstanceController) Find(ctx context.Context, in *pb.FindInstancesRequest) (*pb.FindInstancesResponse, error) {
	err := apt.Validate(in)
	if err != nil {
		util.Logger().Errorf(err, "find instance failed: invalid parameters.")
		return &pb.FindInstancesResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
		}, nil
	}

	tenant := util.ParseTenantProject(ctx)

	opts := serviceUtil.QueryOptions(serviceUtil.WithNoCache(in.NoCache))

	findFlag := util.StringJoin([]string{in.ConsumerServiceId, in.AppId, in.ServiceName, in.VersionRule}, "/")
	service, err := serviceUtil.GetService(ctx, tenant, in.ConsumerServiceId, opts...)
	if err != nil {
		util.Logger().Errorf(err, "find instance failed, %s: get consumer failed.", findFlag)
		return &pb.FindInstancesResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
		}, nil
	}
	if service == nil {
		util.Logger().Errorf(nil, "find instance failed, %s: consumer not exist.", findFlag)
		return &pb.FindInstancesResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Consumer does not exist."),
		}, nil
	}

	// 版本规则
	ids, err := serviceUtil.FindServiceIds(ctx, in.VersionRule, &pb.MicroServiceKey{
		Tenant:      tenant,
		AppId:       in.AppId,
		ServiceName: in.ServiceName,
		Alias:       in.ServiceName,
	}, opts...)
	if err != nil {
		util.Logger().Errorf(err, "find instance failed, %s: get providers failed.", findFlag)
		return &pb.FindInstancesResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Get serviceId failed."),
		}, err
	}
	if len(ids) == 0 {
		util.Logger().Errorf(nil, "find instance failed, %s: no provider matched.", findFlag)
		return &pb.FindInstancesResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service does not exist."),
		}, nil
	}

	instances := []*pb.MicroServiceInstance{}
	for _, serviceId := range ids {
		resp, err := s.GetInstances(ctx, &pb.GetInstancesRequest{
			ConsumerServiceId: in.ConsumerServiceId,
			ProviderServiceId: serviceId,
			Tags:              in.Tags,
			Env:               in.Env,
			NoCache:           in.NoCache,
		})
		if err != nil {
			util.Logger().Errorf(err, "find instance failed, %s: get service %s 's instance failed.", findFlag, serviceId)
			return &pb.FindInstancesResponse{
				Response: resp.GetResponse(),
			}, err
		}
		if len(resp.GetInstances()) > 0 {
			instances = append(instances, resp.GetInstances()...)
		}
	}
	consumer := pb.ToMicroServiceKey(tenant, service)
	//维护version的规则
	providerService, _ := serviceUtil.GetService(ctx, tenant, ids[0], opts...)
	if providerService == nil {
		util.Logger().Errorf(nil, "find instance failed, %s: no provider matched.", findFlag)
		return &pb.FindInstancesResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service does not exist."),
		}, nil
	}
	provider := &pb.MicroServiceKey{
		Tenant:      tenant,
		AppId:       in.AppId,
		ServiceName: providerService.ServiceName,
		Version:     in.VersionRule,
	}

	err = serviceUtil.AddServiceVersionRule(ctx, tenant, provider, consumer)

	if err != nil {
		util.Logger().Errorf(err, "find instance failed, %s: add service version rule failed.", findFlag)
		return &pb.FindInstancesResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
		}, nil
	}
	util.Logger().Infof("find instance: add dependency susscess, %s", findFlag)

	return &pb.FindInstancesResponse{
		Response:  pb.CreateResponse(pb.Response_SUCCESS, "Query service instances successfully."),
		Instances: instances,
	}, nil
}

func (s *InstanceController) UpdateStatus(ctx context.Context, in *pb.UpdateInstanceStatusRequest) (*pb.UpdateInstanceStatusResponse, error) {
	if in == nil || len(in.ServiceId) == 0 || len(in.InstanceId) == 0 {
		util.Logger().Errorf(nil, "update instance status failed: invalid params.")
		return &pb.UpdateInstanceStatusResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Request format invalid."),
		}, nil
	}
	tenant := util.ParseTenantProject(ctx)
	updateStatusFlag := util.StringJoin([]string{in.ServiceId, in.InstanceId, in.Status}, "/")
	if err := apt.Validate(in); err != nil {
		util.Logger().Errorf(nil, "update instance status failed, %s.", updateStatusFlag)
		return &pb.UpdateInstanceStatusResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, err.Error()),
		}, nil
	}

	var err error

	var instance *pb.MicroServiceInstance
	instance, err = serviceUtil.GetInstance(ctx, tenant, in.ServiceId, in.InstanceId)
	if err != nil {
		util.Logger().Errorf(err, "update instance status failed, %s: get instance from etcd failed.", updateStatusFlag)
		return &pb.UpdateInstanceStatusResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Update service instance properties failed."),
		}, err
	}
	if instance == nil {
		util.Logger().Errorf(nil, "update instance status failed, %s: instance not exist.", updateStatusFlag)
		return &pb.UpdateInstanceStatusResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service instance does not exist."),
		}, nil
	}

	instance.Status = in.Status

	err, isInnerErr := updateInstance(ctx, tenant, instance)
	if err != nil {
		util.Logger().Errorf(err, "update instance status failed, %s: update instance lease failed.", updateStatusFlag)
		if isInnerErr {
			return &pb.UpdateInstanceStatusResponse{
				Response: pb.CreateResponse(pb.Response_FAIL, "Update instance status failed."),
			}, err
		}
		return &pb.UpdateInstanceStatusResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Update instance status failed."),
		}, nil
	}

	util.Logger().Infof("update instance status successful: %s.", updateStatusFlag)
	return &pb.UpdateInstanceStatusResponse{
		Response: pb.CreateResponse(pb.Response_SUCCESS, "Update service instance information successfully."),
	}, nil
}

func (s *InstanceController) UpdateInstanceProperties(ctx context.Context, in *pb.UpdateInstancePropsRequest) (*pb.UpdateInstancePropsResponse, error) {
	if in == nil || len(in.ServiceId) == 0 || len(in.InstanceId) == 0 || in.Properties == nil {
		util.Logger().Errorf(nil, "update instance properties failed: invalid params.")
		return &pb.UpdateInstancePropsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Request format invalid."),
		}, nil
	}

	var err error
	tenant := util.ParseTenantProject(ctx)
	instanceFlag := util.StringJoin([]string{in.ServiceId, in.InstanceId}, "/")

	var instance *pb.MicroServiceInstance

	instance, err = serviceUtil.GetInstance(ctx, tenant, in.ServiceId, in.InstanceId)
	if err != nil {
		util.Logger().Errorf(err, "update instance properties failed, %s: get instance from etcd failed.", instanceFlag)
		return &pb.UpdateInstancePropsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Update service instance properties failed."),
		}, err
	}
	if instance == nil {
		util.Logger().Errorf(nil, "update instance properties failed, %s: instance not exist.", instanceFlag)
		return &pb.UpdateInstancePropsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service instance does not exist."),
		}, nil
	}

	instance.Properties = map[string]string{}
	for property := range in.Properties {
		instance.Properties[property] = in.Properties[property]
	}

	err, isInnerErr := updateInstance(ctx, tenant, instance)
	if err != nil {
		util.Logger().Errorf(err, "update instance properties failed, %s: update instance lease failed.", instanceFlag)
		if isInnerErr {
			return &pb.UpdateInstancePropsResponse{
				Response: pb.CreateResponse(pb.Response_FAIL, "Update instance lease failed."),
			}, err
		}
		return &pb.UpdateInstancePropsResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Update instance lease failed."),
		}, nil
	}

	util.Logger().Infof("update instance properties successful: %s.", instanceFlag)
	return &pb.UpdateInstancePropsResponse{
		Response: pb.CreateResponse(pb.Response_SUCCESS, "Update service instance information successfully."),
	}, nil
}

func updateInstance(ctx context.Context, tenant string, instance *pb.MicroServiceInstance) (err error, isInnerErr bool) {
	leaseID, err := serviceUtil.GetLeaseId(ctx, tenant, instance.ServiceId, instance.InstanceId)
	if err != nil {
		return err, true
	}
	if leaseID == -1 {
		return errors.New("Instance's leaseId not exist."), false
	}

	instance.ModTimestamp = strconv.FormatInt(time.Now().Unix(), 10)
	data, err := json.Marshal(instance)
	if err != nil {
		return err, true
	}

	key := apt.GenerateInstanceKey(tenant, instance.ServiceId, instance.InstanceId)
	_, err = registry.GetRegisterCenter().Do(ctx,
		registry.PUT,
		registry.WithStrKey(key),
		registry.WithValue(data),
		registry.WithLease(leaseID))
	if err != nil {
		return err, true
	}

	_, err = store.Store().KeepAlive(ctx,
		registry.WithStrKey(apt.GenerateInstanceLeaseKey(tenant, instance.ServiceId, instance.InstanceId)),
		registry.WithLease(leaseID))
	if err != nil {
		return err, false
	}
	return nil, false
}

func (s *InstanceController) WatchPreOpera(ctx context.Context, in *pb.WatchInstanceRequest) error {
	if in == nil || len(in.SelfServiceId) == 0 {
		return errors.New("Request format invalid.")
	}
	tenant := util.ParseTenantProject(ctx)
	if !serviceUtil.ServiceExist(ctx, tenant, in.SelfServiceId) {
		return errors.New("Service does not exist.")
	}
	return nil
}

func (s *InstanceController) Watch(in *pb.WatchInstanceRequest, stream pb.ServiceInstanceCtrl_WatchServer) error {
	var err error
	if err = s.WatchPreOpera(stream.Context(), in); err != nil {
		util.Logger().Errorf(err, "establish watch failed: invalid params.")
		return err
	}
	tenant := util.ParseTenantProject(stream.Context())
	watcher := nf.NewInstanceWatcher(in.SelfServiceId, apt.GetInstanceRootKey(tenant)+"/")
	err = nf.GetNotifyService().AddSubscriber(watcher)
	util.Logger().Infof("start watch instance status, watcher %s %s", watcher.Subject(), watcher.Id())
	return nf.HandleWatchJob(watcher, stream, nf.GetNotifyService().Config.NotifyTimeout)
}

func (s *InstanceController) WebSocketWatch(ctx context.Context, in *pb.WatchInstanceRequest, conn *websocket.Conn) {
	util.Logger().Infof("New a web socket watch with %s", in.SelfServiceId)
	if err := s.WatchPreOpera(ctx, in); err != nil {
		nf.EstablishWebSocketError(conn, err)
		return
	}
	nf.DoWebSocketWatch(ctx, in.SelfServiceId, conn)
}

func (s *InstanceController) WebSocketListAndWatch(ctx context.Context, in *pb.WatchInstanceRequest, conn *websocket.Conn) {
	util.Logger().Infof("New a web socket list and watch with %s", in.SelfServiceId)
	if err := s.WatchPreOpera(ctx, in); err != nil {
		nf.EstablishWebSocketError(conn, err)
		return
	}
	nf.DoWebSocketListAndWatch(ctx, in.SelfServiceId, func() ([]*pb.WatchInstanceResponse, int64) {
		return serviceUtil.QueryAllProvidersIntances(ctx, in.SelfServiceId)
	}, conn)
}

func (s *InstanceController) ClusterHealth(ctx context.Context) (*pb.GetInstancesResponse, error) {
	tenant := util.StringJoin([]string{apt.REGISTRY_TENANT, apt.REGISTRY_PROJECT}, "/")
	serviceId, err := serviceUtil.GetServiceId(ctx, &pb.MicroServiceKey{
		AppId:       apt.Service.AppId,
		ServiceName: apt.Service.ServiceName,
		Version:     apt.Service.Version,
		Tenant:      tenant,
	})

	if err != nil {
		util.Logger().Errorf(err, "health check failed: get service center serviceId failed.")
		return &pb.GetInstancesResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service center serviceId failed."),
		}, err
	}
	if len(serviceId) == 0 {
		util.Logger().Errorf(nil, "health check failed: get service center serviceId not exist.")
		return &pb.GetInstancesResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service center serviceId not exist."),
		}, nil
	}
	instances := []*pb.MicroServiceInstance{}
	instances, err = serviceUtil.GetAllInstancesOfOneService(ctx, tenant, serviceId, "")
	if err != nil {
		util.Logger().Errorf(err, "health check failed: get service center instances failed.")
		return &pb.GetInstancesResponse{
			Response: pb.CreateResponse(pb.Response_FAIL, "Service center instances failed."),
		}, err
	}
	return &pb.GetInstancesResponse{
		Response:  pb.CreateResponse(pb.Response_SUCCESS, "Health check successfully."),
		Instances: instances,
	}, nil
}
