package code

import (
	"encoding/binary"
	"fmt"
)

type Opcode byte

const (
	OpConstant Opcode = iota
	OpPop
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpTrue
	OpFalse
	OpNull
	OpEqual
	OpNotEqual
	OpGreaterThan
	OpGreaterEqual
	OpMinus
	OpBang
	OpJumpNotTruthy
	OpJump
	OpGetGlobal
	OpSetGlobal
	OpGetLocal
	OpSetLocal
	OpCallBuiltin // operand: builtin index, then arg count
	OpCall        // operand: arg count
	OpReturnValue
	OpReturn
	OpArray
	OpHash
	OpIndex
	OpSetIndex
	OpSetupTry // operand: catch IP
	OpEndTry
	OpThrow
	OpMakeIter
	OpHasIter
	OpIterValue
	OpNew       // operand: argc — class, args… → instance
	OpParentRef // instance → ParentRef (הורה)
	OpGetFree
	OpSetFree
	OpClosure      // operands: const index, num free
	OpGetLocalCell // local → Cell (יוצר תא במקום אם צריך)
	OpGetFreeCell  // free → Cell (לסגירה מקוננת)
	OpJumpFalsyKeep  // אם הצמרת שקרית — קפיצה ושמירת הערך; אחרת Pop והמשך
	OpJumpTruthyKeep // אם הצמרת אמת — קפיצה ושמירת הערך; אחרת Pop והמשך
	OpDup            // שכפול צמרת המחסנית
	OpJumpTruthy     // Pop; אם אמת — קפיצה
	OpPow            // חזקה
	OpJumpNotNullKeep // אם הצמרת אינה ריק — קפיצה ושמירה; אחרת Pop והמשך
)

type Definition struct {
	Name          string
	OperandWidths []int
}

var definitions = map[Opcode]*Definition{
	OpConstant:      {"OpConstant", []int{2}},
	OpPop:           {"OpPop", []int{}},
	OpAdd:           {"OpAdd", []int{}},
	OpSub:           {"OpSub", []int{}},
	OpMul:           {"OpMul", []int{}},
	OpDiv:           {"OpDiv", []int{}},
	OpMod:           {"OpMod", []int{}},
	OpTrue:          {"OpTrue", []int{}},
	OpFalse:         {"OpFalse", []int{}},
	OpNull:          {"OpNull", []int{}},
	OpEqual:         {"OpEqual", []int{}},
	OpNotEqual:      {"OpNotEqual", []int{}},
	OpGreaterThan:   {"OpGreaterThan", []int{}},
	OpGreaterEqual:  {"OpGreaterEqual", []int{}},
	OpMinus:         {"OpMinus", []int{}},
	OpBang:          {"OpBang", []int{}},
	OpJumpNotTruthy: {"OpJumpNotTruthy", []int{2}},
	OpJump:          {"OpJump", []int{2}},
	OpGetGlobal:     {"OpGetGlobal", []int{2}},
	OpSetGlobal:     {"OpSetGlobal", []int{2}},
	OpGetLocal:      {"OpGetLocal", []int{2}},
	OpSetLocal:      {"OpSetLocal", []int{2}},
	OpCallBuiltin:   {"OpCallBuiltin", []int{1, 1}}, // builtin idx, argc
	OpCall:          {"OpCall", []int{1}},
	OpReturnValue:   {"OpReturnValue", []int{}},
	OpReturn:        {"OpReturn", []int{}},
	OpArray:         {"OpArray", []int{2}},
	OpHash:          {"OpHash", []int{2}},
	OpIndex:         {"OpIndex", []int{}},
	OpSetIndex:      {"OpSetIndex", []int{}},
	OpSetupTry:      {"OpSetupTry", []int{2}},
	OpEndTry:        {"OpEndTry", []int{}},
	OpThrow:         {"OpThrow", []int{}},
	OpMakeIter:      {"OpMakeIter", []int{}},
	OpHasIter:       {"OpHasIter", []int{}},
	OpIterValue:     {"OpIterValue", []int{}},
	OpNew:           {"OpNew", []int{1}},
	OpParentRef:     {"OpParentRef", []int{}},
	OpGetFree:        {"OpGetFree", []int{2}},
	OpSetFree:        {"OpSetFree", []int{2}},
	OpClosure:        {"OpClosure", []int{2, 1}},
	OpGetLocalCell:   {"OpGetLocalCell", []int{2}},
	OpGetFreeCell:    {"OpGetFreeCell", []int{2}},
	OpJumpFalsyKeep:  {"OpJumpFalsyKeep", []int{2}},
	OpJumpTruthyKeep: {"OpJumpTruthyKeep", []int{2}},
	OpDup:             {"OpDup", []int{}},
	OpJumpTruthy:      {"OpJumpTruthy", []int{2}},
	OpPow:             {"OpPow", []int{}},
	OpJumpNotNullKeep: {"OpJumpNotNullKeep", []int{2}},
}

func Lookup(op Opcode) (*Definition, error) {
	def, ok := definitions[op]
	if !ok {
		return nil, fmt.Errorf("אופקוד לא מוכר: %d", op)
	}
	return def, nil
}

type Instructions []byte

func Make(op Opcode, operands ...int) []byte {
	def, err := Lookup(op)
	if err != nil {
		return []byte{}
	}
	instructionLen := 1
	for _, w := range def.OperandWidths {
		instructionLen += w
	}
	instruction := make([]byte, instructionLen)
	instruction[0] = byte(op)
	offset := 1
	for i, o := range operands {
		width := def.OperandWidths[i]
		switch width {
		case 2:
			binary.BigEndian.PutUint16(instruction[offset:], uint16(o))
		case 1:
			instruction[offset] = byte(o)
		}
		offset += width
	}
	return instruction
}

func ReadOperands(def *Definition, ins Instructions) ([]int, int) {
	operands := make([]int, len(def.OperandWidths))
	offset := 0
	for i, width := range def.OperandWidths {
		switch width {
		case 2:
			operands[i] = int(binary.BigEndian.Uint16(ins[offset:]))
		case 1:
			operands[i] = int(ins[offset])
		}
		offset += width
	}
	return operands, offset
}

func (ins Instructions) String() string {
	var out string
	i := 0
	for i < len(ins) {
		def, err := Lookup(Opcode(ins[i]))
		if err != nil {
			out += fmt.Sprintf("ERROR: %s\n", err)
			i++
			continue
		}
		operands, read := ReadOperands(def, ins[i+1:])
		out += fmt.Sprintf("%04d %s", i, formatInstruction(def, operands))
		out += "\n"
		i += 1 + read
	}
	return out
}

func formatInstruction(def *Definition, operands []int) string {
	count := len(def.OperandWidths)
	if len(operands) != count {
		return fmt.Sprintf("ERROR: operand len %d want %d", len(operands), count)
	}
	switch count {
	case 0:
		return def.Name
	case 1:
		return fmt.Sprintf("%s %d", def.Name, operands[0])
	case 2:
		return fmt.Sprintf("%s %d %d", def.Name, operands[0], operands[1])
	}
	return def.Name
}
