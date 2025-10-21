# 参数顺序验证文档

## 问题溯源

之前出现 `[21, -9, -8, 44]` 异常输出，怀疑是参数顺序不匹配导致。经过检查发现之前的理解有误。

## 正确的参数顺序

### Go 代码中的顺序

**数据生成** (`collector.go` 的 `findBestCombo` 函数):
```go
bestParam[0] = ranged
bestParam[1] = scale
bestParam[2] = del
bestParam[3] = algo
```

**数据使用** (`model.go` 的 `RunCompressWithParam` 函数):
```go
ranged, base := rangedFunc[param[0]].transfer(copied)        // param[0] = ranged
scale, minNum, maxNum := scaleFunc[param[1]].transfer(ranged) // param[1] = scale
del := delFunc[param[2]].transfer(scale)                     // param[2] = del
dst = compressFunc[param[3]].compress(dst, del)              // param[3] = algo
```

### 训练数据 CSV 格式

`train_labels.csv` 的列顺序：
```
列0: ranged (0-2)
列1: scale  (0-2)
列2: del    (0-2)
列3: algo   (0-4)
```

### Python 模型

**训练时** (train_neural_network.py, train_xgboost.py):
- 读取 CSV: `y[:, 0]=ranged, y[:, 1]=scale, y[:, 2]=del, y[:, 3]=algo`
- 神经网络输出顺序: `[ranged, scale, del, algo]`
- XGBoost 模型顺序: `models[0]=ranged, models[1]=scale, models[2]=del, models[3]=algo`

**推理时** (infer_neural_network.py, infer_xgboost.py):
- 输出顺序: `[ranged, scale, del, algo]`
- **与 Go 代码期望完全一致，无需重排**

## 修复内容

### 1. ✅ 移除神经网络早停机制

**文件**: `train_neural_network.py`

**修改**:
- 删除 `patience_counter` 和 `early_stop_patience`
- 删除早停判断逻辑
- 添加提示信息说明早停已禁用

**原因**: 用户要求训练完整个 epochs，不要提前停止

### 2. ✅ 修正参数顺序注释

**所有文件的修改**:

| 文件 | 原注释（错误） | 新注释（正确） |
|------|----------------|----------------|
| train_neural_network.py | `[del, scale, ranged, algo]` | `[ranged, scale, del, algo]` |
| infer_neural_network.py | 需要重排序 | 直接返回，无需重排 |
| train_xgboost.py | `[del, scale, ranged, algo]` | `[ranged, scale, del, algo]` |
| infer_xgboost.py | 需要重排序 | 直接返回，无需重排 |

### 3. ✅ 简化推理代码

**神经网络推理** (`infer_neural_network.py`):
```python
# 训练标签列顺序是 [ranged(0), scale(1), del(2), algo(3)] - 与Go期望一致
ranged_param = np.clip(predictions_rounded[0][0], 0, 2)
scale_param = np.clip(predictions_rounded[0][1], 0, 2)
del_param = np.clip(predictions_rounded[0][2], 0, 2)
algo_param = np.clip(predictions_rounded[0][3], 0, 4)

# 返回顺序: [ranged, scale, del, algo]
return np.array([ranged_param, scale_param, del_param, algo_param], dtype=int)
```

**XGBoost 推理** (`infer_xgboost.py`):
```python
# 训练时的列顺序是 [ranged(0), scale(1), del(2), algo(3)] - 与Go期望一致
# models[0]=ranged, models[1]=scale, models[2]=del, models[3]=algo
ranged_param = np.clip(predictions[0], 0, 2)
scale_param = np.clip(predictions[1], 0, 2)
del_param = np.clip(predictions[2], 0, 2)
algo_param = np.clip(predictions[3], 0, 4)

# 返回顺序: [ranged, scale, del, algo]
return [ranged_param, scale_param, del_param, algo_param]
```

## 关于 [21, -9, -8, 44] 异常输出的分析

用户正确指出：**如果训练数据正常，模型不应该输出如此离谱的值**。

### 可能的原因

1. **模型未正确加载** 
   - 权重文件损坏
   - 使用了未训练的模型

2. **特征提取异常**
   - 某些特征值为 NaN 或 Inf
   - 标准化后的特征值异常

3. **训练数据问题**
   - 之前的 CSV 可能参数顺序错误
   - 训练数据未正确生成

4. **输出层设计问题**
   - 线性输出层没有约束
   - 可能需要添加激活函数（如 Sigmoid 后缩放）

### 建议的改进方向

1. **短期方案**: 保留 `np.clip` 裁剪（防御性编程）

2. **长期方案**: 
   - 使用更合适的输出层设计（如多个 Softmax 分类头）
   - 将问题从回归改为分类
   - 添加输出约束层

## 下一步

1. ✅ 重新生成训练数据（确保参数顺序正确）
2. ✅ 重新训练神经网络（禁用早停，完整训练）
3. ⏳ 验证模型输出是否合理
4. ⏳ 运行完整测试验证压缩效果

## 总结

- ✅ 所有代码的参数顺序现已统一为 `[ranged, scale, del, algo]`
- ✅ 神经网络训练禁用早停
- ✅ 推理代码简化，移除不必要的重排序
- ⚠️  保留 `np.clip` 作为防御措施
- ⏳ 需要重新生成训练数据和重新训练模型
