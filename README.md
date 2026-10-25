# OpsFeature

OpsFeature 是用 Go 实现的配置驱动型离线特征抽取工具。它从标准输入逐行读取 TSV 样本，按配置顺序执行算子，向标准输出写入标签与特征 ID：

```text
label slot:fid slot:fid ...
```

支持直接取值、分桶、序列、向量、二阶/三阶交叉、算术和对数缩放；通过 `addcol` 串联中间特征。当前没有第三方 Go 依赖。

## 构建与运行

项目的 [go.mod](go.mod) 声明 Go **1.27.1**。在仓库根目录构建：

```bash
go build -o opsfeature .
```

使用仓库现有的 [parse.conf](parse.conf) 和 [feature_list.conf](feature_list.conf) 运行：

```bash
./opsfeature parse.conf < input.tsv > output.txt
```

`input.tsv` 由使用方提供，必须与 `parse.conf` 中的 `fea_cols` 顺序和列数一致；`output.txt` 是运行产生的结果。当前仓库没有附带输入样本、预期输出文件或 `examples` 目录。

当前业务配置定义了 273 个输入字段、24 个附加 token，标签字段为 `is_cov`；特征文件包含 923 条有效算子配置。另有 186 条缺少输入来源的算子及下游链路已注释禁用，原始定义保留，未改动输入列顺序或重新编号 slot。具体列名与参数以两个配置文件为准。

`feature_list` 的相对路径以**程序运行时的工作目录**为基准，不以 `parse.conf` 所在目录为基准。

## 代码结构

```text
.
├── main.go                  # CLI：读取样本、重置状态、抽取并输出
├── cli_regression_test.go   # CLI、业务配置与输入/输出基准
├── go.mod / go.sum          # 模块与工具链声明
├── README.md                # 当前功能与使用说明
├── parse.conf               # 业务输入/输出配置
├── feature_list.conf        # 业务特征配置
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
│   ├── runtime_regression_test.go # 缓存、失败回滚和初始化回归
│   ├── memory_regression_test.go  # 内存释放与动态值防护
│   └── performance_test.go  # 抽取基准
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

**配置顺序就是执行顺序**，项目不自动拓扑排序。`InitParser` 会结合输入列检查依赖是否来自原始输入或之前开启 `addcol` 的算子。

抽取采用整条样本成功或失败的策略：算子报错、FID 转换失败或不支持的动态值会立即终止当前样本，撤销本次追加的 FID 并重置中间状态。CLI 不输出该样本，继续处理后续样本，结束时返回退出码 `1`。正常的可选空字段仍按各算子的规则跳过。


## 输入与输出配置

配置由 `feature/parser.go` 加载。当前 [parse.conf](parse.conf) 使用 `feature_list=./feature_list.conf`、`label=is_cov`、`debug=0`，并通过 `fea_cols` 和 `add_token` 定义业务字段。

| 字段 | 必填 | 说明 |
|---|---|---|
| `feature_list` | 是 | 特征配置路径 |
| `fea_cols` | 是 | 逗号分隔的 TSV 列名，顺序与输入一致 |
| `label` | 否 | 逗号分隔的标签列名；未配置时输出 `0` |
| `add_token` | 否 | 追加到输出 `#` 后面的原始字段，必须存在于 `fea_cols` |
| `debug` | 否 | 只接受 `0` 或 `1`；为 `1` 时打印暂不支持的提示，随后按普通模式运行 |
| `filter` | 否 | 只读取和保存，尚未执行过滤 |

配置采用 `key=value` 格式，忽略空行和以 `#` 开头的行；这里不支持行尾注释。以双引号开头的值通过 `strconv.Unquote` 解析。

`$name` 和 `${name}` 引用同一配置文件中的字段，不读取系统环境变量。支持递归展开；未定义引用和循环引用会在初始化时返回错误，结果不依赖 map 遍历顺序。

