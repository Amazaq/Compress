import argparse
import os
import sys
import joblib
import numpy as np
import pandas as pd


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


def main():
	parser = argparse.ArgumentParser(description="Run inference: read feature file and output predicted operation vector(s)")
	parser.add_argument("--model", default=os.path.join(os.path.dirname(__file__), "model.joblib"))
	parser.add_argument("--features", required=True, help="CSV file with features; each row is one sample")
	parser.add_argument("--out", default=None, help="Optional output CSV to write predictions")
	args = parser.parse_args()

	if not os.path.exists(args.model):
		raise FileNotFoundError(f"Model file not found: {args.model}. Train it first.")

	pipe = joblib.load(args.model)

	# 检查文件格式并相应处理
	if args.features.endswith('.json'):
		# JSON 格式：直接使用 flatten_stats 处理
		import json
		with open(args.features, 'r') as f:
			stats = json.load(f)
		X = flatten_stats(stats)[None, :]
	else:
		# CSV 格式：使用 pandas 读取
		X = pd.read_csv(args.features, header=None).values.astype(np.float64)
	
	pred = pipe.predict(X)

	# Print to stdout as JSON array
	import json
	result = [int(pred[0, 0]), int(pred[0, 1]), int(pred[0, 2]), int(pred[0, 3])]
	print(json.dumps(result))

	if args.out:
		pd.DataFrame(pred.astype(int)).to_csv(args.out, header=False, index=False)
		print(f"Saved predictions to {args.out}")


if __name__ == "__main__":
	main()
