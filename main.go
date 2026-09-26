package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/liamlmy/OpsFeature/feature"
	_ "github.com/liamlmy/OpsFeature/operator"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "cmd args len wrong, args len is", len(os.Args))
		os.Exit(1)
	}

	// parse.conf文件地址
	parseConfPath := os.Args[1]

	// 初始化Parser
	parser, err := feature.InitParser(parseConfPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init offline extractor error: %v\n", err)
		os.Exit(1)
	}

	// if len(os.Args) > 2 {
	// 	//fmt.Println("start to get feature info")
	// 	cmd := os.Args[2]
	// 	switch cmd {
	// 	case "fea_size":
	// 		parser.OutFeaSize()	// 输出所有特征长度之和 featureLength之和
	// 	case "fea_slot":
	// 		parser.OutFeaSlot()	// 输出每个特征的名字、槽位和长度，featureName, slot, featureLength
	// 	case "thead":
	// 		parser.OutFeaThead()	// 输出每个特征名字和长度下标
	// 	}
	// 	return
	// }
	var (
		featureData string
	)

	// ctx := context.Background()
	// fmt.Println("Debug ||%v||", ctx)	// liam

	// 跨行复用对象：State 每行 Reset，
	// 结果 slice 每行传 [:0]，底层数组跨行复用，热路径上不重复分配。
	var (
		opInput = &feature.Input{}
		opState = feature.NewState()
		opFids  []feature.Fid
	)
	if parser.IsDebug == 1 {
		// operator 包没有 Debug()，且 Fid 不带特征名，无法还原原有的 debug 行。
		fmt.Fprintln(os.Stderr, "debug is not supported, fall back to normal output")
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64 * 1024), 8 * 1024 * 1024)
	// 一行一行读入特征数据
	for scanner.Scan() {
		featureData = scanner.Text()
		// 解析一行特征数据
		featureMap, err := feature.ParseFeatureDataPerLineStrict(featureData, parser.FeatureNameList)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip malformed input line: %v\n", err)
			continue
		}
		if !parser.ValidRequiredInput(featureMap) {
			fmt.Fprintln(os.Stderr, "skip input line with empty required key fields")
			continue
		}

		otherFeatureMap := make(map[string]interface{}, len(featureMap))
		for k, v := range featureMap {
			otherFeatureMap[k] = v
		}
		opInput.Score = otherFeatureMap
		opState.Reset()
		opFids, err = parser.OpExtractor.Extract(opInput, opState, opFids[:0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "operator Extract error: %v\n", err)
		}
		output, err := feature.OutputPerLineOp(featureMap, parser.LabelList, parser.AddTokenList, opFids)
		if err != nil {
			fmt.Fprintf(os.Stderr, "OutputPerLineOp error: %v\n", err)
			continue
		}
		fmt.Println(output)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "read stdin error: %v\n", err)
		os.Exit(1)
	}
}