`fea_cols`、`label`、`add_token` 均拒绝空列名、重复列名及带首尾空格的列名。`label` 和 `add_token` 必须存在于 `fea_cols`；未知配置项和重复 key 会导致初始化失败。

## 特征配置与依赖

每行定义一个算子，字段用分号分隔，支持 `#` 行尾注释。下面摘自当前 `feature_list.conf`：

```ini
slot_id=1;name=data_type;class=Bucket;depend=data_type;args=2,1,dummy,15,16
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

未知字段、重复 key、重复特征名和重复非零基础 `slot_id` 会导致初始化失败；还会检查输出 slot/fake slot 区间冲突、依赖数量、空依赖、多余参数、非有限系数及 padding 参数。

依赖查找优先级：

```text
Input.Global → Input.Score → State.inner
```

原始输入优先于同名中间特征。下游使用 `addcol` 时，上游必须出现在前面，并实际开启 `addcol`。中间特征只有产生了值才会写入 `State.inner`。

当前配置中的一个依赖链为：

```ini
slot_id=0;name=u_recall_30days;class=Direct;depend=u_fea_new;args=3,1,kv_int_float,27;addcol=1;list_cache=1
slot_id=27;name=d_u_recall_30days;class=Bucket;depend=u_recall_30days;args=2,1,abslisan,0,2,3,4,5,6,7,8,10,16;list_cache=1
```

第一行从 `u_fea_new` 的 KV 数据中取 key `27`，只保存中间结果；第二行读取该结果并分桶。`slot_id=0` 不生成 FID，因此第一行配置的 `list_cache` 不会缓存 FID。

初始化会拒绝未知依赖、前向引用和未写出中间值的依赖。当前配置已为 192 个有效的上游算子开启 `addcol`；缺少源字段的链路以 `# disabled: unavailable dependencies ...` 注释保留。恢复这些链路前，需要提供实际输入来源并重新通过初始化校验。

初始化验证的是依赖的定义和顺序，不能保证每条样本都带有非空值。缺失值的运行时行为仍遵循具体算子的规则。

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

以下说明对应 `operator/op_registry.go` 注册的 10 个 class。当前业务特征文件使用其中 7 个：`Direct`、`Bucket`、`Seq`、`Seq_emb`、`Combine`、`Combine_multi`、`Arithmetic`；另外已实现 `Seq_field`、`Bucket_truncate`、`Scale`。

`args` 参数均从下标 `0` 开始。没有特别说明时，`hash_version` 默认为 `0`，版本 1/2 的有效默认系数为 `1`。

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

- `Bucket.abslisan`：处理数值精度和缺失值后取绝对值，返回第一个大于或等于数值的阈值下标；保留历史 `1e-8` 比较容差。超过所有阈值返回阈值数量。阈值必须有限且非递减，重复阈值保留原位置；单阈值表也遵循这一规则。
- `Bucket.dummy`：返回整数在枚举表中的下标，未命中返回枚举数量；`-999` 按 `0` 处理。
- `Bucket_truncate`：按配置顺序将闭区间映射到连续位置，从 `0` 开始，未命中返回 `0`。端点倒置、区间重叠或累计位置超出 `int64` 范围时初始化失败；互不重叠的区间仍保留配置顺序。

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

序列和向量缓存键包含完整的解析参数，按 padding 长度、默认值、字段选择与类型隔离；余弦算子的原始向量不会复用经过不同 padding 的结果。

### Combine / Combine_multi

```text
args=addcol,hash_version,coeff
```

`Combine` 严格要求两个依赖，`Combine_multi` 严格要求三个。对依赖值列表做笛卡尔积，用 `_` 连接并去重。首个参数必须是字面量 `addcol`，依赖可以是原始单值或上游的列表结果。

原始的逗号分隔字符串不会在这里自动展开，应先通过 `Direct.str_dict` 和 `addcol=1` 转成列表。

### Arithmetic

```text
args=operation,hash_version,coeff
```

严格需要两个依赖，分别作为左右输入。

