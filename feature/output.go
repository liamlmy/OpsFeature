package feature

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// OutputPerLineOp formats a label, FIDs, and optional trailing tokens.
func OutputPerLineOp(featureMap map[string]string, labelList []string, addTokenList []string, fidList []Fid) (string, error) {
	label, err := buildLabel(featureMap, labelList)
	if err != nil {
		return "", err
	}

	var output strings.Builder
	output.Grow(len(label) + len(fidList)*24)
	output.WriteString(label)

	var numberBuffer [32]byte
	for i := 0; i < len(fidList); i++ {
		output.WriteByte(' ')
		digits := strconv.AppendInt(numberBuffer[:0], int64(fidList[i].Slot), 10)
		output.Write(digits)
		output.WriteByte(':')
		digits = strconv.AppendUint(numberBuffer[:0], fidList[i].Val, 10)
		output.Write(digits)
	}
	if err := makeAddToken(addTokenList, featureMap, &output); err != nil {
		return "", err
	}
	return output.String(), nil
}

func buildLabel(featureMap map[string]string, labelList []string) (string, error) {
	label := "0"
	for i := 0; i < len(labelList); i++ {
		labelName := labelList[i]

		curLabel := featureMap[labelName]
		if curLabel == "" {
			err := errors.New("labelName not found")
			return "", err
		}
		if curLabel != "0" {
			label = strconv.Itoa(len(labelList) - i)
			break
		}
	}
	return label, nil
}

// Every configured token occupies one field, including empty values. The final
// tab is a required terminator; consumers remove exactly one before splitting.
func makeAddToken(addTokenList []string, featureMap map[string]string, output *strings.Builder) error {
	if len(addTokenList) == 0 {
		return nil
	}
	output.WriteString(" #")
	for _, name := range addTokenList {
		val, exists := featureMap[name]
		if !exists {
			return fmt.Errorf("add_token field %q not found in input", name)
		}
		if strings.ContainsAny(val, "\t\r\n") {
			return fmt.Errorf("add_token field %q contains a TSV delimiter", name)
		}
		output.WriteString(val)
		output.WriteByte('\t')
	}
	return nil
}
