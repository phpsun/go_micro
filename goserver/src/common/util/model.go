package util

import "reflect"

// 数据模型转换为协议模型
// @param *interface{} dbModel        数据模型
// @param *interface{} pbModel        协议模型
// @param string 	   tagName        数据模型字段标签名，用于转换为不同的协议模型字段（默认为pb）
// @param bool         ignoreUntagged 是否忽略未定义标签的字段，true表示不转换没标签的字段，false表示用同名字段转换（默认为false）
func DBModelToPBModel(dbModel, pbModel interface{}, opts ...interface{}) {
	tagName := "pb"
	if len(opts) > 0 && opts[0] != nil {
		tagName = opts[0].(string)
	}
	ignoreUntagged := false
	if len(opts) > 1 && opts[1] != nil {
		ignoreUntagged = opts[1].(bool)
	}
	dt := reflect.TypeOf(dbModel).Elem()
	dv := reflect.ValueOf(dbModel).Elem()
	fieldNum := dt.NumField()
	valueMap := make(map[string]reflect.Value, fieldNum)
	for i := 0; i < fieldNum; i++ {
		df := dt.Field(i)
		if name, ok := df.Tag.Lookup(tagName); ok {
			if name == "-" {
				continue
			}
			valueMap[name] = dv.Field(i)
		} else if !ignoreUntagged {
			valueMap[df.Name] = dv.Field(i)
		}

	}
	pt := reflect.TypeOf(pbModel).Elem()
	pv := reflect.ValueOf(pbModel).Elem()
	fieldNum = pt.NumField()
	for i := 0; i < fieldNum; i++ {
		pf := pt.Field(i)
		fieldName := pf.Name
		sourceVal, ok := valueMap[fieldName]
		if !ok || !sourceVal.IsValid() {
			continue
		}
		targetVal := pv.Field(i)
		if targetVal.Type() != sourceVal.Type() {
			targetVal.Set(sourceVal.Convert(targetVal.Type()))
		} else {
			targetVal.Set(sourceVal)
		}
	}
}

// Struct模型转换为Map
// @param obj interface{} struct对象
// @return map[string]interface{}
func ModelToMap(obj interface{}) map[string]interface{} {
	rType := reflect.TypeOf(obj)
	rValue := reflect.ValueOf(obj)
	if rType.Kind() == reflect.Ptr {
		rType = rType.Elem()
		rValue = rValue.Elem()
	}
	if rType.Kind() != reflect.Struct {
		panic("The 'obj' param is not a struct")
	}
	ret := make(map[string]interface{})
	for i, num := 0, rType.NumField(); i < num; i++ {
		rt := rType.Field(i)
		rv := rValue.Field(i)
		if rv.Kind() == reflect.Struct {
			ret[rt.Name] = ModelToMap(rv.Interface())
		} else {
			ret[rt.Name] = rv.Interface()
		}
	}
	return ret
}
