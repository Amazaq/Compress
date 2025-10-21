"""
快速测试模型推理修复
"""
import json
import numpy as np

# 模拟一个简单的特征输入
test_features = {
    "min": 1.0, "max": 100.0, "mean": 50.0, "median": 50.0,
    "std_dev": 10.0, "variance": 100.0, "skewness": 0.0, "kurtosis": 0.0,
    "range": 99.0, "iqr": 20.0, "q1": 40.0, "q3": 60.0,
    "unique_count": 100, "unique_ratio": 1.0,
    "zero_count": 0, "zero_ratio": 0.0,
    "integer_count": 100, "integer_ratio": 1.0,
    "diff_stats": {
        "min": -10.0, "max": 10.0, "mean": 0.0,
        "std_dev": 5.0, "range": 20.0,
        "zero_ratio": 0.1, "unique_count": 50, "unique_ratio": 0.5
    },
    "second_diff_stats": {
        "min": -5.0, "max": 5.0, "mean": 0.0,
        "std_dev": 2.0, "range": 10.0,
        "zero_ratio": 0.2, "unique_count": 30, "unique_ratio": 0.3
    },
    "monotonicity": 0.8, "smoothness": 2.0, "change_points": 5,
    "run_length": {
        "max_run_length": 3, "avg_run_length": 1.5,
        "run_count": 50, "constant_run_ratio": 0.1
    },
    "bit_stats": {
        "avg_set_bits": 30.0, "sign_changes": 10,
        "mantissa_entropy": 10.0, "exponent_range": 5,
        "common_exponent": 10
    },
    "entropy": 0.9, "percentile_95": 95.0, "percentile_5": 5.0,
    "auto_correlation": [0.9, 0.8, 0.7, 0.6, 0.5, 0.4, 0.3, 0.2, 0.1, 0.0],
    "periodicity": 0, "periodic_score": 0.0
}

# 保存到临时文件
with open('test_features.json', 'w') as f:
    json.dump(test_features, f)

print("✅ 测试特征文件已创建: test_features.json")
print("\n测试命令:")
print("1. 神经网络: python algorithms\\model\\py\\infer_neural_network.py --features test_features.json --verbose")
print("2. XGBoost: python algorithms\\model\\py\\infer_xgboost.py --features test_features.json --verbose")
print("\n期望输出: [ranged, scale, del, algo] 格式，每个值在合法范围内")
