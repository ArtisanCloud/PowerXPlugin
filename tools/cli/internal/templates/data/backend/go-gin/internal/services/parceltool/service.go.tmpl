package parceltool

import (
	"encoding/json"
	"errors"
	"math/big"
	"regexp"
	"strings"
)

var ErrInvalid = errors.New("parcel.input_invalid")
var finite = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?$`)

// Volume 是插件自己的领域算法；Core 只核验 typed 输入/输出与实际执行证据。
func Volume(values map[string]json.RawMessage) (string, error) {
	if len(values) != 3 {
		return "", ErrInvalid
	}
	result := big.NewRat(1, 1)
	scale := 0
	for _, key := range []string{"length_cm", "width_cm", "height_cm"} {
		literal := string(values[key])
		if len(literal) > 80 || !finite.MatchString(literal) {
			return "", ErrInvalid
		}
		value, ok := new(big.Rat).SetString(literal)
		if !ok || value.Sign() <= 0 {
			return "", ErrInvalid
		}
		result.Mul(result, value)
		if point := strings.IndexByte(literal, '.'); point >= 0 {
			scale += len(literal) - point - 1
		}
	}
	output := result.FloatString(scale)
	if strings.Contains(output, ".") {
		output = strings.TrimRight(strings.TrimRight(output, "0"), ".")
	}
	if len(output) > 80 {
		return "", ErrInvalid
	}
	return output, nil
}
