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
// @APIVersion 1.0.0
// @APITitle Service manager API
// @APIDescription Service manager API
// @BasePath http://127.0.0.1:8080/
// @Contact tianxiaoliang3@huawei.com
package main

// plugins
import _ "github.com/ServiceComb/service-center/server/bootstrap"
import "github.com/ServiceComb/service-center/server"

func main() {
	server.Run()
}
