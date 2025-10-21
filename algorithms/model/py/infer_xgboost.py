import argparse
import os
import sys
import pickle
import numpy as np
import pandas as pd
import xgboost as xgb


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


def load_xgboost_models(model_path):
	"""加载 XGBoost 模型"""
	if not os.path.exists(model_path):
		raise FileNotFoundError(f"模型文件未找到: {model_path}")
	
	with open(model_path, 'rb') as f:
		models = pickle.load(f)
	
	return models


def predict_params(models, features):
	"""使用模型预测四个参数"""
	predictions = []
	
	# 训练时的列顺序是 [ranged(0), scale(1), del(2), algo(3)] - 与Go期望一致
	# models[0]=ranged, models[1]=scale, models[2]=del, models[3]=algo
	for i, model in enumerate(models):
		pred = model.predict(features)
		# 四舍五入到最近的整数
		pred_int = int(np.round(pred[0]))
		predictions.append(pred_int)
	
	# predictions = [ranged, scale, del, algo]
	# 需要限制到合法范围
	ranged_param = np.clip(predictions[0], 0, 2)
	scale_param = np.clip(predictions[1], 0, 2)
	del_param = np.clip(predictions[2], 0, 2)
	algo_param = np.clip(predictions[3], 0, 4)
	
	# 返回顺序: [ranged, scale, del, algo]
	return [ranged_param, scale_param, del_param, algo_param]


def main():
	parser = argparse.ArgumentParser(
		description="使用 XGBoost 模型进行推理: 读取特征文件并输出预测的压缩参数"
	)
	parser.add_argument(
		"--model", 
		default=os.path.join(os.path.dirname(__file__), "xgboost_models_all.pkl"),
		help="XGBoost 模型文件路径 (默认: xgboost_models_all.pkl)"
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
	args = parser.parse_args()

	# 加载模型
	if args.verbose:
		print(f"正在加载模型: {args.model}")
	
	models = load_xgboost_models(args.model)
	# 输出顺序: [ranged, scale, del, algo] (与 Go 代码期望一致)
	param_names = ['ranged', 'scale', 'del', 'algo']
	
	if args.verbose:
		print(f"成功加载 {len(models)} 个模型")

	# 读取特征
	if args.verbose:
		print(f"正在读取特征文件: {args.features}")
	
	if args.features.endswith('.json'):
		# JSON 格式：使用 flatten_stats 处理
		import json
		with open(args.features, 'r') as f:
			stats = json.load(f)
		X = flatten_stats(stats)[None, :]
	else:
		# CSV 格式：使用 pandas 读取
		X = pd.read_csv(args.features, header=None).values.astype(np.float64)
	
	if args.verbose:
		print(f"特征形状: {X.shape}")

	# 预测
	predictions = predict_params(models, X)
	
	# 输出结果为 JSON 数组
	import json
	print(json.dumps(predictions))
	
	if args.verbose:
		print("\n预测结果:")
		for name, pred in zip(param_names, predictions):
			print(f"  {name}: {pred}")

	# 保存到文件（如果指定）
	if args.out:
		result_df = pd.DataFrame([predictions], columns=param_names)
		result_df.to_csv(args.out, header=True, index=False)
		if args.verbose:
			print(f"\n✅ 预测结果已保存到: {args.out}")


if __name__ == "__main__":
	main()