| operation | 行为 |
|---|---|
| `add` | 左值加右值，`-999` 按 `0` 处理 |
| `minus` | 左值减右值，`-999` 按 `0` 处理 |
| `multiply` | 左值乘右值，`-999` 按 `0` 处理 |
| `match` | 两边都为空输出 `2`，相同输出 `1`，否则输出 `0` |
| `div` | 左值除右值，任一值接近 `0` 时输出 `0` |
| `cos` | 两个逗号分隔向量的余弦相似度；空向量或长度不等时不输出，零范数向量输出 `0` |

### Scale

```text
args=hash_version,coeff,mode
```

`mode` 默认且目前仅支持 `log`：先将负数截断为 `0`，再计算 `log(value+1)`。

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

配置初始化失败、输入读取失败、输出写入失败以及任意样本的抽取失败返回 `1`。列数不符、必填标识为空、标签或 token 格式错误仍按行跳过。正常结果写 stdout，诊断信息写 stderr。

## 包接口与状态管理

其他 Go 程序使用内置算子时，除导入 `feature` 包，还需要副作用导入 `github.com/liamlmy/OpsFeature/operator` 完成注册。

| 接口 | 用途 |
|---|---|
| `feature.InitParser(path)` | 读取完整的输入/输出配置和特征配置 |
| `feature.New(io.Reader)` | 从 Reader 加载特征配置 |
| `feature.NewFromText(text)` | 从字符串加载特征配置 |
| `Extractor.Extract(input, state, out)` | 按配置顺序抽取，成功追加到 `out`，失败保留调用前的输出前缀 |
| `Extractor.ValidateInputs(names)` | 根据原始/公共输入列名检查依赖顺序；`InitParser` 自动调用 |
| `feature.NewState()` / `State.Reset()` | 创建样本状态、在样本之间重置状态 |
| `feature.Register()` / `feature.Classes()` | 注册算子、查询已注册名称 |

`Input.Global` 保存公共特征，`Input.Score` 保存当前样本特征。CLI 只填充 `Score`。复用输出切片时传入 `out[:0]`；复用 State 时，调用方必须在不同样本之间调用 `Reset()`。传入 `nil` State 时，`Extract` 自动创建状态，并在返回前回收其数据源。使用 `New` 或 `NewFromText` 的调用方应显式调用 `ValidateInputs`，传入原始和公共特征名。

`State` 持有中间特征、解析结果、FID 缓存、去重集合及排序缓冲，不应被多个 goroutine 并发使用。`Reset()` 清除中间特征和解析/FID 缓存，并将数据源归还对象池；同时清理去重集合和排序缓冲中的引用。

### source.go 与 source_pool.go

- `source.go` 定义数据源类型、解析逻辑与按当前样本缓存的读取入口。
- `source_pool.go` 使用 `sync.Pool` 管理这些类型的申请和回收，并处理容器清理与大对象保留上限。
- 首次读取经 `source()` → `newSource()` → `get*()` 取得对象，解析成功后存入 `State.sources`；同一样本的后续读取可复用解析结果。
- 解析失败或执行 `State.Reset()` 时，通过 `release()` 将对象归还对象池。

整数序列和浮点向量通过 `ParseArrayIntNewInto`、`ParseArrayNewInto` 复用已有切片，并使用流式切分避免完整分割数组。旧的数组工具接口仍保留，空值、坏元素与补齐规则保持兼容。

对象池中的序列容量、KV 条目数以及 State 的缓存/缓冲以 4096 个元素或条目为保留界限，超过时在重置或回收时丢弃。map 和字符串引用会清理。CLI 单独限制跨样本保留的 FID 切片容量为 8192 项，给 4096 项向量的切片增长留出空间；这不是单条样本的输出数量上限。

新增算子需实现 `feature.Operator` 的 `Conf()`、`Init()`、`Emit()`，在 `init()` 中调用 `feature.Register`，并通过 `Base` 的 `Emit*`/`Flush` 方法生成 FID 和中间特征。`feature/api.go` 是供 `operator` 包使用的公开接口。

