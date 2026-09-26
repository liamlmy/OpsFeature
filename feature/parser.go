package feature

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Parser struct {
	OpExtractor     *Extractor
	FeatureNameList []string
	IsDebug         int
	AddTokenList    []string
	LabelList       []string
	Label           string
	Filter          string
	requiredInput   []string
}

func InitParser(parseConfPath string) (*Parser, error) {
	parseConfMap, err := loadParseConfig(parseConfPath)
	if err != nil {
		return nil, err
	}

	featureListPath := getValue(parseConfMap, "feature_list")
	if featureListPath == "" {
		err = errors.New("feature_list not found")
		return nil, err
	}

	parser := new(Parser)

	feaCols := getValue(parseConfMap, "fea_cols")
	if feaCols == "" {
		err = errors.New("fea_cols is blank")
		return nil, err
	}
	var featureNameList []string
	featureNameList = strings.Split(feaCols, ",")
	parser.FeatureNameList = featureNameList
	parser.Filter = getValue(parseConfMap, "filter")
	parser.requiredInput = buildRequiredInput(featureNameList)

	debugStr := getValue(parseConfMap, "debug")
	var isDebug int
	if debugStr != "" {
		isDebug, err = strconv.Atoi(debugStr)
		if err != nil {
			return nil, err
		}
	}
	parser.IsDebug = isDebug

	addToken := getValue(parseConfMap, "add_token")
	var addTokenList []string
	if addToken != "" {
		addTokenList = strings.Split(addToken, ",")
	}
	parser.AddTokenList = addTokenList

	label := getValue(parseConfMap, "label")
	var labelList []string
	if label != "" {
		labelList = strings.Split(label, ",")
	}
	parser.LabelList = labelList
	parser.Label = label
	parser.OpExtractor, err = initOpExtractor(featureListPath)
	if err != nil {
		return nil, err
	}
	return parser, nil
}

func initOpExtractor(featureListPath string) (*Extractor, error) {
	file, err := os.Open(featureListPath)
	if err != nil {
		return nil, fmt.Errorf("open feature conf file error, featureListPath is %v, err is %v", featureListPath, err)
	}
	defer file.Close()
	return New(file)
}

func ParseFeatureDataPerLineStrict(data string, featureNameList []string) (map[string]string, error) {
	if data == "" {
		return nil, errors.New("empty line")
	}
	data = strings.TrimRight(data, "\r")

	dataList := strings.Split(data, "\t")
	if len(dataList) != len(featureNameList) {
		return nil, fmt.Errorf("field count mismatch, got %d, want %d", len(dataList), len(featureNameList))
	}
	featureMap := make(map[string]string)
	for i := 0; i < len(dataList); i++ {
		if i >= len(featureNameList) {
			break
		}
		featureMap[featureNameList[i]] = dataList[i]
	}
	return featureMap, nil
}

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
	makeAddToken(addTokenList, featureMap, &output)
	return output.String(), nil
}

func buildLabel(featureMap map[string]string, labelList []string) (string, error) {
	label := "0"
	for i := 0; i < len(labelList); i++ {
		labelName := labelList[i]

		curLabel := getValue(featureMap, labelName)
		if curLabel == "" {
			err := errors.New("labelName not found")
			return "", err
		}
		if curLabel != "0" {
			label = fmt.Sprintf("%v", len(labelList) - i)
			break
		}
	}
	return label, nil
}

func makeAddToken(addTokenList []string, featureMap map[string]string, output *strings.Builder) {
	if len(addTokenList) == 0 {
		return
	}
	output.WriteString(" #")

	for i := 0; i < len(addTokenList); i++ {
		val := getValue(featureMap, addTokenList[i])
		if val == "" {
			fmt.Fprintf(os.Stderr, "token not found, token is %v\n", addTokenList[i])
			continue
		}
		output.WriteString(val)
		output.WriteByte('\t')
	}
}

func getValue(configMap map[string]string, key string) string {
	if value, ok := configMap[key]; ok {
		return value
	}
	return ""
}

func loadParseConfig(parseConfPath string) (map[string]string, error) {
	file, err := os.Open(parseConfPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	parseConfMap := make(map[string]string)
	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		lineList := strings.SplitN(line, "=", 2)
		if len(lineList) != 2 {
			return nil, fmt.Errorf("parse config line error: %s", line)
		}
		key := strings.TrimSpace(lineList[0])
		value := strings.TrimSpace(lineList[1])
		if key == "" {
			return nil, fmt.Errorf("parse config key is empty: %s", line)
		}
		if strings.HasPrefix(value, "\"") {
			unquoted, err := strconv.Unquote(value)
			if err != nil {
				return nil, fmt.Errorf("parse quoted config value error, line is %s, err is %v", line, err)
			}
			value = unquoted
		}
		parseConfMap[key] = value
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	for key, value := range parseConfMap {
		parseConfMap[key] = os.Expand(value, func(name string) string {
			if val, ok := parseConfMap[name]; ok {
				return val
			}
			return ""
		})
	}
	return parseConfMap, nil
}

func buildRequiredInput(featureNameList []string) []string {
	featureSet := make(map[string]bool, len(featureNameList))
	for _, name := range featureNameList {
		featureSet[name] = true
	}
	required := make([]string, 0, 2)
	for _, name := range []string{"traceid", "s_id"} {
		if featureSet[name] {
			required = append(required, name)
		}
	}
	return required
}

func (off *Parser) ValidRequiredInput(featureMap map[string]string) bool {
	for _, name := range off.requiredInput {
		if strings.TrimSpace(getValue(featureMap, name)) == "" {
			return false
		}
	}
	return true
}

// func (off *offlineExtractor) OutFeaSize() {
// 	var res int64
// 	for _, featureBase := range off.Extractor.featureBaseList {
// 		if featureBase.GetSlot
// 	}
// }
