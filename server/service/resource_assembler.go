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
	pb "github.com/ServiceComb/service-center/server/core/proto"
)

func AssembleResources() (pb.ServiceCtrlServer, pb.SerivceInstanceCtrlServerEx, pb.GovernServiceCtrlServerEx) {
	var serviceController ServiceController
	var instanceController InstanceController
	var governServiceAPI GovernServiceController
	return &serviceController, &instanceController, &governServiceAPI
}
