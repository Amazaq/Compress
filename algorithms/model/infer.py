import json
import os
import sys
import joblib
import numpy as np


def flatten_stats(s: dict) -> np.ndarray:
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
	fallback = [0, 0, 0]
	try:
		if len(sys.argv) < 2:
			print(json.dumps(fallback))
			return
		features_path = sys.argv[1]
		with open(features_path, "r", encoding="utf-8") as f:
			stats = json.load(f)
		x = flatten_stats(stats)[None, :]
		model_path = os.path.join(os.path.dirname(__file__), "py", "model.joblib")
		if not os.path.exists(model_path):
			print(json.dumps(fallback))
			return
		pipe = joblib.load(model_path)
		pred = pipe.predict(x)
		res = [int(pred[0, 0]), int(pred[0, 1]), int(pred[0, 2])]
		print(json.dumps(res))
	except Exception:
		print(json.dumps(fallback))


if __name__ == "__main__":
	main()
