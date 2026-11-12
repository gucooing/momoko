// Package jsondoc 编辑 JSON 文档并保留键的原有顺序与未识别的字段：
// 读取 → 按点分路径读取 / 写入 → 以两个空格缩进写回。用于改写第三方程序的配置文件，
// 只动需要改的键，其余内容（包括本程序不认识的字段）原样保留。
package jsondoc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Object 是保持键顺序的 JSON 对象。
type Object struct {
	keys   []string
	values map[string]any
}

// NewObject 创建空对象。
func NewObject() *Object { return &Object{values: map[string]any{}} }

// Keys 按原有顺序返回键。
func (o *Object) Keys() []string { return append([]string(nil), o.keys...) }

// Get 读取一个键。
func (o *Object) Get(key string) (any, bool) {
	v, ok := o.values[key]
	return v, ok
}

// Set 写入一个键（新键追加在末尾）。
func (o *Object) Set(key string, v any) {
	if _, ok := o.values[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.values[key] = v
}

// Delete 删除一个键。
func (o *Object) Delete(key string) {
	if _, ok := o.values[key]; !ok {
		return
	}
	delete(o.values, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
}

// MarshalJSON 按原有顺序输出。
func (o *Object) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		vb, err := marshalValue(o.values[k])
		if err != nil {
			return nil, err
		}
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func marshalValue(v any) ([]byte, error) {
	enc := new(bytes.Buffer)
	e := json.NewEncoder(enc)
	e.SetEscapeHTML(false)
	if err := e.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(enc.Bytes(), "\n"), nil
}

// Parse 解析一个 JSON 文档（顶层必须是对象）。数字保留原文（json.Number）。
func Parse(raw []byte) (*Object, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("jsondoc: top level must be an object")
	}
	obj, err := parseObject(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("jsondoc: trailing data after the document")
	}
	return obj, nil
}

func parseObject(dec *json.Decoder) (*Object, error) {
	obj := NewObject()
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("jsondoc: object key expected")
		}
		v, err := parseValue(dec)
		if err != nil {
			return nil, err
		}
		obj.Set(key, v)
	}
	if _, err := dec.Token(); err != nil { // '}'
		return nil, err
	}
	return obj, nil
}

func parseValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			return parseObject(dec)
		case '[':
			arr := []any{}
			for dec.More() {
				v, err := parseValue(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, v)
			}
			if _, err := dec.Token(); err != nil { // ']'
				return nil, err
			}
			return arr, nil
		}
		return nil, fmt.Errorf("jsondoc: unexpected delimiter %v", t)
	default:
		return t, nil
	}
}

// Lookup 按点分路径读取（例如 "database.driver"）。
func (o *Object) Lookup(path string) (any, bool) {
	cur := any(o)
	for _, part := range strings.Split(path, ".") {
		obj, ok := cur.(*Object)
		if !ok {
			return nil, false
		}
		cur, ok = obj.values[part]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

// Assign 按点分路径写入，缺少的中间对象自动创建（追加在末尾）。
func (o *Object) Assign(path string, v any) error {
	parts := strings.Split(path, ".")
	cur := o
	for _, part := range parts[:len(parts)-1] {
		next, ok := cur.values[part]
		if !ok {
			child := NewObject()
			cur.Set(part, child)
			cur = child
			continue
		}
		child, ok := next.(*Object)
		if !ok {
			return fmt.Errorf("jsondoc: %s is not an object", part)
		}
		cur = child
	}
	cur.Set(parts[len(parts)-1], v)
	return nil
}

// Plain 转为普通的 map / slice 结构（数字仍为 json.Number），便于序列化给前端或做比较。
func Plain(v any) any {
	switch t := v.(type) {
	case *Object:
		m := make(map[string]any, len(t.keys))
		for _, k := range t.keys {
			m[k] = Plain(t.values[k])
		}
		return m
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = Plain(x)
		}
		return out
	default:
		return v
	}
}

// FromPlain 把普通结构（map[string]any 等）转回文档值；新对象的键按字母序。
func FromPlain(v any) any {
	switch t := v.(type) {
	case map[string]any:
		obj := NewObject()
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			obj.Set(k, FromPlain(t[k]))
		}
		return obj
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = FromPlain(x)
		}
		return out
	default:
		return v
	}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// Marshal 以两个空格缩进输出（末尾带换行）。
func Marshal(o *Object) ([]byte, error) {
	raw, err := o.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, raw, "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

// Equal 判断两个文档值是否相同（数字按数值比较）。
func Equal(a, b any) bool {
	ra, err1 := json.Marshal(normalizeNumbers(Plain(a)))
	rb, err2 := json.Marshal(normalizeNumbers(Plain(b)))
	return err1 == nil && err2 == nil && bytes.Equal(ra, rb)
}

func normalizeNumbers(v any) any {
	switch t := v.(type) {
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t.String()
	case map[string]any:
		for k, x := range t {
			t[k] = normalizeNumbers(x)
		}
		return t
	case []any:
		for i, x := range t {
			t[i] = normalizeNumbers(x)
		}
		return t
	default:
		return v
	}
}
