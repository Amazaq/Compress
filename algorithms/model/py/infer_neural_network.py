import argparse
import os
import sys
import pickle
import numpy as np
import pandas as pd
import torch
import torch.nn as nn
import json


class CompressionNet(nn.Module):
    """压缩参数预测神经网络 (需要与训练时保持一致)"""
    def __init__(self, input_dim, hidden_dims=[256, 128, 64, 32], dropout=0.3):
        super(CompressionNet, self).__init__()
        
        layers = []
        prev_dim = input_dim
        
        # 构建隐藏层
        for i, hidden_dim in enumerate(hidden_dims):
            layers.append(nn.Linear(prev_dim, hidden_dim))
            layers.append(nn.BatchNorm1d(hidden_dim))
            layers.append(nn.ReLU())
            if i < len(hidden_dims) - 1:
                layers.append(nn.Dropout(dropout))
            prev_dim = hidden_dim
        
        # 输出层
        layers.append(nn.Linear(prev_dim, 4))
        
        self.network = nn.Sequential(*layers)
    
    def forward(self, x):
        return self.network(x)


def flatten_stats(s: dict) -> np.ndarray:
    """
    将统计字典转换为特征向量，与训练时的格式保持一致
    返回61维特征向量: 基本统计12 + 数值特性6 + 差分统计16 + 时序3 + 游程4 + 位级5 + 分布3 + 自相关10 + 周期2
    """
    v = []
    # 基本统计（按 Go 的字段顺序）
    v.extend([
        s.get("min", 0.0), s.get("max", 0.0), s.get("mean", 0.0), s.get("median", 0.0),
        s.get("std_dev", 0.0), s.get("variance", 0.0), s.get("skewness", 0.0), s.get("kurtosis", 0.0),
        s.get("range", 0.0), s.get("iqr", 0.0), s.get("q1", 0.0), s.get("q3", 0.0),
    ])
    # 数值特性
    v.extend([
        float(s.get("unique_count", 0) or 0), s.get("unique_ratio", 0.0) or 0.0,
        float(s.get("zero_count", 0) or 0), s.get("zero_ratio", 0.0) or 0.0,
        float(s.get("integer_count", 0) or 0), s.get("integer_ratio", 0.0) or 0.0,
    ])
    # 一阶差分
    diff = s.get("diff_stats") or None
    if diff is not None:
        v.extend([
            diff.get("min", 0.0), diff.get("max", 0.0), diff.get("mean", 0.0),
            diff.get("std_dev", 0.0), diff.get("range", 0.0),
            diff.get("zero_ratio", 0.0), float(diff.get("unique_count", 0) or 0), diff.get("unique_ratio", 0.0),
        ])
    else:
        v.extend([0.0] * 8)
    # 二阶差分
    sec = s.get("second_diff_stats") or None
    if sec is not None:
        v.extend([
            sec.get("min", 0.0), sec.get("max", 0.0), sec.get("mean", 0.0),
            sec.get("std_dev", 0.0), sec.get("range", 0.0),
            sec.get("zero_ratio", 0.0), float(sec.get("unique_count", 0) or 0), sec.get("unique_ratio", 0.0),
        ])
    else:
        v.extend([0.0] * 8)
    # 时序特征
    v.extend([
        s.get("monotonicity", 0.0) or 0.0, s.get("smoothness", 0.0) or 0.0, float(s.get("change_points", 0) or 0),
    ])
    # 游程
    runlen = s.get("run_length") or None
    if runlen is not None:
        v.extend([
            float(runlen.get("max_run_length", 0) or 0), runlen.get("avg_run_length", 0.0) or 0.0,
            float(runlen.get("run_count", 0) or 0), runlen.get("constant_run_ratio", 0.0) or 0.0,
        ])
    else:
        v.extend([0.0] * 4)
    # 位级
    bits = s.get("bit_stats") or None
    if bits is not None:
        v.extend([
            bits.get("avg_set_bits", 0.0) or 0.0, float(bits.get("sign_changes", 0) or 0),
            bits.get("mantissa_entropy", 0.0) or 0.0, float(bits.get("exponent_range", 0) or 0),
            float(bits.get("common_exponent", 0) or 0),
        ])
    else:
        v.extend([0.0] * 5)
    # 分布
    v.extend([
        s.get("entropy", 0.0) or 0.0, s.get("percentile_95", 0.0) or 0.0, s.get("percentile_5", 0.0) or 0.0,
    ])
    # 自相关
    auto = s.get("auto_correlation") or []
    v.extend([float(x) for x in auto])
    # 周期
    v.extend([
        float(s.get("periodicity", 0) or 0), s.get("periodic_score", 0.0) or 0.0,
    ])
    return np.array(v, dtype=np.float64)


