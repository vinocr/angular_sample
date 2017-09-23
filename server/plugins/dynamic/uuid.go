package dynamic

import (
	"github.com/ServiceComb/service-center/pkg/plugins"
	"github.com/ServiceComb/service-center/util"
	"github.com/ServiceComb/service-center/util/uuid"
	"strings"
)

func buildinUnidFunc() string {
	return strings.Replace(uuid.NewV1().String(), string(uuid.DASH), "", -1)
}

func findUuidFunc(funcName string) func() string {
	ff, err := plugins.FindFunc("uuid", funcName)
	if err != nil {
		return buildinUnidFunc
	}
	f, ok := ff.(func() string)
	if !ok {
		util.Logger().Warnf(nil, "unexpected function '%s' format found in plugin 'uuid'.", funcName)
		return buildinUnidFunc
	}
	return f
}

func GetServiceId() string {
	f := findUuidFunc("GetServiceId")
	return f()
}

func GetInstanceId() string {
	f := findUuidFunc("GetInstanceId")
	return f()
}

func GenerateUuid() string {
	return buildinUnidFunc()
}