## 开发检查

```bash
go test ./...
go test -race ./...
go vet ./...
go build -o opsfeature .
```

可用下面的命令检查现有配置和有序依赖能否通过初始化；它不会验证真实业务样本的抽取结果：

```bash
./opsfeature parse.conf < /dev/null
```

回归测试覆盖缓存参数隔离、整行失败回滚、不可比较动态值、业务配置依赖、初始化参数校验、分桶边界、缓冲复用和释放、CLI 错误策略，以及 CityHash 固定结果。测试使用内嵌的合成输入，不额外增加业务配置或示例目录。

## 测量与兼容性说明

在相同机器（Apple M4，darwin/arm64，Go 1.27.1）上先建立基线，再修改序列分配路径。下表为 `-benchtime=100ms -count=3` 的测量范围，使用合成数据，不代表生产吞吐。

| 场景 | 修改前 B/op；allocs/op | 修改后 B/op；allocs/op | 修改前 → 修改后耗时 |
|---|---|---|---|
| 单标量抽取 | 24；1 | 24；1 | 42.6–44.8 → 46.4–51.9 ns |
| 64 项 KV | 24；1 | 24；1 | 6.16–6.29 → 6.19–6.25 μs |
| 32 项整数序列 | 1040；8 | 24；1 | 808–817 → 697–713 ns |
| 4096 维向量抽取 | 约 194156；18 | 251–285；1 | 122–125 → 102–104 μs |
| 16×16 交叉 | 2635–2636；261 | 2635；261 | 11.5–12.1 → 11.5–11.6 μs |
| uint64/float64 CityHash | 0；0 | 0；0 | 均约 2 ns，未消除原本不存在的堆分配 |

CLI 现有的 273 列映射路径测得 46272 B/op、282 allocs/op；1024 项 FID 格式化测得 27264 B/op、1 alloc/op，作为后续优化依据。

安全检查给标量路径增加了少量开销；此次未改写 CLI 双 map 或交叉字符串路径。对应基准也已保留，后续优化应依据测量，不根据源码中的 `make` 直接推断堆分配。

复现命令：

```bash
go test ./... -run '^$' -bench . -benchmem -benchtime=100ms -count=3
go test ./feature -run '^$' -bench 'BenchmarkExtract/vector4096' -benchtime=1s -cpuprofile=/tmp/opsfeature-cpu.out -memprofile=/tmp/opsfeature-memory.out -o /tmp/opsfeature-profile.test
go tool pprof -top /tmp/opsfeature-profile.test /tmp/opsfeature-cpu.out
```

CityHash 底层读取改为 `binary.LittleEndian`，移除 `unsafe` 与本机字节序依赖。固定向量覆盖全部长度分支和非对齐字节切片，结果与修改前的小端实现一致。已通过 linux/ppc64 大端目标的交叉编译，但未在大端机器上实际运行测试。

行为变更需要下游留意：失败样本不再输出部分 FID；缺少来源的 186 条业务链路被禁用；启用中间值后，原先未生成的交叉特征会产生结果；单阈值桶的上界行为已修正，现有 `dummy_preferShopScore` 的递减阈值从 `0,-1` 调整为 `-1,0`；倒置或重叠区间改为启动时报错。

## 当前限制

- `filter` 尚未执行过滤；debug 尚无特征名级别的调试输出。
- 数值依赖解析失败仍可能被当作 `0`；FID 缩放的小数系数和溢出处理仍需完善。
- 部分序列坏元素处理可能改变后续位置，尚未统一空输入与占位规则。
- 交叉特征没有组合数量上限；缓冲保留界限不等于单条样本的资源上限。
- 当前没有真实业务输入和预期输出用于端到端核对；被禁用的链路需要补齐输入来源后才能恢复。

后续工作与验收建议见 [todo.md](todo.md)。
