package utils

import (
	"encoding/json"
	"reflect"
	"strings"
)

// Coalesce 返回 val，如果 val 是零值则返回 defaultVal
func Coalesce[T comparable](val, defaultVal T) T {
	var zero T
	if val == zero {
		return defaultVal
	}
	return val
}

// SafeJSONLoad 将 map[string]any 转换为目标类型 T
func SafeJSONLoad(data string, dst any) error {
	if data == "" {
		return nil
	}
	err := json.Unmarshal([]byte(data), &dst)
	return err
}

// MapToStruct 将 map[string]any 转换为目标结构体
// 优先使用 json tag 进行字段映射，如果没有 json tag 则使用字段名
// dst 必须是结构体指针
func MapToStruct(data map[string]any, dst any) {
	if data == nil || dst == nil {
		return
	}

	// 通过 JSON 序列化/反序列化实现转换（自动处理 json tag）
	bytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	json.Unmarshal(bytes, dst)
}

// GetFieldName 获取结构体字段的映射名称
// 优先返回 json tag，如果没有 json tag 则返回字段名
func GetFieldName(field reflect.StructField) string {
	// 获取 json tag
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" {
		// 处理 json tag 中的选项（如 "name,omitempty"）
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" && parts[0] != "-" {
			return parts[0]
		}
	}
	// 没有 json tag，使用字段名
	return field.Name
}

// StructToMap 将结构体转换为 map[string]any
// 优先使用 json tag 进行字段映射，如果没有 json tag 则使用字段名
func StructToMap(obj any) map[string]any {
	result := make(map[string]any)

	// 通过 JSON 序列化/反序列化实现转换
	bytes, err := json.Marshal(obj)
	if err != nil {
		return result
	}

	json.Unmarshal(bytes, &result)
	return result
}
