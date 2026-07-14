package evaluator

import (
	"sync"

	"yod/internal/object"
)

func init() {
	builtins["משימה"] = &object.Builtin{Fn: builtinTask}
	builtins["המתן"] = &object.Builtin{Fn: builtinAwait}
	builtins["במקביל"] = &object.Builtin{Fn: builtinParallel}
}

func builtinTask(args ...object.Object) object.Object {
	if len(args) != 1 {
		return &object.Error{Message: "משימה מצפה לפונקציה אחת"}
	}
	fn, ok := args[0].(*object.Function)
	if !ok {
		return &object.Error{Message: "משימה מצפה לפונקציה"}
	}
	t := &object.Task{Ch: make(chan object.Object, 1)}
	go func() {
		res := callUserFunction(fn, nil, nil, nil, 1)
		t.Ch <- res
	}()
	return t
}

func builtinAwait(args ...object.Object) object.Object {
	if len(args) != 1 {
		return &object.Error{Message: "המתן מצפה למשימה אחת"}
	}
	return awaitOne(args[0])
}

func awaitOne(obj object.Object) object.Object {
	t, ok := obj.(*object.Task)
	if !ok {
		return &object.Error{Message: "המתן מצפה לערך מסוג משימה"}
	}
	if t.Done {
		return t.Val
	}
	val := <-t.Ch
	t.Val = val
	t.Done = true
	return val
}

func builtinParallel(args ...object.Object) object.Object {
	if len(args) != 1 {
		return &object.Error{Message: "במקביל מצפה לרשימת משימות"}
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return &object.Error{Message: "במקביל מצפה לרשימה"}
	}
	out := make([]object.Object, len(arr.Elements))
	var wg sync.WaitGroup
	for i, el := range arr.Elements {
		wg.Add(1)
		go func(i int, el object.Object) {
			defer wg.Done()
			out[i] = awaitOne(el)
		}(i, el)
	}
	wg.Wait()
	return &object.Array{Elements: out}
}
