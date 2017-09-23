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
package integrationtest_test

import (
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/widuu/gojson"
	"net/http"
	"strings"

	"bytes"
	. "github.com/ServiceComb/service-center/integration"
	"io/ioutil"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

var _ = Describe("MicroService Api Test", func() {
	var serviceName = ""
	var serviceId = ""
	var serviceAppId = "integrationtestAppIdInstance"
	var serviceVersion = "0.0.2"
	var serviceInstanceID = ""
	Context("Tesing MicroService Instances API's", func() {
		BeforeEach(func() {
			schema := []string{"testSchema"}
			properties := map[string]string{"attr1": "aa"}
			serviceName = strconv.Itoa(rand.Intn(15))
			servicemap := map[string]interface{}{
				"serviceName": serviceName,
				"appId":       serviceAppId,
				"version":     serviceVersion,
				"description": "examples",
				"level":       "FRONT",
				"schemas":     schema,
				"status":      "UP",
				"properties":  properties,
			}
			bodyParams := map[string]interface{}{
				"service": servicemap,
			}
			body, _ := json.Marshal(bodyParams)
			bodyBuf := bytes.NewReader(body)
			req, _ := http.NewRequest(POST, SCURL+REGISTERMICROSERVICE, bodyBuf)
			req.Header.Set("X-tenant-name", "default")
			resp, err := scclient.Do(req)
			Expect(err).To(BeNil())
			defer resp.Body.Close()

			// Validate the service creation
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			respbody, _ := ioutil.ReadAll(resp.Body)
			serviceId = gojson.Json(string(respbody)).Get("serviceId").Tostring()
			Expect(len(serviceId)).Should(BeNumerically("==", 32))

			//Register MicroService Instance
			endpoints := []string{"cse://127.0.0.1:9984"}
			propertiesInstance := map[string]interface{}{
				"_TAGS":  "A,B",
				"attr1":  "a",
				"nodeIP": "one",
			}
			healthcheck := map[string]interface{}{
				"mode":     "push",
				"interval": 30,
				"times":    2,
			}
			instance := map[string]interface{}{
				"endpoints":   endpoints,
				"hostName":    "cse",
				"status":      "UP",
				"environment": "production",
				"properties":  propertiesInstance,
				"healthCheck": healthcheck,
			}

			bodyParams = map[string]interface{}{
				"instance": instance,
			}
			url := strings.Replace(REGISTERINSTANCE, ":serviceId", serviceId, 1)
			body, _ = json.Marshal(bodyParams)
			bodyBuf = bytes.NewReader(body)
			req, _ = http.NewRequest(POST, SCURL+url, bodyBuf)
			req.Header.Set("X-tenant-name", "default")
			resp, err = scclient.Do(req)
			Expect(err).To(BeNil())
			defer resp.Body.Close()

			// Validate the instance registration
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			respbody, _ = ioutil.ReadAll(resp.Body)
			serviceInstanceID = gojson.Json(string(respbody)).Get("instanceId").Tostring()
			Expect(len(serviceId)).Should(BeNumerically("==", 32))

		})

		AfterEach(func() {
			if serviceInstanceID != "" {
				url := strings.Replace(UNREGISTERINSTANCE, ":serviceId", serviceId, 1)
				url = strings.Replace(url, ":instanceId", serviceInstanceID, 1)
				req, _ := http.NewRequest(DELETE, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				resp, _ := scclient.Do(req)
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			}

			if serviceId != "" {
				url := strings.Replace(UNREGISTERMICROSERVICE, ":serviceId", serviceId, 1)
				req, _ := http.NewRequest(DELETE, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				resp, _ := scclient.Do(req)
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			}

		})

		By("Register MicroService Instance API", func() {
			It("Register MicroService Instance with invalid params", func() {
				instance := map[string]interface{}{
					"hostName":    "cse",
					"status":      "UP",
					"environment": "production",
				}

				bodyParams := map[string]interface{}{
					"instancse": instance,
				}
				url := strings.Replace(REGISTERINSTANCE, ":serviceId", serviceId, 1)
				body, _ := json.Marshal(bodyParams)
				bodyBuf := bytes.NewReader(body)
				req, _ := http.NewRequest(POST, SCURL+url, bodyBuf)
				req.Header.Set("X-tenant-name", "default")
				resp, err := scclient.Do(req)
				Expect(err).To(BeNil())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})

			It("Register MicroService Instance with duplicate Params", func() {
				endpoints := []string{"cse://127.0.0.1:9984"}
				propertiesInstance := map[string]interface{}{
					"_TAGS":  "A,B",
					"attr1":  "a",
					"nodeIP": "one",
				}
				healthcheck := map[string]interface{}{
					"mode":     "push",
					"interval": 30,
					"times":    2,
				}
				instance := map[string]interface{}{
					"endpoints":   endpoints,
					"hostName":    "cse",
					"status":      "UP",
					"environment": "production",
					"properties":  propertiesInstance,
					"healthCheck": healthcheck,
				}

				bodyParams := map[string]interface{}{
					"instance": instance,
				}
				url := strings.Replace(REGISTERINSTANCE, ":serviceId", serviceId, 1)
				body, _ := json.Marshal(bodyParams)
				bodyBuf := bytes.NewReader(body)
				req, _ := http.NewRequest(POST, SCURL+url, bodyBuf)
				req.Header.Set("X-tenant-name", "default")
				resp, err := scclient.Do(req)
				Expect(err).To(BeNil())
				defer resp.Body.Close()

				// Validate the instance registration
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				respbody, _ := ioutil.ReadAll(resp.Body)

				//Validate the instance id is same as the old one
				Expect(gojson.Json(string(respbody)).Get("instanceId").Tostring()).To(Equal(serviceInstanceID))
			})

			It("Register MicroService Instance with valid params", func() {
				endpoints := []string{"cse://127.0.0.1:9985"}
				propertiesInstance := map[string]interface{}{
					"_TAGS":  "A,B",
					"attr1":  "a",
					"nodeIP": "one",
				}
				healthcheck := map[string]interface{}{
					"mode":     "push",
					"interval": 30,
					"times":    2,
				}
				instance := map[string]interface{}{
					"endpoints":   endpoints,
					"hostName":    "cse",
					"status":      "UP",
					"environment": "production",
					"properties":  propertiesInstance,
					"healthCheck": healthcheck,
				}

				bodyParams := map[string]interface{}{
					"instance": instance,
				}
				url := strings.Replace(REGISTERINSTANCE, ":serviceId", serviceId, 1)
				body, _ := json.Marshal(bodyParams)
				bodyBuf := bytes.NewReader(body)
				req, _ := http.NewRequest(POST, SCURL+url, bodyBuf)
				req.Header.Set("X-tenant-name", "default")
				resp, err := scclient.Do(req)
				Expect(err).To(BeNil())
				defer resp.Body.Close()

				// Validate the instance registration
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				respbody, _ := ioutil.ReadAll(resp.Body)
				serviceInst := gojson.Json(string(respbody)).Get("instanceId").Tostring()

				//Verify the instanceID is different for two instance
				Expect(serviceInst).NotTo(Equal(serviceInstanceID))
				//Delete Instance
				url = strings.Replace(UNREGISTERINSTANCE, ":serviceId", serviceId, 1)
				url = strings.Replace(url, ":instanceId", serviceInst, 1)
				req, _ = http.NewRequest(DELETE, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				resp, _ = scclient.Do(req)
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
		})

		By("Discover MicroService Instance API", func() {
			It("Find Micro-service Info by AppID", func() {
				req, _ := http.NewRequest(GET, SCURL+FINDINSTANCE+"?appId="+serviceAppId+"&serviceName="+serviceName+"&version="+serviceVersion, nil)
				req.Header.Set("X-tenant-name", "default")
				req.Header.Set("X-ConsumerId", serviceId)
				resp, _ := scclient.Do(req)
				respbody, _ := ioutil.ReadAll(resp.Body)
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				servicesStruct := map[string][]map[string]interface{}{}
				json.Unmarshal(respbody, &servicesStruct)
				foundMicroServiceInstance := false
				for _, services := range servicesStruct["instances"] {
					if services["instanceId"] == serviceInstanceID {
						foundMicroServiceInstance = true
						break
					}
				}
				Expect(foundMicroServiceInstance).To(Equal(true))
			})

			It("Find Micro-service Info by invalid AppID", func() {
				req, _ := http.NewRequest(GET, SCURL+FINDINSTANCE+"?appId=XXXX&serviceName="+serviceName+"&version="+serviceVersion, nil)
				req.Header.Set("X-tenant-name", "default")
				req.Header.Set("X-ConsumerId", serviceId)
				resp, _ := scclient.Do(req)
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})

			It("Find Micro-Service Instance by ServiceId", func() {
				url := strings.Replace(GETINSTANCE, ":serviceId", serviceId, 1)
				req, _ := http.NewRequest(GET, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				req.Header.Set("X-ConsumerId", serviceId)
				resp, _ := scclient.Do(req)
				respbody, _ := ioutil.ReadAll(resp.Body)
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				servicesStruct := map[string][]map[string]interface{}{}
				json.Unmarshal(respbody, &servicesStruct)
				foundMicroServiceInstance := false
				for _, services := range servicesStruct["instances"] {
					if services["instanceId"] == serviceInstanceID {
						foundMicroServiceInstance = true
						break
					}
				}
				Expect(foundMicroServiceInstance).To(Equal(true))
			})

			It("Find Micro-Service Instance by Invalid ServiceId", func() {
				url := strings.Replace(GETINSTANCE, ":serviceId", "XX", 1)
				req, _ := http.NewRequest(GET, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				req.Header.Set("X-ConsumerId", serviceId)
				resp, _ := scclient.Do(req)
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})

			It("Find MicroServiceInstance with Service and IstanceID", func() {
				url := strings.Replace(GETINSTANCEBYINSTANCEID, ":serviceId", serviceId, 1)
				url = strings.Replace(url, ":instanceId", serviceInstanceID, 1)
				req, _ := http.NewRequest(GET, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				req.Header.Set("X-ConsumerId", serviceId)
				resp, _ := scclient.Do(req)
				respbody, _ := ioutil.ReadAll(resp.Body)
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				servicesStruct := map[string]map[string]interface{}{}
				json.Unmarshal(respbody, &servicesStruct)
				instance := servicesStruct["instance"]
				Expect(instance["instanceId"]).To(Equal(serviceInstanceID))
			})

			It("Find Micro-Service Instance by Invalid InstanceID", func() {
				url := strings.Replace(GETINSTANCEBYINSTANCEID, ":serviceId", serviceId, 1)
				url = strings.Replace(url, ":instanceId", "XX", 1)
				req, _ := http.NewRequest(GET, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				req.Header.Set("X-ConsumerId", serviceId)
				resp, _ := scclient.Do(req)
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})
		})
		By("Update Micro-Service Instance Information API's", func() {
			It("Update Micro-Service Instance Properties", func() {
				propertiesInstance := map[string]interface{}{
					"_TAGS":  "A,B",
					"attr1":  "NewValue",
					"nodeIP": "two",
				}
				bodyParams := map[string]interface{}{
					"properties": propertiesInstance,
				}
				url := strings.Replace(UPDATEINSTANCEMETADATA, ":serviceId", serviceId, 1)
				url = strings.Replace(url, ":instanceId", serviceInstanceID, 1)
				body, _ := json.Marshal(bodyParams)
				bodyBuf := bytes.NewReader(body)
				req, _ := http.NewRequest(UPDATE, SCURL+url, bodyBuf)
				req.Header.Set("X-tenant-name", "default")
				resp, err := scclient.Do(req)
				Expect(err).To(BeNil())
				time.Sleep(time.Second)
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				//Verify the updated properties
				url = strings.Replace(GETINSTANCE, ":serviceId", serviceId, 1)
				req, _ = http.NewRequest(GET, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				req.Header.Set("X-ConsumerId", serviceId)
				resp, _ = scclient.Do(req)
				respbody, _ := ioutil.ReadAll(resp.Body)
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				servicesStruct := map[string][]map[string]interface{}{}
				json.Unmarshal(respbody, &servicesStruct)
				time.Sleep(time.Second)
				for _, services := range servicesStruct["instances"] {
					if services["instanceId"] == serviceInstanceID {
						newproperties := services["properties"]
						Expect(newproperties).To(Equal(propertiesInstance))
						break
					}
				}

			})

			It("Update Micro-Service Instance Properties with invalid params", func() {
				propertiesInstance := map[string]interface{}{}
				bodyParams := map[string]interface{}{
					"properties": propertiesInstance,
				}
				url := strings.Replace(UPDATEINSTANCEMETADATA, ":serviceId", serviceId, 1)
				url = strings.Replace(url, ":instanceId", "WRONGINSTANCEID", 1)
				body, _ := json.Marshal(bodyParams)
				bodyBuf := bytes.NewReader(body)
				req, _ := http.NewRequest(UPDATE, SCURL+url, bodyBuf)
				req.Header.Set("X-tenant-name", "default")
				resp, err := scclient.Do(req)
				Expect(err).To(BeNil())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})

			It("Update Micro-Service Instance Status", func() {
				url := strings.Replace(UPDATEINSTANCESTATUS, ":serviceId", serviceId, 1)
				url = strings.Replace(url, ":instanceId", serviceInstanceID, 1)
				req, _ := http.NewRequest(UPDATE, SCURL+url+"?value=DOWN", nil)
				req.Header.Set("X-tenant-name", "default")
				resp, err := scclient.Do(req)
				Expect(err).To(BeNil())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				time.Sleep(time.Second)

				//Verify the Instance Status
				url = strings.Replace(GETINSTANCE, ":serviceId", serviceId, 1)
				req, _ = http.NewRequest(GET, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				req.Header.Set("X-ConsumerId", serviceId)
				resp, _ = scclient.Do(req)
				respbody, _ := ioutil.ReadAll(resp.Body)
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				servicesStruct := map[string][]map[string]interface{}{}
				json.Unmarshal(respbody, &servicesStruct)
				time.Sleep(time.Second)
				for _, services := range servicesStruct["instances"] {
					if services["instanceId"] == serviceInstanceID {
						newSTATUS := services["status"]
						Expect(newSTATUS).To(Equal("DOWN"))
						break
					}
				}
			})

			It("Update Micro-Service Instance Status with invalid Status", func() {
				url := strings.Replace(UPDATEINSTANCESTATUS, ":serviceId", serviceId, 1)
				url = strings.Replace(url, ":instanceId", serviceInstanceID, 1)
				req, _ := http.NewRequest(UPDATE, SCURL+url+"?value=INVALID", nil)
				req.Header.Set("X-tenant-name", "default")
				resp, err := scclient.Do(req)
				Expect(err).To(BeNil())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

				//Verify the Instance Status does not change the value
				url = strings.Replace(GETINSTANCE, ":serviceId", serviceId, 1)
				req, _ = http.NewRequest(GET, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				req.Header.Set("X-ConsumerId", serviceId)
				resp, _ = scclient.Do(req)
				respbody, _ := ioutil.ReadAll(resp.Body)
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				servicesStruct := map[string][]map[string]interface{}{}
				json.Unmarshal(respbody, &servicesStruct)
				for _, services := range servicesStruct["instances"] {
					if services["instanceId"] == serviceInstanceID {
						newSTATUS := services["status"]
						Expect(newSTATUS).To(Equal("UP"))
						break
					}
				}
			})
		})
		By("Micro-Service Instance heartbeat API", func() {
			It("Send HeartBeat for micro-service instance", func() {
				url := strings.Replace(INSTANCEHEARTBEAT, ":serviceId", serviceId, 1)
				url = strings.Replace(url, ":instanceId", serviceInstanceID, 1)
				req, _ := http.NewRequest(UPDATE, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				resp, err := scclient.Do(req)
				Expect(err).To(BeNil())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})

			It("Send HeartBeat for wrong micro-service instance", func() {
				url := strings.Replace(INSTANCEHEARTBEAT, ":serviceId", serviceId, 1)
				url = strings.Replace(url, ":instanceId", "XXX", 1)
				req, _ := http.NewRequest(UPDATE, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				resp, err := scclient.Do(req)
				Expect(err).To(BeNil())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})
		})

		By("Micro-Sevrice Instance Watchern Api", func() {
			It("Call the watcher API ", func() {
				//This api gives 400 bad request for the integration test
				// as integration test is not able to make ws connection
				url := strings.Replace(INSTANCEWATCHER, ":serviceId", serviceId, 1)
				req, _ := http.NewRequest(GET, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				resp, err := scclient.Do(req)
				Expect(err).To(BeNil())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})
			It("Call the listwatcher API ", func() {
				//This api gives 400 bad request for the integration test
				// as integration test is not able to make ws connection
				url := strings.Replace(INSTANCELISTWATCHER, ":serviceId", serviceId, 1)
				req, _ := http.NewRequest(GET, SCURL+url, nil)
				req.Header.Set("X-tenant-name", "default")
				resp, err := scclient.Do(req)
				Expect(err).To(BeNil())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})
		})
	})

})

func BenchmarkRegisterMicroServiceInstance(b *testing.B) {
	schema := []string{"testSchema"}
	properties := map[string]string{"attr1": "aa"}
	servicemap := map[string]interface{}{
		"serviceName": "testInstance" + strconv.Itoa(rand.Int()),
		"appId":       "testApp",
		"version":     "1.0",
		"description": "examples",
		"level":       "FRONT",
		"schemas":     schema,
		"status":      "UP",
		"properties":  properties,
	}
	bodyParams := map[string]interface{}{
		"service": servicemap,
	}
	body, _ := json.Marshal(bodyParams)
	bodyBuf := bytes.NewReader(body)
	req, _ := http.NewRequest(POST, SCURL+REGISTERMICROSERVICE, bodyBuf)
	req.Header.Set("X-tenant-name", "default")
	resp, err := scclient.Do(req)
	Expect(err).To(BeNil())
	defer resp.Body.Close()

	// Validate the service creation
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respbody, _ := ioutil.ReadAll(resp.Body)
	serviceId := gojson.Json(string(respbody)).Get("serviceId").Tostring()
	Expect(len(serviceId)).Should(BeNumerically("==", 32))

	for i := 0; i < b.N; i++ {
		//Register MicroService Instance
		endpoints := []string{"cse://127.0.0.1:9984"}
		propertiesInstance := map[string]interface{}{
			"_TAGS":  "A,B",
			"attr1":  "a",
			"nodeIP": "one",
		}
		healthcheck := map[string]interface{}{
			"mode":     "push",
			"interval": 30,
			"times":    2,
		}
		instance := map[string]interface{}{
			"endpoints":   endpoints,
			"hostName":    "cse",
			"status":      "UP",
			"environment": "production",
			"properties":  propertiesInstance,
			"healthCheck": healthcheck,
		}

		bodyParams = map[string]interface{}{
			"instance": instance,
		}
		url := strings.Replace(REGISTERINSTANCE, ":serviceId", serviceId, 1)
		body, _ = json.Marshal(bodyParams)
		bodyBuf = bytes.NewReader(body)
		req, _ = http.NewRequest(POST, SCURL+url, bodyBuf)
		req.Header.Set("X-tenant-name", "default")
		resp, err = scclient.Do(req)
		Expect(err).To(BeNil())
		defer resp.Body.Close()

		// Validate the instance registration
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		respbody, _ = ioutil.ReadAll(resp.Body)
		serviceInstanceID := gojson.Json(string(respbody)).Get("instanceId").Tostring()
		Expect(len(serviceId)).Should(BeNumerically("==", 32))

		if serviceInstanceID != "" {
			url := strings.Replace(UNREGISTERINSTANCE, ":serviceId", serviceId, 1)
			url = strings.Replace(url, ":instanceId", serviceInstanceID, 1)
			req, _ := http.NewRequest(DELETE, SCURL+url, nil)
			req.Header.Set("X-tenant-name", "default")
			resp, _ := scclient.Do(req)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		}
	}
	if serviceId != "" {
		url := strings.Replace(UNREGISTERMICROSERVICE, ":serviceId", serviceId, 1)
		req, _ := http.NewRequest(DELETE, SCURL+url, nil)
		req.Header.Set("X-tenant-name", "default")
		resp, _ := scclient.Do(req)
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	}
}
