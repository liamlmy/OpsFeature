package feature

import (
	"errors"
	"fmt"
	"strings"
)

// Input holds read-only features for one sample. Global takes precedence over Score.
type Input struct {
	Global map[string]interface{} // 请求级公共特征
	Score  map[string]interface{} // 当前候选的特征
}

func (in *Input) get(name string) (interface{}, bool) {
	if in == nil {
		return nil, false
	}
	if v, ok := in.Global[name]; ok {
		return v, true
	}
	if v, ok := in.Score[name]; ok {
		return v, true
	}
	return nil, false
}

// ParseFeatureDataPerLineStrict maps a TSV row to its column names.
func ParseFeatureDataPerLineStrict(data string, featureNameList []string) (map[string]string, error) {
	if data == "" {
		return nil, errors.New("empty line")
	}
	data = strings.TrimRight(data, "\r")

	dataList := strings.Split(data, "\t")
	if len(dataList) != len(featureNameList) {
		return nil, fmt.Errorf("field count mismatch, got %d, want %d", len(dataList), len(featureNameList))
	}
	featureMap := make(map[string]string, len(dataList))
	for i, value := range dataList {
		featureMap[featureNameList[i]] = value
	}
	return featureMap, nil
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

func (p *Parser) ValidRequiredInput(featureMap map[string]string) bool {
	for _, name := range p.requiredInput {
		if strings.TrimSpace(featureMap[name]) == "" {
			return false
		}
	}
	return true
}
