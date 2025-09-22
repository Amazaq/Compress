# 压缩算法选择模型

这个目录包含用于训练和推理压缩算法选择模型的Python代码，使用先进的集成学习方法。

## 文件说明

- `train.py`: 模型训练脚本（集成学习：XGBoost + LightGBM + 神经网络 + 随机森林）
- `inference.py`: 模型推理脚本
- `test.py`: 测试脚本，验证模型功能
- `requirements.txt`: Python依赖包列表
- `compression_model.pkl`: 训练后保存的集成模型文件（训练后生成）
- `scaler.pkl`: 特征标准化器文件（训练后生成）

## 安装依赖

```bash
pip install -r requirements.txt
```

主要依赖：
- pandas: 数据处理
- numpy: 数值计算
- scikit-learn: 机器学习算法和神经网络
- joblib: 模型序列化
- xgboost: 梯度提升算法
- lightgbm: 轻量级梯度提升算法

## 使用方法

### 1. 训练模型

```bash
python train.py
```

这将：
- 加载 `../../dataset/train_features.csv` 和 `../../dataset/train_labels.csv`
- 训练集成学习模型（XGBoost + LightGBM + 神经网络 + 随机森林）
- 评估各个子模型和集成模型的性能
- 保存集成模型到 `compression_model.pkl`
- 保存特征标准化器到 `scaler.pkl`

### 2. 推理预测

```bash
python inference.py '[1000.0, 0.509117, 21.028437, ...]'
```

输入：
- 一个JSON格式的特征数组（63个特征值）

输出：
- 预测的压缩算法向量（JSON格式）
- 预测置信度（输出到stderr）

### 3. 测试模型

```bash
python test.py
```

这将测试推理功能和JSON接口。

## Go语言调用示例

在Go代码中，你可以这样调用推理脚本：

```go
package main

import (
    "encoding/json"
    "fmt"
    "os/exec"
)

func predictCompression(features []float64) ([]int, error) {
    // 将特征转换为JSON字符串
    featuresJSON, err := json.Marshal(features)
    if err != nil {
        return nil, err
    }
    
    // 调用Python推理脚本
    cmd := exec.Command("python", "algorithms/model/py/inference.py", string(featuresJSON))
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    
    // 解析结果
    var result []int
    err = json.Unmarshal(output, &result)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}
```

## 模型说明

### 集成学习架构
- **XGBoost**: 梯度提升决策树，擅长处理结构化数据
- **LightGBM**: 轻量级梯度提升，训练速度快，内存占用少
- **神经网络**: 多层感知机（MLP），能够学习复杂的非线性关系
- **随机森林**: 集成决策树，提供稳定的基线性能
- **投票分类器**: 结合所有子模型的预测结果

### 数据处理
- 特征标准化：使用StandardScaler对输入特征进行标准化
- 多标签分类：每个样本可以同时选择多个压缩算法

### 性能评估
- 准确率（Accuracy）
- F1分数（F1-score）
- 各子模型独立评估
- 集成模型综合评估

## 数据格式

### 输入特征（train_features.csv）
每行包含63个特征值，表示数据的统计特性。

### 输出标签（train_labels.csv）
每行包含3个值，表示要使用的压缩算法组合：
- 第1列：算法1的选择（0或1）
- 第2列：算法2的选择（0或1）
- 第3列：算法3的选择（0或1）

## 模型文件

训练完成后会生成两个文件：
- `compression_model.pkl`: 集成学习模型
- `scaler.pkl`: 特征标准化器

这两个文件都是推理时必需的。