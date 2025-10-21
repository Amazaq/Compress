# 模型参数顺序修复文档

## 问题描述

神经网络模型输出了异常值 `[21, -9, -8, 44]`，导致数组越界错误。经过排查，发现了两个关键问题：

### 问题1: 参数顺序不匹配
- **训练数据 CSV 列顺序**: `[del(列0), scale(列1), ranged(列2), algo(列3)]`
- **Go 代码期望顺序**: `[ranged, scale, del, algo]`
- **模型输出顺序**: `[del, scale, ranged, algo]` (与训练时一致)

Go 代码在 `model.go` 的 `RunCompressWithParam` 函数中：
```go
ranged, base := rangedFunc[param[0]].transfer(copied)        // 期望 param[0] = ranged
scale, minNum, maxNum := scaleFunc[param[1]].transfer(ranged) // 期望 param[1] = scale  
del := delFunc[param[2]].transfer(scale)                     // 期望 param[2] = del
// ...
dst = compressFunc[param[3]].compress(dst, del)              // 期望 param[3] = algo
```

### 问题2: 缺少范围限制
模型输出层是线性的，没有激活函数限制，可以输出任意值。预测后必须裁剪到合法范围：
- `ranged`: [0, 2]
- `scale`: [0, 2]
- `del`: [0, 2]
- `algo`: [0, 4]

## 修复方案

### 1. 神经网络推理脚本 (infer_neural_network.py)

**位置**: `algorithms/model/py/infer_neural_network.py`

**修改**: `predict_params` 函数

```python
def predict_params(model, scaler, features, device):
    """使用模型预测压缩参数"""
    # 标准化特征
    features_scaled = scaler.transform(features)
    
    # 转换为张量
    features_tensor = torch.FloatTensor(features_scaled).to(device)
    
    # 预测
    with torch.no_grad():
        predictions = model(features_tensor).cpu().numpy()
    
    # 四舍五入到最近的整数
    predictions_rounded = np.round(predictions).astype(int)
    
    # 训练标签列顺序是 [del, scale, ranged, algo]
    # 裁剪到合法范围
    del_param = np.clip(predictions_rounded[0][0], 0, 2)
    scale_param = np.clip(predictions_rounded[0][1], 0, 2)
    ranged_param = np.clip(predictions_rounded[0][2], 0, 2)
    algo_param = np.clip(predictions_rounded[0][3], 0, 4)
    
    # Go 代码期望的顺序是 [ranged, scale, del, algo]
    # 需要重新排列: [del, scale, ranged, algo] -> [ranged, scale, del, algo]
    return np.array([ranged_param, scale_param, del_param, algo_param], dtype=int)
```

### 2. XGBoost 推理脚本 (infer_xgboost.py)

**位置**: `algorithms/model/py/infer_xgboost.py`

**修改**: `predict_params` 函数

```python
def predict_params(models, features):
    """使用模型预测四个参数"""
    predictions = []
    
    # 训练时的列顺序是 [del(0), scale(1), ranged(2), algo(3)]
    # models[0]=del, models[1]=scale, models[2]=ranged, models[3]=algo
    for i, model in enumerate(models):
        pred = model.predict(features)
        # 四舍五入到最近的整数
        pred_int = int(np.round(pred[0]))
        predictions.append(pred_int)
    
    # predictions = [del, scale, ranged, algo]
    # 需要限制到合法范围
    del_param = np.clip(predictions[0], 0, 2)
    scale_param = np.clip(predictions[1], 0, 2)
    ranged_param = np.clip(predictions[2], 0, 2)
    algo_param = np.clip(predictions[3], 0, 4)
    
    # Go 代码期望的顺序是 [ranged, scale, del, algo]
    # 重排: [del, scale, ranged, algo] -> [ranged, scale, del, algo]
    return [ranged_param, scale_param, del_param, algo_param]
```

### 3. 训练脚本注释更新

**train_neural_network.py**:
```python
"""
加载训练数据
特征: 61维 (基本统计12 + 数值特性6 + 差分统计16 + 时序3 + 游程4 + 位级5 + 分布3 + 自相关10 + 周期2)
标签: 4个压缩参数，CSV列顺序为 [del(0), scale(1), ranged(2), algo(3)]
"""
```

**train_xgboost.py**:
```python
# 训练四个模型（对应四个参数）
# 注意: CSV 列顺序是 [del(0), scale(1), ranged(2), algo(3)]
# 所以 models[0]=del, models[1]=scale, models[2]=ranged, models[3]=algo
models = []
param_names = ['del', 'scale', 'ranged', 'algo']  # 实际训练顺序
```

## 参数范围说明

| 参数 | 范围 | 说明 |
|------|------|------|
| ranged | 0-2 | 预处理类型 (null/RangedArr/MinMaxRangedArr) |
| scale | 0-2 | 缩放类型 (null/delta/delta-of-delta) |
| del | 0-2 | 差分类型 (null/delta/delta-of-delta) |
| algo | 0-4 | 压缩算法 (huffman/elf/chimp128/fpc/zstd) |

## 验证方法

1. 运行测试确认不再有数组越界错误
2. 检查模型输出的参数都在合法范围内
3. 对比模型预测的参数与最优参数的差异

## 数据特征计算验证

已检查 `common/dataStats.go` 中的关键特征计算函数：

### ✅ 已验证正确的计算

1. **百分位数计算** (`percentile` 函数，行 899-916)
   - 使用线性插值方法
   - 处理边界情况
   - 公式正确

2. **自相关系数** (`computeAutoCorrelation` 函数，行 812-867)
   - 计算 lag 1-10 的自相关
   - 正确处理 NaN 和 Inf 值
   - 归一化方差

3. **信息熵** (`computeEntropy` 函数，行 758-809)
   - 使用 100 bins 直方图
   - Shannon 熵公式: $H = -\sum p_i \log_2(p_i)$
   - 处理边界情况

4. **基本统计量** (mean, std, variance, skewness, kurtosis)
   - 在线计算，避免多次扫描
   - 正确处理 NaN 和 Inf

5. **差分统计** (DiffStats, SecondDiffStats)
   - 一阶和二阶差分
   - 完整的统计指标

6. **位级统计** (BitLevelStats)
   - IEEE 754 浮点数位操作
   - 尾数熵、指数范围等

### ⚠️ 潜在问题点

1. **percentile_95 和 percentile_5 在某些数据中为 0**
   - 这可能是数据特性而非计算错误
   - 需要检查具体数据分布

2. **periodicity 为 0**
   - 可能数据本身不具有周期性
   - 或检测算法不够敏感

## 总结

所有模型推理脚本已统一：
- ✅ 输出顺序统一为 `[ranged, scale, del, algo]`
- ✅ 添加参数范围裁剪 (防止越界)
- ✅ 更新注释说明训练和推理的顺序映射
- ✅ 验证数据特征计算正确性

下一步：重新运行测试验证修复效果。
