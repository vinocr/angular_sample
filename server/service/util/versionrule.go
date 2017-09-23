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
package util

import (
	"bytes"
	"github.com/ServiceComb/service-center/util"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"sort"
	"strconv"
	"strings"
)

type VersionRule func(sorted []string, kvs map[string]string, start, end string) []string

func (vr VersionRule) Match(kvs []*mvccpb.KeyValue, ops ...string) []string {
	sorter := &serviceKeySorter{
		sortArr: make([]string, len(kvs)),
		kvs:     make(map[string]string),
		cmp:     Larger,
	}
	for i, kv := range kvs {
		key := util.BytesToStringWithNoCopy(kv.Key)
		sorter.sortArr[i] = key[strings.LastIndex(key, "/")+1:]
		sorter.kvs[sorter.sortArr[i]] = util.BytesToStringWithNoCopy(kv.Value)
	}
	sort.Sort(sorter)

	args := [2]string{}
	switch {
	case len(ops) > 1:
		args[1] = ops[1]
		fallthrough
	case len(ops) > 0:
		args[0] = ops[0]
	}
	return vr(sorter.sortArr, sorter.kvs, args[0], args[1])
}

type serviceKeySorter struct {
	sortArr []string
	kvs     map[string]string
	cmp     func(i, j string) bool
}

func (sks *serviceKeySorter) Len() int {
	return len(sks.sortArr)
}

func (sks *serviceKeySorter) Swap(i, j int) {
	sks.sortArr[i], sks.sortArr[j] = sks.sortArr[j], sks.sortArr[i]
}

func (sks *serviceKeySorter) Less(i, j int) bool {
	return sks.cmp(sks.sortArr[i], sks.sortArr[j])
}

func stringToBytesVersion(versionStr string) []byte {
	verSet := strings.Split(versionStr, ".")
	verBytes := make([]byte, len(verSet))
	for i, v := range verSet {
		integer, err := strconv.ParseInt(v, 10, 8)
		if err != nil {
			return []byte{}
		}
		verBytes[i] = byte(integer)
	}
	return verBytes[:]
}

func Larger(start, end string) bool {
	startVerBytes := stringToBytesVersion(start)
	endVerBytes := stringToBytesVersion(end)
	return bytes.Compare(startVerBytes, endVerBytes) > 0
}

func Latest(sorted []string, kvs map[string]string, start, end string) []string {
	if len(sorted) == 0 {
		return []string{}
	}
	return []string{kvs[sorted[0]]}
}

func Range(sorted []string, kvs map[string]string, start, end string) []string {
	result := make([]string, len(sorted))
	i, flag := 0, 0

	if Larger(start, end) {
		start, end = end, start
	}

	if len(sorted) == 0 || Larger(start, sorted[0]) || Larger(sorted[len(sorted)-1], end) {
		return []string{}
	}

	for _, k := range sorted {
		// end >= k >= start
		switch flag {
		case 0:
			if Larger(k, end) {
				continue
			}
			flag = 1
		case 1:
			if Larger(start, k) {
				return result[:i]
			}
		}

		result[i] = kvs[k]
		i++
	}
	return result[:i]
}

func AtLess(sorted []string, kvs map[string]string, start, end string) []string {
	result := make([]string, len(sorted))

	if len(sorted) == 0 || Larger(start, sorted[0]) {
		return []string{}
	}

	for i, k := range sorted {
		if Larger(start, k) {
			return result[:i]
		}
		result[i] = kvs[k]
	}
	return result[:]
}

func ParseVersionRule(versionRule string) func(kvs []*mvccpb.KeyValue) []string {
	rangeIdx := strings.Index(versionRule, "-")
	switch {
	case versionRule == "latest":
		return func(kvs []*mvccpb.KeyValue) []string {
			return VersionRule(Latest).Match(kvs)
		}
	case versionRule[len(versionRule)-1:] == "+":
		// 取最低版本及高版本集合
		start := versionRule[:len(versionRule)-1]
		return func(kvs []*mvccpb.KeyValue) []string {
			return VersionRule(AtLess).Match(kvs, start)
		}
	case rangeIdx > 0:
		// 取版本范围集合
		start := versionRule[:rangeIdx]
		end := versionRule[rangeIdx+1:]
		return func(kvs []*mvccpb.KeyValue) []string {
			return VersionRule(Range).Match(kvs, start, end)
		}
	default:
		// 精确匹配
		return nil
	}
}

func VersionMatchRule(version string, versionRule string) bool {
	match := ParseVersionRule(versionRule)
	if match == nil {
		return version == versionRule
	}

	return len(match([]*mvccpb.KeyValue{
		{Key: util.StringToBytesWithNoCopy("/" + version)},
	})) > 0
}
