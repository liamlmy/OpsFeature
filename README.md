# OpsFeature

OpsFeature 是用 Go 实现的配置驱动型离线特征抽取工具。它从标准输入逐行读取 TSV 样本，按配置顺序执行算子，向标准输出写入标签与特征 ID：

```text
label slot:fid slot:fid ...
```

支持直接取值、分桶、序列、向量、二阶/三阶交叉、算术和对数缩放；通过 `addcol` 串联中间特征。当前没有第三方 Go 依赖。

## 快速开始

在仓库根目录执行。项目的 [go.mod](go.mod) 声明 Go **1.27.1**，请使用满足该要求的工具链。

```bash
go build -o opsfeature .
./opsfeature examples/minimal/parse.conf < examples/minimal/input.tsv > /tmp/opsfeature-output.txt
diff -u examples/minimal/output.txt /tmp/opsfeature-output.txt
```

`diff` 没有输出即表示结果一致。示例文件均已提供：

| 文件 | 用途 |
|---|---|
| [examples/minimal/parse.conf](examples/minimal/parse.conf) | 输入列、标签、附加字段和特征配置路径 |
| [examples/minimal/features.conf](examples/minimal/features.conf) | 价格取值、价格分桶和历史序列 |
| [examples/minimal/input.tsv](examples/minimal/input.tsv) | 两条输入样本，使用真实 Tab 分隔 |
| [examples/minimal/output.txt](examples/minimal/output.txt) | 对应的完整预期输出，保留末尾 Tab |

示例中 `12.5` 乘以系数 `100` 得到 FID `1250`；价格落入桶号 `1`；历史序列 `10_20` 补齐为三项，并输出到 `32768..32770`。第一条结果为（`\t` 表示真实 Tab）：

```text
1 10:1250 11:1 32768:10 32769:20 32770:0 #trace-001\t10001\t
```

根目录的 [parse.conf](parse.conf) 和 [feature_list.conf](feature_list.conf) 是规模更大的业务配置，需要与其列定义匹配的 TSV 输入。自定义配置的调用方式相同：

```bash
./opsfeature parse.conf < input.tsv > output.txt
```

所有配置文件中的相对路径均以**程序运行时的工作目录**为基准。

## 代码结构

```text
.
├── main.go                  # CLI：读取样本、重置状态、抽取并输出
├── main_test.go             # CLI 示例、坏行、长行和退出状态测试
├── go.mod
├── parse.conf               # 业务输入/输出配置
├── feature_list.conf        # 业务特征配置
├── examples/minimal/        # 可直接运行的最小示例及预期输出
├── feature/                 # 配置与抽取运行时
│   ├── parser.go            # parse.conf 加载和 Parser 初始化
│   ├── input.go             # Input、TSV 解析和必填标识检查
│   ├── output.go            # 标签与附加 token 格式化
│   ├── extract.go           # 特征配置加载、校验和按序执行
│   ├── feature.go           # Operator 接口、Base 配置和参数校验
│   ├── registry.go          # 算子注册与创建
│   ├── api.go               # 提供给 operator 包的公共接口
│   ├── depend.go            # 依赖查找和类型转换
│   ├── source.go            # KV、序列和向量数据解析
│   ├── source_pool.go       # 数据源对象池与回收
│   ├── state.go             # 样本状态、重置和有序 map key
│   ├── cache.go             # FID 缓存与多值去重
│   ├── value.go             # 内部标量表示、转换和哈希
│   ├── fid.go               # Fid 类型、编码和统一输出入口
│   ├── extract_test.go      # 算子链路、编码和输出测试
│   └── parser_test.go       # 列名校验与附加 token 测试
├── operator/                # 算子实现及注册
│   ├── op_registry.go
│   ├── op_direct.go
│   ├── op_bucket.go
│   ├── op_seq.go
│   ├── op_combine.go
│   ├── op_arith.go
│   └── op_scale.go
├── utils/                   # 数组、数值、精度、哈希、地理和时间工具
└── todo.md                  # 已知风险与后续改进项
```

