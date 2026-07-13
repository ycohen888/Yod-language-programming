package stdlib

import (
	"encoding/json"
	"fmt"

	"yod/internal/object"
)

func NewJSONModule() *object.Module {
	m := &object.Module{Name: "JSON", Attrs: map[string]object.Object{}}
	m.Attrs["פרסר"] = &object.Builtin{Fn: jsonParse}
	m.Attrs["מחרוזת"] = &object.Builtin{Fn: jsonStringify}
	return m
}

func jsonParse(args ...object.Object) object.Object {
	if err := expectArgs("JSON.פרסר", 1, args); err != nil {
		return err
	}
	s, ok := asString(args[0])
	if !ok {
		return errObj("JSON.פרסר מצפה למחרוזת")
	}
	var raw any
	if e := json.Unmarshal([]byte(s), &raw); e != nil {
		return errObj("JSON לא תקין: " + e.Error())
	}
	return fromJSON(raw)
}

func jsonStringify(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 2 {
		return errObj("JSON.מחרוזת מצפה לערך ואופציונלי ייפוי (אמת/שקר)")
	}
	pretty := false
	if len(args) == 2 {
		if b, ok := args[1].(*object.Boolean); ok {
			pretty = b.Value
		} else {
			return errObj("JSON.מחרוזת: הארגומנט השני חייב להיות אמת/שקר")
		}
	}
	raw, err := toJSON(args[0])
	if err != nil {
		return errObj(err.Error())
	}
	var data []byte
	var e error
	if pretty {
		data, e = json.MarshalIndent(raw, "", "  ")
	} else {
		data, e = json.Marshal(raw)
	}
	if e != nil {
		return errObj("לא הצלחתי להמיר ל־JSON: " + e.Error())
	}
	return &object.String{Value: string(data)}
}

func fromJSON(v any) object.Object {
	switch x := v.(type) {
	case nil:
		return object.Nil
	case bool:
		return &object.Boolean{Value: x}
	case float64:
		return &object.Number{Value: x}
	case string:
		return &object.String{Value: x}
	case []any:
		arr := &object.Array{Elements: make([]object.Object, len(x))}
		for i, el := range x {
			arr.Elements[i] = fromJSON(el)
		}
		return arr
	case map[string]any:
		h := &object.Hash{Pairs: make(map[string]object.Object, len(x))}
		for k, el := range x {
			h.Pairs[k] = fromJSON(el)
		}
		return h
	default:
		return errObj(fmt.Sprintf("טיפוס JSON לא נתמך: %T", v))
	}
}

func toJSON(obj object.Object) (any, error) {
	switch o := obj.(type) {
	case *object.Null:
		return nil, nil
	case *object.Boolean:
		return o.Value, nil
	case *object.Number:
		return o.Value, nil
	case *object.String:
		return o.Value, nil
	case *object.Array:
		out := make([]any, len(o.Elements))
		for i, e := range o.Elements {
			v, err := toJSON(e)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	case *object.Hash:
		out := make(map[string]any, len(o.Pairs))
		for k, e := range o.Pairs {
			v, err := toJSON(e)
			if err != nil {
				return nil, err
			}
			out[k] = v
		}
		return out, nil
	default:
		return nil, fmt.Errorf("לא ניתן להמיר %s ל־JSON", o.Type())
	}
}
