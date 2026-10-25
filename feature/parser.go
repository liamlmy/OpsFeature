package feature

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Parser holds the input schema, output settings, and ordered feature extractor.
type Parser struct {
	OpExtractor     *Extractor
	FeatureNameList []string
	IsDebug         int
	AddTokenList    []string
	LabelList       []string
	Label           string
	Filter          string

	requiredInput []string
}

// InitParser loads parse.conf and the feature configuration it references.
func InitParser(parseConfPath string) (*Parser, error) {
	conf, err := loadParseConfig(parseConfPath)
	if err != nil {
		return nil, err
	}
	featureListPath := conf["feature_list"]
	if featureListPath == "" {
		return nil, errors.New("feature_list not found")
	}
	if conf["fea_cols"] == "" {
		return nil, errors.New("fea_cols is blank")
	}

	featureNames, err := parseColumnList(conf["fea_cols"], "fea_cols")
	if err != nil {
		return nil, err
	}
	parser := &Parser{
		FeatureNameList: featureNames,
		Filter:          conf["filter"],
		Label:           conf["label"],
	}
	parser.requiredInput = buildRequiredInput(featureNames)
	if value := conf["debug"]; value != "" {
		parser.IsDebug, err = strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
	}
	if value := conf["add_token"]; value != "" {
		parser.AddTokenList, err = parseColumnList(value, "add_token")
		if err != nil {
			return nil, err
		}
	}
	inputNames := make(map[string]bool, len(featureNames))
	for _, name := range featureNames {
		inputNames[name] = true
	}
	for _, name := range parser.AddTokenList {
		if !inputNames[name] {
			return nil, fmt.Errorf("add_token field %q not found in fea_cols", name)
		}
	}
	if parser.Label != "" {
		parser.LabelList = strings.Split(parser.Label, ",")
	}
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

func parseColumnList(value, key string) ([]string, error) {
	names := strings.Split(value, ",")
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if name == "" || strings.TrimSpace(name) != name {
			return nil, fmt.Errorf("%s contains an empty or whitespace-padded field name", key)
		}
		if seen[name] {
			return nil, fmt.Errorf("%s contains duplicate field %q", key, name)
		}
		seen[name] = true
	}
	return names, nil
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
