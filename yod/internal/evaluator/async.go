package evaluator

import (
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
	return object.SpawnTask(args[0])
}

func builtinAwait(args ...object.Object) object.Object {
	if len(args) != 1 {
		return &object.Error{Message: "המתן מצפה למשימה אחת"}
	}
	return object.AwaitTask(args[0])
}

func builtinParallel(args ...object.Object) object.Object {
	if len(args) != 1 {
		return &object.Error{Message: "במקביל מצפה לרשימת משימות"}
	}
	return object.ParallelTasks(args[0])
}
