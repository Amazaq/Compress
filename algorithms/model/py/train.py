import argparse
import os
import joblib
import numpy as np
import pandas as pd
from sklearn.compose import ColumnTransformer
from sklearn.impute import SimpleImputer
from sklearn.metrics import classification_report
from sklearn.model_selection import train_test_split
from sklearn.pipeline import Pipeline
from sklearn.preprocessing import StandardScaler
from sklearn.ensemble import RandomForestClassifier
from sklearn.multioutput import MultiOutputClassifier


def load_data(features_path: str, labels_path: str):
	"""
	加载训练数据
	特征: 61维 (基本统计12 + 数值特性6 + 差分统计16 + 时序3 + 游程4 + 位级5 + 分布3 + 自相关10 + 周期2)
	标签: 4个压缩参数 (ranged, scale, del, algo)
	"""
	X = pd.read_csv(features_path, header=None)
	Y = pd.read_csv(labels_path, header=None)
	# Ensure shapes align
	if X.shape[0] != Y.shape[0]:
		raise ValueError(f"Features rows {X.shape[0]} and labels rows {Y.shape[0]} mismatch")
	return X.values.astype(np.float64), Y.values.astype(np.int64)


def build_pipeline(random_state: int = 42) -> Pipeline:
	# Numeric pipeline with imputation and scaling
	numeric_transformer = Pipeline(steps=[
		("imputer", SimpleImputer(strategy="median")),
		("scaler", StandardScaler(with_mean=True, with_std=True)),
	])

	# Full pipeline (all columns numeric)
	preprocess = ColumnTransformer(
		transformers=[
			("num", numeric_transformer, slice(0, None)),
		]
	)

	base_clf = RandomForestClassifier(
		n_estimators=200,
		max_depth=15,
		min_samples_split=5,
		min_samples_leaf=2,
		max_features="sqrt",
		class_weight="balanced",  # 自动平衡类别权重
		random_state=random_state,
		n_jobs=-1,
	)
	multi_clf = MultiOutputClassifier(base_clf, n_jobs=-1)

	pipe = Pipeline(steps=[
		("preprocess", preprocess),
		("clf", multi_clf),
	])
	return pipe


def main():
	parser = argparse.ArgumentParser(description="Train multi-output classifier for compression action selection")
	parser.add_argument("--features", default=os.path.join("..", "..", "..", "dataset", "train_features.csv"))
	parser.add_argument("--labels", default=os.path.join("..", "..", "..", "dataset", "train_labels.csv"))
	parser.add_argument("--out", default=os.path.join(os.path.dirname(__file__), "model.joblib"))
	parser.add_argument("--test_size", type=float, default=0.15)
	parser.add_argument("--random_state", type=int, default=42)
	args = parser.parse_args()

	X, Y = load_data(args.features, args.labels)
	
	# 检查标签分布 - Y是4维输出，每列代表一个任务
	print("标签分布:")
	print(f"标签形状: {Y.shape}")
	for i in range(Y.shape[1]):
		unique, counts = np.unique(Y[:, i], return_counts=True)
		print(f"任务 {i} 分布: {dict(zip(unique, counts))}")
	
	# 对于多输出分类，不使用分层抽样（sklearn的stratify不支持多输出）
	stratify = None
	print("多输出分类不使用分层抽样")
	
	X_train, X_test, Y_train, Y_test = train_test_split(
		X, Y, test_size=args.test_size, random_state=args.random_state, stratify=stratify
	)

	pipe = build_pipeline(random_state=args.random_state)
	pipe.fit(X_train, Y_train)

	Y_pred = pipe.predict(X_test)
	print("Evaluation on holdout set:")
	
	# 获取任务名称
	task_names = ["ranged", "scale", "delta", "algorithm"]
	
	for i in range(Y.shape[1]):
		print(f"\nTask {i} ({task_names[i]}):")
		# 获取该任务的所有类别
		all_classes = np.unique(np.concatenate([Y_train[:, i], Y_test[:, i]]))
		print(f"训练集类别: {np.unique(Y_train[:, i])}")
		print(f"测试集类别: {np.unique(Y_test[:, i])}")
		print(f"预测类别: {np.unique(Y_pred[:, i])}")
		
		# 使用zero_division参数避免警告
		print(classification_report(
			Y_test[:, i], Y_pred[:, i], 
			labels=all_classes,
			zero_division=0,
			digits=4
		))

	os.makedirs(os.path.dirname(args.out), exist_ok=True)
	joblib.dump(pipe, args.out)
	print(f"Saved model to {args.out}")


if __name__ == "__main__":
	main()