`feature` 包负责配置和运行时，`operator` 包依赖它实现算子，并在 `init()` 中注册。命令行入口通过副作用导入加载全部内置算子。

## 处理流程

1. `InitParser` 读取输入/输出配置和特征配置。
2. 根据 `class` 创建算子，依次加载公共配置、执行 `Init()`、校验公共字段。
3. 每条 TSV 按 `fea_cols` 映射为原始字段，检查列数和必填标识。
4. 调用 `State.Reset()`，按配置顺序执行算子。
5. 生成标签、FID 列表和可选附加 token，写入标准输出。

**配置顺序就是执行顺序**，项目不会构建依赖图或自动拓扑排序。单个算子报错后，其余算子仍继续执行；CLI 记录错误并可能输出已生成的部分 FID。

## 输入与输出配置

最小示例的 `parse.conf`：

```ini
feature_list=./examples/minimal/features.conf
fea_cols=traceid,user_id,item_price,history,label_0,label_1
label=label_0,label_1
add_token=traceid,user_id
debug=0
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `feature_list` | 是 | 特征配置路径 |
| `fea_cols` | 是 | 逗号分隔的 TSV 列名，顺序与输入一致 |
| `label` | 否 | 逗号分隔的标签列名；未配置时输出 `0` |
| `add_token` | 否 | 追加到输出 `#` 后面的原始字段，必须存在于 `fea_cols` |
| `debug` | 否 | 整数；为 `1` 时打印暂不支持的提示，随后按普通模式运行 |
| `filter` | 否 | 只读取和保存，尚未执行过滤 |

配置采用 `key=value` 格式，忽略空行和以 `#` 开头的行；这里不支持行尾注释。以双引号开头的值通过 `strconv.Unquote` 解析。

`$name` 和 `${name}` 引用同一配置文件中的字段，不读取系统环境变量。当前展开仅遍历一次，多层和循环引用没有可靠保证，建议直接填写路径。

`fea_cols` 和 `add_token` 拒绝空列名、重复列名及带首尾空格的列名。`label` 当前只按逗号切分，不会逐项清理空格，也不在初始化时检查其是否存在于 `fea_cols`。未知的配置项会保留，重复 key 以后一个为准。

## 特征配置与依赖

每行定义一个算子，字段用分号分隔，支持 `#` 行尾注释：