def load_model(model_path, config_path, scaler_path, device):
    """加载训练好的模型和标准化器"""
    
    # 方法1: 尝试加载完整模型
    full_model_path = model_path.replace('.pth', '_full.pth')
    if os.path.exists(full_model_path):
        model = torch.load(full_model_path, map_location=device, weights_only=False)
        model.eval()
    else:
        # 方法2: 加载模型配置和权重
        if not os.path.exists(config_path):
            raise FileNotFoundError(f"模型配置文件未找到: {config_path}")
        
        with open(config_path, 'rb') as f:
            config = pickle.load(f)
        
        model = CompressionNet(
            input_dim=config['input_dim'],
            hidden_dims=config['hidden_dims'],
            dropout=config['dropout']
        ).to(device)
        
        if not os.path.exists(model_path):
            raise FileNotFoundError(f"模型文件未找到: {model_path}")
        
        model.load_state_dict(torch.load(model_path, map_location=device, weights_only=True))
        model.eval()
    
    # 加载标准化器
    if not os.path.exists(scaler_path):
        raise FileNotFoundError(f"标准化器文件未找到: {scaler_path}")
    
    with open(scaler_path, 'rb') as f:
        scaler = pickle.load(f)
    
    return model, scaler


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
    
    # 训练标签列顺序是 [ranged(0), scale(1), del(2), algo(3)] - 与Go期望一致
    # 裁剪到合法范围
    ranged_param = np.clip(predictions_rounded[0][0], 0, 2)
    scale_param = np.clip(predictions_rounded[0][1], 0, 2)
    del_param = np.clip(predictions_rounded[0][2], 0, 2)
    algo_param = np.clip(predictions_rounded[0][3], 0, 4)
    
    # 返回顺序: [ranged, scale, del, algo]
    return np.array([ranged_param, scale_param, del_param, algo_param], dtype=int)


def main():
    parser = argparse.ArgumentParser(
        description="使用神经网络模型进行推理: 读取特征文件并输出预测的压缩参数"
    )
    parser.add_argument(
        "--model", 
        default=os.path.join(os.path.dirname(__file__), "neural_network_model.pth"),
        help="模型权重文件路径 (默认: neural_network_model.pth)"
    )
    parser.add_argument(
        "--config",
        default=os.path.join(os.path.dirname(__file__), "neural_network_config.pkl"),
        help="模型配置文件路径 (默认: neural_network_config.pkl)"
    )
    parser.add_argument(
        "--scaler",
        default=os.path.join(os.path.dirname(__file__), "neural_network_scaler.pkl"),
        help="标准化器文件路径 (默认: neural_network_scaler.pkl)"
    )
    parser.add_argument(
        "--features", 
        required=True, 
        help="特征文件路径 (JSON 或 CSV 格式)"
    )
    parser.add_argument(
        "--out", 
        default=None, 
        help="可选: 输出 CSV 文件保存预测结果"
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="显示详细信息"
    )
    parser.add_argument(
        "--cpu",
        action="store_true",
        help="强制使用 CPU (即使有 GPU 可用)"
    )
    args = parser.parse_args()

    # 设置设备
    if args.cpu:
        device = torch.device('cpu')
    else:
        device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
    
    if args.verbose:
        print(f"使用设备: {device}")

    # 加载模型
    if args.verbose:
        print(f"正在加载模型...")
        print(f"  模型: {args.model}")
        print(f"  配置: {args.config}")
        print(f"  标准化器: {args.scaler}")
    
    try:
        model, scaler = load_model(args.model, args.config, args.scaler, device)
        if args.verbose:
            print(f"✅ 模型加载成功")
    except Exception as e:
        print(f"❌ 模型加载失败: {e}", file=sys.stderr)
        sys.exit(1)

    # 读取特征
    if args.verbose:
        print(f"\n正在读取特征文件: {args.features}")
    
    try:
        if args.features.endswith('.json'):
            # JSON 格式：使用 flatten_stats 处理
            with open(args.features, 'r') as f:
                stats = json.load(f)
            X = flatten_stats(stats)[None, :]
        else:
            # CSV 格式：使用 pandas 读取
            X = pd.read_csv(args.features, header=None).values.astype(np.float64)
        
        if args.verbose:
            print(f"特征形状: {X.shape}")
    except Exception as e:
        print(f"❌ 特征文件读取失败: {e}", file=sys.stderr)
        sys.exit(1)

    # 预测
    try:
        predictions = predict_params(model, scaler, X, device)
        
        # 输出结果为 JSON 数组
        result = predictions.tolist()
        print(json.dumps(result))
        
        param_names = ['ranged', 'scale', 'del', 'algo']
        
        if args.verbose:
            print("\n预测结果:")
            for name, pred in zip(param_names, result):
                print(f"  {name}: {pred}")

        # 保存到文件（如果指定）
        if args.out:
            result_df = pd.DataFrame([result], columns=param_names)
            result_df.to_csv(args.out, header=True, index=False)
            if args.verbose:
                print(f"\n✅ 预测结果已保存到: {args.out}")
    
    except Exception as e:
        print(f"❌ 预测失败: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