```ini
name=price;class=Direct;slot_id=10;depend=item_price;args=1,100,double # 价格放大 100 倍
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `name` | 是 | 唯一特征名，也是写入中间结果的名称 |
| `class` | 是 | 已注册的算子名称，区分大小写 |
| `slot_id` | 是 | 基础槽位，范围 `0..32768`；`0` 不输出 FID |
| `depend` | 视算子而定 | 逗号分隔的依赖字段 |
| `args` | 视算子而定 | 逗号分隔的算子参数 |
| `share_slot` | hash version 4 必填 | 共享哈希槽位，范围 `1..32767` |
| `addcol` | 否 | 非零时将产生的值转成字符串列表，保存到 `State.inner` |
| `list_cache` | 否 | 非零时在当前样本内缓存该特征的 FID 计算结果 |
| `enable_str_out` | 否 | 非零时不向最终列表追加 FID，通常配合 `addcol` 使用 |

未知字段、重复 key、重复特征名和重复非零基础 `slot_id` 会导致初始化失败。不同序列的 fake slot 区间尚未做全局冲突检查。

依赖查找优先级：

```text
Input.Global → Input.Score → State.inner
```

原始输入优先于同名中间特征。下游使用 `addcol` 时，上游必须出现在前面，且应使用独立的中间特征名。例如：

```ini
name=user_tags;class=Direct;slot_id=0;depend=interests;args=0,1,str_dict;addcol=1
name=user_item_cross;class=Combine;slot_id=20;depend=user_tags,item_type;args=addcol,0,1
```

这里的 `interests` 和 `item_type` 需要来自输入；`interests=sports,music`、`item_type=book` 会交叉出 `sports_book` 和 `music_book`。

## FID 编码

`Fid` 包含输出槽位 `Slot` 和编码结果 `Val`。文本格式为 `Slot:Val`。多数算子的前两个参数是 `hash_version`、`coeff`，但 `Combine` 和 `Arithmetic` 的位置不同，见下文。

| hash version | 编码规则 |
|---|---|
| `0` | 基础 `slot_id` 左移 48 位，与值的 CityHash64 低 48 位组合；哈希输入与值的类型有关 |
| `1` | 将数值乘以系数后转为无符号整数，不嵌入 slot |
| `2` | 将数值按版本 1 转换，再取低 48 位并嵌入基础 `slot_id` |
| `3` | 转为 `float32`，以其二进制位模式作为 FID；不嵌入 slot，忽略系数 |
| `4` | 将值转成字符串做 CityHash，与 `share_slot` 组合；忽略系数 |

版本 1/2 在 `coeff <= 1e-6` 时使用系数 `1`。当前整数路径会先把系数转换为整数，小数系数可能被截断；大数乘法也没有完整的溢出保护。字符串值不能直接使用版本 1/2。详细风险见 [todo.md](todo.md)。

序列输出的 `Slot` 是对应位置的 fake slot；版本 0/2 的 FID 内嵌槽位仍是配置的**基础 `slot_id`**，版本 4 使用 `share_slot`。

## 内置算子

以下 `args` 参数均从下标 `0` 开始。没有特别说明时，`hash_version` 默认为 `0`，版本 1/2 的有效默认系数为 `1`。

### Direct

```text
args=hash_version,coeff,value_type,col
```

`value_type` 默认 `double`，`col` 默认 `0`。

| value_type | 输入示例 | 行为 |
|---|---|---|
| `double` | `12.5` | 单个数值 |
| `str_pos` | `vip` | 第一个依赖字段的单个字符串；空值不输出 |
| `kv_int_float` | `101:0.8,205:1.5` | `col` 指定整数 key，输出对应数值 |
| `kv_int_str` | `101:vip,205:new` | `col` 指定整数 key，输出对应字符串 |
| `str_dict` | `sports,music` | 逗号切分并去重输出字符串 |
| `kv_str_float` | `sports:0.8,music:1.5` | `col=0` 输出 key，否则输出数值 |
| `tags` | `1001:0.7,1002:0.3` | `col=0` 输出整数 key，否则输出数值 |
| `add_col` | 上游的中间结果 | 展开字符串、浮点或整数列表，也支持单值；遵循统一依赖优先级 |

map 多值按 key 排序后输出，保证顺序稳定。数值取值遇到 `-999` 时通常按 `0` 处理，依赖名为 `UBDiscountRatio`、`UBDiscountAvg` 时保留原值。

### Bucket / Bucket_truncate

```text
Bucket:          args=hash_version,coeff,mode,阈值或枚举值...
Bucket_truncate: args=hash_version,coeff,low,high[,low,high...]
```

- `Bucket.abslisan`：处理数值精度和缺失值后取绝对值，在升序阈值表中查桶。当前不校验阈值顺序，且单阈值表总返回 `0`。
- `Bucket.dummy`：返回整数在枚举表中的下标，未命中返回枚举数量；`-999` 按 `0` 处理。
- `Bucket_truncate`：按配置顺序将闭区间映射到连续位置，从 `0` 开始。区间端点倒置时自动交换；未命中返回 `0`，不检查区间重叠。

```ini
name=price_bucket;class=Bucket;slot_id=30;depend=item_price;args=1,1,abslisan,10,50,100
name=age_range;class=Bucket_truncate;slot_id=31;depend=age;args=1,1,0,17,18,35,36,60
```

### Seq / Seq_field / Seq_emb

三者公共的前五个参数：

```text
args[0] hash_version
args[1] coeff
args[2] slot_start，必须 >= 32768
args[3] slot_end
args[4] num，必须 > 0 且等于 slot_end-slot_start+1
```

| class | 输入 | 后续参数与行为 |
|---|---|---|
| `Seq` | `10_20_30` | `args[5]` 为无符号整数 padding 值，默认 `0`；非空序列补齐/截断到 `num`，空输入不输出 |
| `Seq_field` | `0:1_2,1:3_4` | `args[5]` 选择字段，默认 `0`；`args[6]` 为 `int` 或 `str`，默认 `int`；`args[7]=1` 启用 padding；`args[8]` 为默认值；未找到子序列或子序列为空时不输出 |
| `Seq_emb` | `0.1,0.2,0.3` | `args[5]` 为 padding 长度，只能是 `0` 或 `num`；`args[6]` 为默认值；空输入也会按配置长度补齐 |

`Seq_field` 配置第二个依赖字段时，将其作为基准时间，计算 `(base_timestamp - sequence_value) / 86400`。

```ini
name=history_seq;class=Seq;slot_id=40;depend=history;args=1,1,32768,32770,3,0
name=embedding;class=Seq_emb;slot_id=41;depend=embedding;args=3,1,32771,32773,3,3,0
```

同一依赖字段使用不同 padding 参数时可能发生缓存串用，详见“当前限制”。

### Combine / Combine_multi

```text
args=addcol,hash_version,coeff
```

`Combine` 严格要求两个依赖，`Combine_multi` 严格要求三个。对依赖值列表做笛卡尔积，用 `_` 连接并去重。首个参数必须是字面量 `addcol`，依赖可以是原始单值或上游的列表结果。

```ini
name=user_item_cross;class=Combine;slot_id=50;depend=user_type,item_type;args=addcol,0,1
```

原始的逗号分隔字符串不会在这里自动展开，应先通过 `Direct.str_dict` 和 `addcol=1` 转成列表。

### Arithmetic

```text
args=operation,hash_version,coeff
```

至少需要两个依赖，使用前两个。

| operation | 行为 |
|---|---|
| `add` | 左值加右值，`-999` 按 `0` 处理 |
| `minus` | 左值减右值，`-999` 按 `0` 处理 |
| `multiply` | 左值乘右值，`-999` 按 `0` 处理 |
| `match` | 两边都为空输出 `2`，相同输出 `1`，否则输出 `0` |
| `div` | 左值除右值，任一值接近 `0` 时输出 `0` |
| `cos` | 两个逗号分隔向量的余弦相似度；空向量或长度不等时不输出，零范数向量输出 `0` |

```ini
name=price_diff;class=Arithmetic;slot_id=60;depend=price_a,price_b;args=minus,3,1
```

### Scale

```text
args=hash_version,coeff,mode
```

`mode` 默认且目前仅支持 `log`：先将负数截断为 `0`，再计算 `log(value+1)`。

```ini
name=price_log;class=Scale;slot_id=61;depend=item_price;args=3,1,log
```

## 输入、标签与附加字段

每行输入一条 TSV，列数必须与 `fea_cols` 完全一致。空行、列数不符以及必填标识为空的行会跳过并写诊断信息。仅当输入列中配置了 `traceid` 或 `s_id` 时，它们才属于必填标识，纯空白同样视为空。

CLI 的 Scanner 单行缓冲上限为 8 MiB，接近该上限时还需考虑行结束符。输入读取失败或超限会终止程序并返回退出码 `1`。

标签规则：

1. 未配置 `label` 时输出 `0`。
2. 按配置顺序读取标签；访问到空值或不存在的字段时报错并跳过该行。
3. 找到第一个不等于字符串 `0` 的标签后立即停止。下标为 `i` 时输出 `len(labelList)-i`，不再检查后面的标签。
4. 全部为 `0` 时输出 `0`。

例如 `label=label_0,label_1` 时，`0,1` 输出 `1`，`1,0` 输出 `2`；输入标签不会作为原始值直接输出。

配置 `add_token` 后的格式：

```text
label slot:fid ... #token1\ttoken2\t
```

token 严格按配置顺序输出，空字符串保留字段位置；末尾始终有一个终止 Tab。消费者应先去掉**恰好一个**终止 Tab，再按 Tab 切分，避免丢失末尾空字段。包 API 会拒绝缺失的 token 字段或包含 Tab、回车、换行的 token 值。

配置初始化失败和输入读取失败返回 `1`；逐行数据错误只跳过或记录，不会使最终退出码自动变为非零。正常结果写 stdout，诊断信息写 stderr。

## 作为 Go 包使用

使用内置算子时，必须副作用导入 `operator` 包以完成注册：

```go
package main

import (
    "fmt"

    "github.com/liamlmy/OpsFeature/feature"
    _ "github.com/liamlmy/OpsFeature/operator"
)

func main() {
    extractor, err := feature.NewFromText(
        "name=price;class=Direct;slot_id=10;depend=price;args=1,100,double",
    )
    if err != nil {
        panic(err)
    }
    state := feature.NewState()
    var fids []feature.Fid
    for _, price := range []string{"12.5", "75"} {
        state.Reset()
        input := &feature.Input{Score: map[string]interface{}{"price": price}}
        fids, err = extractor.Extract(input, state, fids[:0])
        if err != nil {
            panic(err)
        }
        fmt.Println(fids)
    }
    // [{10 1250}]
    // [{10 7500}]
}
```

也可使用 `feature.New(io.Reader)` 读取特征配置，或 `feature.InitParser(path)` 加载完整配置。`Extract` 会向传入切片追加结果；复用切片时需传入 `fids[:0]`。传入 `nil` State 时会自动创建，否则调用方必须在不同样本之间自行 `Reset()`。

`State` 持有中间特征、解析后的数据源、FID 缓存、去重集合及排序缓冲。`Reset()` 清除样本中间结果和缓存，并将数据源归还对象池。一个 State 不应被多个 goroutine 并发使用。

新增算子需实现 `feature.Operator` 的 `Conf()`、`Init()`、`Emit()`，在 `init()` 中调用 `feature.Register`，并通过 `Base` 提供的 `Emit*`/`Flush` 方法生成 FID 和中间特征。注册名即配置中的 `class`；已注册名称可通过 `feature.Classes()` 查询。

## 开发与验证

```bash
go test ./...
go vet ./...
go build -o opsfeature .
```

测试覆盖最小示例的完整输出、多行坏数据跳过、长行读取、初始化/读取失败的退出状态，以及 FID 编码、算子顺序、序列补齐、数组兼容行为、附加 token 和列名校验。

## 当前限制

- `filter` 尚未执行过滤；debug 尚无特征名级别的调试输出。
- 复杂数据源缓存未完整包含解析参数，同一列以不同 padding 配置读取时可能复用先前结果。
- 数值解析失败可能被当作 `0`，部分坏元素被跳过；数值缩放的小数系数和溢出处理仍需完善。
- 序列 fake slot 区间没有全局冲突检查；部分坏元素处理可能改变后续位置。
- `Extract` 可能返回部分 FID 和汇总错误，CLI 仍可能写出部分样本。FID 转换失败也可能只写日志而不进入汇总错误。
- 交叉特征没有组合数量上限；公共 API 传入不支持的动态类型时，部分缓存或去重路径可能 panic。
- 变量展开、配置和边界条件尚未全面校验，测试也未覆盖全部异常路径。

后续工作与验收建议见 [todo.md](todo.md)。
